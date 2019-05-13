package server

import (
	"bytes"
	"fmt"
	"net"

	"github.com/mit-dci/go-bverify/crypto/fastsha256"
	"github.com/mit-dci/go-bverify/mpt"
	"github.com/mit-dci/go-bverify/wire"
)

// Interface to be able to mock the LogProcessor for the
// test cases
type LogProcessor interface {
	Process()
	SendProofs(delta *mpt.DeltaMPT) error
	ProcessMessage(t wire.MessageType, m []byte) error
	ProcessRequestProof(msg *wire.RequestProofMessage) error
	ProcessCreateLog(scls *wire.SignedCreateLogStatement) error
	ProcessAppendLog(sls *wire.SignedLogStatement) error
}

type ServerLogProcessor struct {
	conn        *wire.Connection
	logIDs      [][]byte
	server      *Server
	autoUpdates bool
}

func NewLogProcessor(c net.Conn, srv *Server) LogProcessor {
	proc := &ServerLogProcessor{conn: wire.NewConnection(c), server: srv, logIDs: make([][]byte, 0)}
	srv.registerProcessor(proc)
	return proc
}

func (lp *ServerLogProcessor) Process() {
	for {
		t, m, e := lp.conn.ReadNextMessage()
		if e != nil {
			lp.server.unregisterProcessor(lp)
			lp.conn.Close()
			return
		}

		e = lp.ProcessMessage(t, m)
		if e != nil {
			lp.conn.WriteMessage(wire.MessageTypeError, []byte(e.Error()))
			lp.server.unregisterProcessor(lp)
			lp.conn.Close()
			return
		}
	}
}

func (lp *ServerLogProcessor) SendProofs(delta *mpt.DeltaMPT) error {
	if lp.autoUpdates && len(lp.logIDs) > 0 {
		clientDelta, err := delta.GetUpdatesForKeys(lp.logIDs)
		if err != nil {
			return err
		}

		err = lp.conn.WriteMessage(wire.MessageTypeProofUpdate, clientDelta.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

func (lp *ServerLogProcessor) ProcessMessage(t wire.MessageType, m []byte) error {
	if t == wire.MessageTypeCreateLog {
		pm, err := wire.NewSignedCreateLogStatementFromBytes(m)
		if err != nil {
			return err
		}
		return lp.ProcessCreateLog(pm)
	}

	if t == wire.MessageTypeAppendLog {
		pm, err := wire.NewSignedLogStatementFromBytes(m)
		if err != nil {
			return err
		}
		return lp.ProcessAppendLog(pm)
	}

	if t == wire.MessageTypeRequestProof {
		pm, err := wire.NewRequestProofMessageFromBytes(m)
		if err != nil {
			return err
		}
		return lp.ProcessRequestProof(pm)
	}

	if t == wire.MessageTypeRequestDeltaProof {
		pm, err := wire.NewRequestProofMessageFromBytes(m)
		if err != nil {
			return err
		}
		return lp.ProcessRequestDeltaProof(pm)
	}

	if t == wire.MessageTypeRequestCommitmentDetails {
		pm, err := wire.NewRequestCommitmentDetailsMessageFromBytes(m)
		if err != nil {
			return err
		}
		return lp.ProcessRequestCommitmentDetails(pm)
	}

	if t == wire.MessageTypeRequestCommitmentHistory {
		pm, err := wire.NewRequestCommitmentHistoryMessageFromBytes(m)
		if err != nil {
			return err
		}
		return lp.ProcessRequestCommitmentHistory(pm)
	}

	if t == wire.MessageTypeSubscribeProofUpdates {
		lp.autoUpdates = true
		lp.conn.WriteMessage(wire.MessageTypeAck, []byte{})
		return nil
	}

	if t == wire.MessageTypeUnsubscribeProofUpdates {
		lp.autoUpdates = false
		lp.conn.WriteMessage(wire.MessageTypeAck, []byte{})
		return nil
	}

	return fmt.Errorf("Unrecognized message type received: %x", byte(t))
}

func (lp *ServerLogProcessor) ProcessRequestProof(msg *wire.RequestProofMessage) error {
	keys := make([][]byte, len(msg.LogIDs))
	// If we didn't receive any keys as parameter, assume all
	// logs the client created or modified
	if len(keys) == 0 {
		keys = make([][]byte, len(lp.logIDs))
		for i, key := range lp.logIDs {
			keys[i] = make([]byte, 32)
			copy(keys[i], key[:])
		}
	} else {
		for i, key32 := range msg.LogIDs {
			keys[i] = make([]byte, 32)
			copy(keys[i], key32[:])
		}
	}
	fmt.Printf("Retrieving proof for %d keys\n", len(keys))
	proof, err := lp.server.GetProofForKeys(keys)
	if err != nil {
		return err
	}
	lp.conn.WriteMessagePrefix(wire.MessageTypeProof, proof.ByteSize())
	proof.Serialize(lp.conn)
	return nil

}

func (lp *ServerLogProcessor) ProcessRequestDeltaProof(msg *wire.RequestProofMessage) error {
	keys := make([][]byte, len(msg.LogIDs))
	// If we didn't receive any keys as parameter, assume all
	// logs the client created or modified
	if len(keys) == 0 {
		keys = make([][]byte, len(lp.logIDs))
		for i, key := range lp.logIDs {
			keys[i] = make([]byte, 32)
			copy(keys[i], key[:])
		}
	} else {
		for i, key32 := range msg.LogIDs {
			keys[i] = make([]byte, 32)
			copy(keys[i], key32[:])
		}
	}

	proof, err := lp.server.GetDeltaProofForKeys(keys)
	if err != nil {
		return err
	}
	lp.conn.WriteMessage(wire.MessageTypeDeltaProof, proof.Bytes())
	return nil
}

func (lp *ServerLogProcessor) ProcessCreateLog(scls *wire.SignedCreateLogStatement) error {

	err := scls.VerifySignature()
	if err != nil {
		return err
	}

	hash := fastsha256.Sum256(scls.CreateStatement.Bytes())

	err = lp.server.RegisterLogID(hash, scls.CreateStatement.ControllingKey)
	if err != nil {
		return err
	}

	witness := fastsha256.Sum256(scls.Bytes())

	// The only possible error is a wrong index. Given we _just_ created the log,
	// which should error out on already existing, the 0 index is always correct.
	// Therefore, it's safe to ignore this.
	_ = lp.server.RegisterLogStatement(hash, 0, witness[:])

	lp.SubscribeToLog(hash)
	lp.conn.WriteMessage(wire.MessageTypeAck, []byte{})
	return nil
}

func (lp *ServerLogProcessor) ProcessAppendLog(sls *wire.SignedLogStatement) error {
	err := lp.VerifyAppendLog(sls)
	if err != nil {
		return err
	}

	err = lp.CommitAppendLog(sls)
	if err != nil {
		return err
	}

	return lp.AckAppendLog(sls)
}

func (lp *ServerLogProcessor) VerifyAppendLog(sls *wire.SignedLogStatement) error {
	pk, err := lp.server.GetPubKeyForLogID(sls.Statement.LogID)
	if err != nil {
		return err
	}

	err = sls.VerifySignature(pk)
	if err != nil {
		return err
	}
	return nil
}
func (lp *ServerLogProcessor) CommitAppendLog(sls *wire.SignedLogStatement) error {
	witness := fastsha256.Sum256(sls.Bytes())
	err := lp.server.RegisterLogStatement(sls.Statement.LogID, sls.Statement.Index, witness[:])
	if err != nil {
		return err
	}
	return nil
}

func (lp *ServerLogProcessor) AckAppendLog(sls *wire.SignedLogStatement) error {
	lp.SubscribeToLog(sls.Statement.LogID)
	return lp.conn.WriteMessage(wire.MessageTypeAck, []byte{})
}

func (lp *ServerLogProcessor) SubscribeToLog(logID [32]byte) {
	for _, lid := range lp.logIDs {
		if bytes.Equal(lid[:], logID[:]) {
			return
		}
	}

	lp.logIDs = append(lp.logIDs, logID[:])
}

func (lp *ServerLogProcessor) ProcessRequestCommitmentDetails(pm *wire.RequestCommitmentDetailsMessage) error {
	c, err := lp.server.GetCommitmentDetails(pm.Commitment)
	if err != nil {
		return err
	}
	msg := wire.NewCommitmentDetailsMessage(c)
	return lp.conn.WriteMessage(wire.MessageTypeCommitmentDetails, msg.Bytes())
}

func (lp *ServerLogProcessor) ProcessRequestCommitmentHistory(pm *wire.RequestCommitmentHistoryMessage) error {
	c := lp.server.GetCommitmentHistory(pm.SinceCommitment)
	msg := wire.NewCommitmentHistoryMessage(c)
	return lp.conn.WriteMessage(wire.MessageTypeCommitmentHistory, msg.Bytes())
}
