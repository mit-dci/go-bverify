package wire

import (
	"bytes"
	"encoding/hex"
)

import (
	"testing"
)

func TestSignedCreateLogStatement(t *testing.T) {
	pubKey, _ := hex.DecodeString("768412320f7b0aa5812fce428dc4706b3cae50e02a64caa16a782249bfe8efc4b7")
	pubKey33 := [33]byte{}
	copy(pubKey33[:], pubKey)
	n := NewSignedCreateLogStatement(pubKey33, []byte("Hello world"))
	sig, _ := hex.DecodeString("ee26b0dd4af7e749aa1a8ee3c10ae9923f618980772e473f8819a5d4940e0db27ac185f8a0e1d5f84f88bc887fd67b143732c304cc5fa9ad8e6f57f50028a8ff")
	copy(n.Signature[:], sig)
	n2, err := NewSignedCreateLogStatementFromBytes(n.Bytes())
	if err != nil {
		t.Error(err)
		return
	}

	if !bytes.Equal(n.Signature[:], n2.Signature[:]) {
		t.Errorf("Deserialized and serialized Signature not equal")
		return
	}

	if !bytes.Equal(n.CreateStatement.ControllingKey[:], n2.CreateStatement.ControllingKey[:]) {
		t.Errorf("Deserialized and serialized ControllingKey not equal")
		return
	}

	if !bytes.Equal(n.CreateStatement.InitialStatement[:], n2.CreateStatement.InitialStatement[:]) {
		t.Errorf("Deserialized and serialized InitialStatement not equal")
		return
	}

	err = n2.VerifySignature()
	if err == nil {
		t.Error("Expected signature verification to fail, but it succeeded")
		return
	}

	_, err = NewSignedCreateLogStatementFromBytes([]byte{0x00}) // Invalid, expect error
	if err == nil {
		t.Error("Expected deserialization error but got none")
		return
	}

	// does contain sig, but no complete controllingkey
	faultyMsg := make([]byte, 66)
	copy(faultyMsg[:], sig)

	_, err = NewSignedCreateLogStatementFromBytes(faultyMsg) // Invalid, expect error
	if err == nil {
		t.Error("Expected deserialization error but got none")
		return
	}
}

func TestSignedLogStatement(t *testing.T) {
	logId, _ := hex.DecodeString("768412320f7b0aa5812fce428dc4706b3cae50e02a64caa16a782249bfe8efc4")
	logId32 := [32]byte{}
	copy(logId32[:], logId)
	n := NewSignedLogStatement(0, logId32, []byte("Hello world"))
	sig, _ := hex.DecodeString("ee26b0dd4af7e749aa1a8ee3c10ae9923f618980772e473f8819a5d4940e0db27ac185f8a0e1d5f84f88bc887fd67b143732c304cc5fa9ad8e6f57f50028a8ff")
	copy(n.Signature[:], sig)
	n2, err := NewSignedLogStatementFromBytes(n.Bytes())
	if err != nil {
		t.Error(err)
		return
	}

	if !bytes.Equal(n.Signature[:], n2.Signature[:]) {
		t.Errorf("Deserialized and serialized Signature not equal")
		return
	}

	if !bytes.Equal(n.Statement.LogID[:], n.Statement.LogID[:]) {
		t.Errorf("Deserialized and serialized LogID not equal")
		return
	}

	if !bytes.Equal(n.Statement.Statement[:], n2.Statement.Statement[:]) {
		t.Errorf("Deserialized and serialized Statement not equal")
		return
	}

	if n.Statement.Index != n2.Statement.Index {
		t.Errorf("Deserialized and serialized Index not equal")
		return
	}

	pubKey, _ := hex.DecodeString("768412320f7b0aa5812fce428dc4706b3cae50e02a64caa16a782249bfe8efc4b7")
	pubKey33 := [33]byte{}
	copy(pubKey33[:], pubKey)
	err = n2.VerifySignature(pubKey33)
	if err == nil {
		t.Error("Expected signature verification to fail, but it succeeded")
		return
	}

	_, err = NewSignedLogStatementFromBytes([]byte{0x00}) // Invalid, expect error
	if err == nil {
		t.Error("Expected deserialization error but got none")
		return
	}

	// does contain sig, but no complete logID
	faultyMsg := make([]byte, 66)
	copy(faultyMsg[:], sig)

	_, err = NewSignedLogStatementFromBytes(faultyMsg) // Invalid, expect error
	if err == nil {
		t.Error("Expected deserialization error but got none")
		return
	}
}
