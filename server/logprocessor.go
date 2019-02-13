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
	ProcessCreateLog(scls *wire.SignedCreateLogStatement) error
	ProcessAppendLog(sls *wire.SignedLogStatement) error
}

type ServerLogProcessor struct {
	conn   *wire.Connection
	logIDs [][]byte
	server *Server
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
	if len(lp.logIDs) > 0 {
		clientDelta, err := delta.GetUpdatesForKeys(lp.logIDs)
		if err != nil {
			return err
		}
		lp.conn.WriteMessage(wire.MessageTypeProofUpdate, clientDelta.Bytes())
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

	return fmt.Errorf("Unrecognized message type received")
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
	pk, err := lp.server.GetPubKeyForLogID(sls.Statement.LogID)
	if err != nil {
		return err
	}

	err = sls.VerifySignature(pk)
	if err != nil {
		return err
	}

	witness := fastsha256.Sum256(sls.Bytes())
	err = lp.server.RegisterLogStatement(sls.Statement.LogID, sls.Statement.Index, witness[:])
	if err != nil {
		return err
	}

	lp.SubscribeToLog(sls.Statement.LogID)
	lp.conn.WriteMessage(wire.MessageTypeAck, []byte{})
	return nil
}

func (lp *ServerLogProcessor) SubscribeToLog(logID [32]byte) {
	for _, lid := range lp.logIDs {
		if bytes.Equal(lid[:], logID[:]) {
			return
		}
	}

	lp.logIDs = append(lp.logIDs, logID[:])

}
