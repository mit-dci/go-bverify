package server

import (
	"bytes"
	"fmt"
	"net"
	"testing"

	"crypto/rand"

	"github.com/mit-dci/go-bverify/crypto/btcec"
	"github.com/mit-dci/go-bverify/crypto/fastsha256"
	"github.com/mit-dci/go-bverify/crypto/sig64"
	"github.com/mit-dci/go-bverify/mpt"
	"github.com/mit-dci/go-bverify/wire"
)

func sendMessageTest(title string, c *wire.Connection, tSend, tExpected wire.MessageType, msg []byte, t *testing.T) bool {
	c.WriteMessage(tSend, msg)
	mt, m, err := c.ReadNextMessage()
	if err != nil {
		t.Error(err)
		return false
	}

	if mt != tExpected {
		fmt.Printf("< [%s]", string(m))
		t.Errorf("%s: Expected message type [%x] response, got something else: [%x]", title, tExpected, byte(mt))
		return false
	}
	return true
}

func newDummyClient(srv *Server) *wire.Connection {
	server, client := net.Pipe()
	p := NewLogProcessor(server, srv)
	go p.Process()
	return wire.NewConnection(client)
}

func TestLogProcessor(t *testing.T) {

	createLog, appendLog, appendLog2, err := generateCreateAppendMessages()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Creating new server...")
	srv, _ := NewServer("", 0)
	c := newDummyClient(srv)

	createLogInvalidSig := make([]byte, len(createLog))
	copy(createLogInvalidSig[:], createLog)
	copy(createLogInvalidSig[0:], []byte{0x00, 0x00, 0x00, 0x00})
	if !sendMessageTest("Invalid signature create", c, wire.MessageTypeCreateLog, wire.MessageTypeError, createLogInvalidSig, t) {
		return
	}

	// Server will disconnect us after an error, so re"connect"
	c.Close()
	c = newDummyClient(srv)

	if !sendMessageTest("AppendLog for non-existent log", c, wire.MessageTypeAppendLog, wire.MessageTypeError, appendLog, t) {
		return
	}

	c.Close()
	c = newDummyClient(srv)

	if !sendMessageTest("Normal create log", c, wire.MessageTypeCreateLog, wire.MessageTypeAck, createLog, t) {
		return
	}

	if !sendMessageTest("Create duplicate", c, wire.MessageTypeCreateLog, wire.MessageTypeError, createLog, t) {
		return
	}

	c.Close()
	c = newDummyClient(srv)

	appendLogInvalidSig := make([]byte, len(appendLog))
	copy(appendLogInvalidSig[:], appendLog)
	copy(appendLogInvalidSig[0:], []byte{0x00, 0x00, 0x00, 0x00})
	if !sendMessageTest("Invalid signature append", c, wire.MessageTypeAppendLog, wire.MessageTypeError, appendLogInvalidSig, t) {
		return
	}

	c.Close()
	c = newDummyClient(srv)

	// Normal append logs, should succeed
	if !sendMessageTest("Append 1", c, wire.MessageTypeAppendLog, wire.MessageTypeAck, appendLog, t) {
		return
	}

	if !sendMessageTest("Append 2", c, wire.MessageTypeAppendLog, wire.MessageTypeAck, appendLog2, t) {
		return
	}

	// Subscribe to proof updates
	if !sendMessageTest("SubscribeProof", c, wire.MessageTypeSubscribeProofUpdates, wire.MessageTypeAck, []byte{}, t) {
		return
	}

	// Test get proofs
	go srv.Commit()
	mt, m, _ := c.ReadNextMessage() // Should be a proof update
	if mt != wire.MessageTypeProofUpdate {
		t.Errorf("Expected proof update, in stead got message type [%x]", mt)
		return
	}

	// Get commitment from the proof update and compare with the one the server committed.
	partialMpt, _ := mpt.NewPartialMPTFromBytes(m)
	comm := partialMpt.Commitment()
	if !bytes.Equal(srv.lastCommitment[:], comm) {
		t.Errorf("Proof update contains wrong commitment: [%x], expected [%x]", comm, srv.lastCommitment[:])
		return
	}

	if !sendMessageTest("Duplicate append", c, wire.MessageTypeAppendLog, wire.MessageTypeError, appendLog, t) {
		return
	}

	c.Close()
	c = newDummyClient(srv)

	if !sendMessageTest("Send the server a proof update", c, wire.MessageTypeProofUpdate, wire.MessageTypeError, []byte{}, t) {
		return
	}

	c.Close()
}

func generateCreateAppendMessages() ([]byte, []byte, []byte, error) {
	key := [32]byte{}
	rand.Read(key[:])
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key[:])
	var pk [33]byte
	copy(pk[:], pub.SerializeCompressed())

	l := wire.NewSignedCreateLogStatement(pk, []byte("Hello World"))
	logId := fastsha256.Sum256(l.CreateStatement.Bytes())
	sig, err := priv.Sign(logId[:])

	if err != nil {
		return nil, nil, nil, err
	}
	csig, err := sig64.SigCompress(sig.Serialize())
	if err != nil {
		return nil, nil, nil, err
	}
	l.Signature = csig

	l2 := wire.NewSignedLogStatement(1, logId, []byte("Hello World 2"))
	hash := fastsha256.Sum256(l2.Statement.Bytes())
	sig, err = priv.Sign(hash[:])

	if err != nil {
		return nil, nil, nil, err
	}
	csig, err = sig64.SigCompress(sig.Serialize())
	if err != nil {
		return nil, nil, nil, err
	}
	l2.Signature = csig

	l3 := wire.NewSignedLogStatement(2, logId, []byte("Hello World 3"))
	hash = fastsha256.Sum256(l3.Statement.Bytes())
	sig, err = priv.Sign(hash[:])

	if err != nil {
		return nil, nil, nil, err
	}
	csig, err = sig64.SigCompress(sig.Serialize())
	if err != nil {
		return nil, nil, nil, err
	}
	l3.Signature = csig

	return l.Bytes(), l2.Bytes(), l3.Bytes(), nil
}
