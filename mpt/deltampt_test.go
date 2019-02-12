package mpt

import (
	"encoding/hex"
	"testing"
)

func TestDeltaMpt(t *testing.T) {
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
	deltaMpt, err := NewDeltaMPT(mpt)
	if err != nil {
		t.Error(err.Error())
	}
	deltaMpt.GetUpdatesForKey(k1)

	mpt.Insert(k1, v2)
	deltaMpt, err = NewDeltaMPT(mpt)
	if err != nil {
		t.Error(err.Error())
	}
	deltaMpt.GetUpdatesForKey(k1)

	mpt.Insert(k5, v5)
	mpt.Insert(k6, v6)
	mpt.Insert(k4, v4)
	mpt.Insert(k2, v2)
	mpt.Insert(k3, v3)
	mpt.Reset()

	deltaMpt, err = NewDeltaMPT(mpt)
	if err != nil {
		t.Error(err.Error())
	}

	deltaMpt.GetUpdatesForKey(k1)

	mpt.Insert(k1, v1)
	deltaMpt2, err := NewDeltaMPT(mpt)
	if err != nil {
		t.Error(err.Error())
	}

	deltaMpt2.GetUpdatesForKey(k2)
	deltaMpt2.GetUpdatesForKey(k3)

	deltaMpt3, err := NewDeltaMPTFromBytes(deltaMpt.Bytes())
	if err != nil {
		t.Error(err.Error())
	}

	if !deltaMpt3.root.Equals(deltaMpt.root) {
		t.Error("Deserialized mpt not equal to input")
	}

}

func TestDeltaMptSerialize(t *testing.T) {
	_, err := NewDeltaMPTFromBytes([]byte{})
	if err == nil {
		t.Error("Expected error on deserialize with invalid input, but got none")
	}

	eln, _ := NewEmptyLeafNode()
	_, err = NewDeltaMPTFromBytes(eln.Bytes())
	if err == nil {
		t.Error("Expected error on deserialize with invalid input, but got none")
	}
}
