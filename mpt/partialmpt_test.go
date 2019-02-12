package mpt

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestPartialMpt(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	k2, _ := hex.DecodeString("4043567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 2
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2
	k3, _ := hex.DecodeString("1843567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 3
	v3, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 3
	k4, _ := hex.DecodeString("ff34567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 4
	v4, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 4
	k5, _ := hex.DecodeString("bf34567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 5
	v5, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 5
	k6, _ := hex.DecodeString("d724567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 6
	v6, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 6
	mpt, err := NewFullMPT()
	if err != nil {
		t.Error(err.Error())
	}

	mpt.Insert(k1, v1)
	mpt.Insert(k5, v5)
	mpt.Insert(k6, v6)
	mpt.Insert(k4, v4)
	mpt.Insert(k2, v2)
	mpt.Insert(k3, v3)

	partialMpt, err := NewPartialMPT(mpt)
	if err != nil {
		t.Error(err.Error())
	}

	if !bytes.Equal(partialMpt.Commitment(), mpt.Commitment()) {
		t.Errorf("Expected commitment of partial and full MPT to match. They don't")
	}

	partialMpt, err = NewPartialMPTIncludingKey(mpt, k1)
	if err != nil {
		t.Error(err.Error())
	}

	if !bytes.Equal(partialMpt.Commitment(), mpt.Commitment()) {
		t.Errorf("Expected commitment of partial and full MPT to match. They don't")
	}

	v11, err := partialMpt.Get(k1)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(v11, v1) {
		t.Errorf("Expected value of k1 in partial and full MPT to match. They don't")
	}

	partialMpt, err = NewPartialMPTIncludingKeys(mpt, [][]byte{k1, k5})
	if err != nil {
		t.Error(err.Error())
	}

	if !bytes.Equal(partialMpt.Commitment(), mpt.Commitment()) {
		t.Errorf("Expected commitment of partial and full MPT to match. They don't")
	}

	v11, err = partialMpt.Get(k1)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(v11, v1) {
		t.Errorf("Expected value of k1 in partial and full MPT to match. They don't")
	}

	v21, err := partialMpt.Get(k2) // absent
	if err == nil && v21 != nil {
		t.Errorf("Expected value to be missing from partial MPT, but found it anyway?")
	}

	partialMpt2, err := NewPartialMPTFromBytes(partialMpt.Bytes())
	if err != nil {
		t.Error(err.Error())
	}

	if !bytes.Equal(partialMpt2.Commitment(), mpt.Commitment()) {
		t.Errorf("Expected commitment of deserialized partial MPT and full MPT to match. They don't")
	}

	partialMpt, _ = NewPartialMPTIncludingKey(mpt, k1)
	mpt.Insert(k1, v2)
	partialMpt2, _ = NewPartialMPTIncludingKey(mpt, k1)

	partialMpt.ProcessUpdates(partialMpt2)
	if !bytes.Equal(partialMpt.Commitment(), mpt.Commitment()) {
		t.Errorf("Expected commitment of deserialized partial MPT after update and full MPT to match. They don't")
	}

	partialMpt, _ = NewPartialMPTIncludingKey(mpt, k1)
	mpt.Insert(k1, v2)
	partialMpt2, _ = NewPartialMPTIncludingKey(mpt, k1)
	partialMpt.ProcessUpdatesFromBytes(partialMpt2.Bytes())
	if !bytes.Equal(partialMpt.Commitment(), mpt.Commitment()) {
		t.Errorf("Expected commitment of deserialized partial MPT after update (via bytes) and full MPT to match. They don't")
	}

	err = partialMpt.ProcessUpdatesFromBytes([]byte{})
	if err == nil {
		t.Errorf("Expected error in ProcessUpdateFromBytes with invalid slice. Got none")
	}

}

func TestPartialMptSerialize(t *testing.T) {
	_, err := NewPartialMPTFromBytes([]byte{})
	if err == nil {
		t.Error("Expected error on deserialize with invalid input, but got none")
	}

	eln, _ := NewEmptyLeafNode()
	_, err = NewPartialMPTFromBytes(eln.Bytes())
	if err == nil {
		t.Error("Expected error on deserialize with invalid input, but got none")
	}
}
