package mpt

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestNewFullMpt(t *testing.T) {
	mpt, err := NewFullMPT()
	if err != nil {
		t.Error(err.Error())
	}
	if mpt.CountNodes() != 3 {
		t.Error("Node count for new Full MPT is wrong")
	}
	if mpt.CountEmptyLeafNodes() != 2 {
		t.Error("Empty leaf node count for new Full MPT is wrong")
	}
	if mpt.CountInteriorNodes() != 1 {
		t.Error("Interior node count for new Full MPT is wrong")
	}
}

func TestFullMptInsertGetDeleteCommitment(t *testing.T) {

	// Test set: bit0(k1)=bit0(k2)=bit0(k3)=0 bit0(k4)=1 bit0(k5)=1
	// bit1(k4)==1 bit1(k5)==0
	// bit1(k4)==bit1(k6)
	// bit1(k1)==bit1(k3)!=bit1(k2)
	// bit2(k1)!=bit2(k3)
	// This ensures all insert paths are followed.

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
	c, _ := hex.DecodeString("a16df59e761b896e5307bcb4cfae20983d05dbb25a5d5cedf53704dcfe254e47")  // Expected commitment after writing the two key/value pairs
	c2, _ := hex.DecodeString("59f32be322b2cd259540a90c3d4cb6d41dfdf915c8e4fce5d5cb9a9e8069ff5b") // Expected commitment after deleting key2
	mpt, err := NewFullMPT()
	if err != nil {
		t.Error(err.Error())
	}
	mpt.Insert(k1, v2)
	mpt.Insert(k4, v4)
	mpt.Insert(k2, v2)
	mpt.Insert(k3, v3)
	mpt.Insert(k6, v6)
	mpt.Insert(k5, v5)

	if mpt.Size() != 6 {
		t.Errorf("MPT size incorrect. Expected 6, got %d", mpt.Size())
	}

	if mpt.MaxHeight() != 5 {
		t.Errorf("Wrong MaxHeight. Expected 5, got %d", mpt.MaxHeight())
	}

	// Insert duplicate key. Will just update the value
	mpt.Insert(k1, v1)
	if mpt.Size() != 6 {
		t.Errorf("MPT size incorrect. Expected 6, got %d", mpt.Size())
	}

	comm := mpt.Commitment()
	if !bytes.Equal(comm, c) {
		t.Errorf("Commitment hash incorrect. Expected %x, got %x", c, comm)
	}

	if mpt.MaxHeight() != 5 {
		t.Errorf("Wrong MaxHeight. Expected 5, got %d", mpt.MaxHeight())
	}

	mpt.Delete(k2)
	comm = mpt.Commitment()
	if !bytes.Equal(comm, c2) {
		t.Errorf("Commitment hash after deleting key 2 incorrect. Expected %x, got %x", c2, comm)
	}

	mpt.Delete(k2) // No longer in tree
	comm = mpt.Commitment()
	if !bytes.Equal(comm, c2) {
		t.Errorf("Commitment hash after deleting key 2 second time incorrect. Expected %x, got %x", c2, comm)
	}

	getV1 := mpt.Get(k1)
	if !bytes.Equal(getV1, v1) {
		t.Errorf("Getting back value 1 from mpt failed. Expected %x, got %x", v1, getV1)
	}

	getV2 := mpt.Get(k2)
	if getV2 != nil {
		t.Errorf("Got a value for key 2. Expected (nil), got %x", getV2)
	}

	if mpt.MaxHeight() != 5 {
		t.Errorf("Wrong MaxHeight. Expected 5, got %d", mpt.MaxHeight())
	}

	if !mpt.root.Changed() {
		t.Errorf("Expected mpt.root.Changed() to be true")
	}
	mpt.Reset()
	if mpt.root.Changed() {
		t.Errorf("Expected mpt.root.Changed() to be false")
	}

	mpt.Delete(k5)
	mpt.Delete(k3)
	mpt.Delete(k4)

	mpt.Delete(k6)
	if mpt.MaxHeight() != 1 {
		t.Errorf("Wrong MaxHeight. Expected 1, got %d", mpt.MaxHeight())
	}
}

func TestFullMptSerialize(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	k2, _ := hex.DecodeString("2134567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 2
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2
	mpt, err := NewFullMPT()
	if err != nil {
		t.Error(err.Error())
	}
	mpt.Insert(k1, v1)
	mpt.Insert(k2, v2)
	c1 := mpt.Commitment()
	b := mpt.Bytes()
	mpt2, err := NewFullMPTFromBytes(b)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if mpt2.Size() != mpt.Size() {
		t.Error("Size of deserialized tree not equal to input")
	}

	c2 := mpt2.Commitment()
	if !bytes.Equal(c1, c2) {
		t.Error("Commitment of deserialized tree not equal to input")
	}

	// Test invalid byte slices
	_, err = NewFullMPTFromBytes([]byte{})
	if err == nil {
		t.Error("Expected error from deserialization but got none")
	}

	eln, _ := NewEmptyLeafNode()
	_, err = NewFullMPTFromBytes(eln.Bytes())
	if err == nil {
		t.Error("Expected error from deserialization but got none")
	}
}
