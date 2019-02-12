package mpt

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestDictionaryLeafNodeLeftChild(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	k2, _ := hex.DecodeString("2134567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 2
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected SetLeftChild to panic, but it did not")
		}
	}()

	dln, err := NewDictionaryLeafNode(k1, v1)
	if err != nil {
		t.Error(err.Error())
	}
	dln2, err := NewDictionaryLeafNode(k2, v2)
	if err != nil {
		t.Error(err.Error())
	}

	n := dln.GetLeftChild()
	if n != nil {
		t.Error("Expected left child to return nil, but it did not")
	}

	dln.SetLeftChild(dln2)
}

func TestDictionaryLeafNodeRightChild(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	k2, _ := hex.DecodeString("2134567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 2
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected SetRightChild to panic, but it did not")
		}
	}()

	dln, err := NewDictionaryLeafNode(k1, v1)
	if err != nil {
		t.Error(err.Error())
	}
	dln2, err := NewDictionaryLeafNode(k2, v2)
	if err != nil {
		t.Error(err.Error())
	}

	n := dln.GetRightChild()
	if n != nil {
		t.Error("Expected right child to return nil, but it did not")
	}

	dln.SetRightChild(dln2)
}

func TestDictionaryLeafNodeKeyValue(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2

	dln, err := NewDictionaryLeafNode(k1, v1)
	if err != nil {
		t.Error(err.Error())
	}

	n := dln.GetValue()
	if !bytes.Equal(n, v1) {
		t.Errorf("Expected GetValue to return %x, but it returned %x", v1, n)
	}

	n = dln.GetKey()
	if !bytes.Equal(n, k1) {
		t.Errorf("Expected GetKey to return %x, but it returned %x", k1, n)
	}

	dln.SetValue(v2)
	n = dln.GetValue()
	if !bytes.Equal(n, v2) {
		t.Errorf("Expected GetValue to return %x, but it returned %x", v2, n)
	}

}

func TestDictionaryLeafNodeEquals(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	k2, _ := hex.DecodeString("2134567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 2
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2

	dln, err := NewDictionaryLeafNode(k1, v1)
	if err != nil {
		t.Error(err.Error())
	}
	dln2, err := NewDictionaryLeafNode(k1, v1)
	if err != nil {
		t.Error(err.Error())
	}
	dln3, err := NewDictionaryLeafNode(k1, v2)
	if err != nil {
		t.Error(err.Error())
	}
	dln4, err := NewDictionaryLeafNode(k2, v2)
	if err != nil {
		t.Error(err.Error())
	}
	s, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}

	if !dln.Equals(dln2) {
		t.Error("Expected Equals for two DictionaryLeafNodes with the same keys and values to be true, but it was not")
	}

	if dln.Equals(dln3) {
		t.Error("Expected Equals for two DictionaryLeafNodes with the same keys and different values to be false, but it was not")
	}

	if dln.Equals(dln4) {
		t.Error("Expected Equals for two DictionaryLeafNodes with different keys and values to be false, but it was not")
	}

	if dln.Equals(s) {
		t.Error("Expected Equals for a DictionaryLeafNode and Stub to be false, but it was not")
	}
}

func TestDictionaryLeafNodeGetHash(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	h, _ := hex.DecodeString("56b2b68d5ce93c98e90f1a353370b7422c2953ecc3e0ba839cabe5cb2e3cfda2")  // Expected hash
	dln, err := NewDictionaryLeafNode(k1, v1)
	if err != nil {
		t.Error(err)
	}
	dh := dln.GetHash()
	if !bytes.Equal(dh, h) {
		t.Errorf("GetHash should return %x, but got %x", h, dh)
	}
}

func TestDictionaryLeafNodeSerialize(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	dln, err := NewDictionaryLeafNode(k1, v1)
	if err != nil {
		t.Error(err.Error())
	}
	b := dln.Bytes()
	n, err := NodeFromBytes(b)
	if err != nil {
		t.Error(err.Error())
	}
	dln2, ok := n.(*DictionaryLeafNode)
	if !ok {
		t.Error("Failed to deserialize DictionaryLeafNode")
	}
	if !dln.Equals(dln2) {
		t.Error("Deserialized node did not equal input")
	}

	n, err = NewDictionaryLeafNodeFromBytes([]byte{0xFF, 0xAB}) // Length for key, but no bytes with actual data
	if err == nil {
		t.Error("NodeFromBytes with invalid data should have returned an error, but did not")
	}

	n, err = NewDictionaryLeafNodeFromBytes([]byte{0xFF}) // No data for key nor value
	if err == nil {
		t.Error("NodeFromBytes with invalid data should have returned an error, but did not")
	}

	n, err = NewDictionaryLeafNodeFromBytes([]byte{0xFF, 0x00}) // zero-length key
	if err == nil {
		t.Error("NodeFromBytes with invalid data should have returned an error, but did not")
	}

	n, err = NewDictionaryLeafNodeFromBytes([]byte{0xFF, 0x01, 0x01, 0x00}) // zero-length value
	if err == nil {
		t.Error("NodeFromBytes with invalid data should have returned an error, but did not")
	}

	n, err = NewDictionaryLeafNodeFromBytes([]byte{0xFF, 0x01, 0x01}) // No data for value
	if err == nil {
		t.Error("NodeFromBytes with invalid data should have returned an error, but did not")
	}

	n, err = NewDictionaryLeafNodeFromBytes([]byte{}) // No type indicator
	if err == nil {
		t.Error("NodeFromBytes with invalid data should have returned an error, but did not")
	}
}

func TestDictionaryLeafNodeCounts(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	dln, err := NewDictionaryLeafNode(k1, v1)
	if err != nil {
		t.Error(err.Error())
	}

	if dln.EmptyLeafNodesInSubtree() != 0 {
		t.Error("Expected EmptyLeafNodesInSubtree to return 0, but it did not")
	}
	if dln.InteriorNodesInSubtree() != 0 {
		t.Error("Expected InteriorNodesInSubtree to return 0, but it did not")
	}
	if dln.NodesInSubtree() != 1 {
		t.Error("Expected NodesInSubtree to return 1, but it did not")
	}
	if dln.NonEmptyLeafNodesInSubtree() != 1 {
		t.Error("Expected NonEmptyLeafNodesInSubtree to return 1, but it did not")
	}
}

func TestDictionaryLeafNodeChanged(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	dln, err := NewDictionaryLeafNode(k1, v1)
	if err != nil {
		t.Error(err.Error())
	}
	if err != nil {
		t.Error(err.Error())
	}
	if !dln.Changed() {
		t.Error("Expected changed for a new node to be true, but it was not")
	}
	dln.MarkUnchangedAll()
	if dln.Changed() {
		t.Error("Expected changed after call to MarkUnchangedAll to be false, but it was not")
	}
	dln.MarkChangedAll()
	if !dln.Changed() {
		t.Error("Expected changed after call to MarkChangedAll to be true, but it was not")
	}
}

func TestDictionaryLeafNodeProperties(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	dln, err := NewDictionaryLeafNode(k1, v1)
	if err != nil {
		t.Error(err.Error())
	}
	if dln.IsEmpty() {
		t.Error("Expected IsEmpty for an DictionaryLeafNode to be false, but it was not")
	}
	if dln.IsStub() {
		t.Error("Expected IsStub for an DictionaryLeafNode to be false, but it was not")
	}
	if !dln.IsLeaf() {
		t.Error("Expected IsLeaf for an DictionaryLeafNode to be true, but it was not")
	}
	if dln.CountHashesRequiredForGetHash() != 1 {
		t.Error("Expected CountHashesRequiredForGetHash for an DictionaryLeafNode to be 1, but it was not")
	}
	dln.GetHash()
	if dln.CountHashesRequiredForGetHash() != 0 {
		t.Error("Expected CountHashesRequiredForGetHash after GetHash() to be 0, but it was not")
	}
}
