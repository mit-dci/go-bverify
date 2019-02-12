package mpt

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestInteriorNodeKeyValue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected SetValue to panic, but it did not")
		}
	}()

	l1, _ := NewEmptyLeafNode()
	l2, _ := NewEmptyLeafNode()

	in, err := NewInteriorNode(l1, l2)
	if err != nil {
		t.Error(err.Error())
	}

	n := in.GetValue()
	if n != nil {
		t.Error("Expected GetValue to return nil, but it did not")
	}

	n = in.GetKey()
	if n != nil {
		t.Error("Expected GetKey to return nil, but it did not")
	}

	in.SetValue([]byte{})

}

func TestInteriorNodeChildren(t *testing.T) {
	in, err := NewInteriorNode(nil, nil)
	if err != nil {
		t.Error(err.Error())
	}
	if in.HasLeft() {
		t.Error("Expected hasLeft to return false, got true")
	}
	if in.HasRight() {
		t.Error("Expected hasRight to return false, got true")
	}

	l1, _ := NewEmptyLeafNode()
	l2, _ := NewEmptyLeafNode()

	in.SetLeftChild(l1)
	in.SetRightChild(l2)

	if !in.GetLeftChild().Equals(l1) {
		t.Error("GetLeftChild did not equal the input")
	}

	if !in.GetRightChild().Equals(l2) {
		t.Error("GetRightChild did not equal the input")
	}

	if !in.HasLeft() {
		t.Error("Expected hasLeft to return true, got false")
	}
	if !in.HasRight() {
		t.Error("Expected hasRight to return true, got false")
	}

}

func TestInterorNodeEquals(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	k2, _ := hex.DecodeString("2134567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 2
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2

	dln, _ := NewDictionaryLeafNode(k1, v1)
	dln2, _ := NewDictionaryLeafNode(k2, v2)

	in, _ := NewInteriorNode(dln, dln2)
	in2, _ := NewInteriorNode(dln, dln2)
	in3, _ := NewInteriorNode(dln2, dln)

	if !in.Equals(in2) {
		t.Error("Expected two interior nodes with the same children to n1.Equals(n2) == true (got false)")
	}

	if in.Equals(in3) {
		t.Error("Expected two interior nodes with different children to n1.Equals(n2) == false (got true)")
	}

	if in.Equals(dln) {
		t.Error("Expected interior node and dictionaryleafnode to n1.Equals(n2) == false (got true)")
	}
}

func TestInterorNodeGetHash(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	k2, _ := hex.DecodeString("2134567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 2
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2
	h, _ := hex.DecodeString("b0a16687f6701fea2b317cf84642a7de3093b34312e78b24da6cee66aee1fb23")  // Expected hash

	dln, _ := NewDictionaryLeafNode(k1, v1)
	dln2, _ := NewDictionaryLeafNode(k2, v2)

	in, _ := NewInteriorNode(dln, dln2)

	if in.CountHashesRequiredForGetHash() != 3 {
		t.Error("Expected CountHashesRequiredForGetHash to be 3, but it was not")
	}
	dh := in.GetHash()
	if !bytes.Equal(dh, h) {
		t.Errorf("GetHash should return %x, but got %x", h, dh)
	}
	if in.CountHashesRequiredForGetHash() != 0 {
		t.Error("Expected CountHashesRequiredForGetHash to be 0, but it was not")
	}
}

func TestInteriorNodeSerialize(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	k2, _ := hex.DecodeString("2134567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 2
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2
	dln, _ := NewDictionaryLeafNode(k1, v1)
	dln2, _ := NewDictionaryLeafNode(k2, v2)
	in, _ := NewInteriorNode(dln, dln2)
	b := in.Bytes()
	n, err := NodeFromBytes(b)
	if err != nil {
		t.Error(err.Error())
	}
	in2, ok := n.(*InteriorNode)
	if !ok {
		t.Error("Failed to deserialize DictionaryLeafNode")
	}
	if !in2.Equals(in) {
		t.Error("Deserialized node did not equal input")
	}
}

func TestInteriorNodeCounts(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	k2, _ := hex.DecodeString("2134567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 2
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2
	dln, _ := NewDictionaryLeafNode(k1, v1)
	dln2, _ := NewDictionaryLeafNode(k2, v2)
	in, _ := NewInteriorNode(dln, dln2)

	if in.EmptyLeafNodesInSubtree() != 0 {
		t.Error("Expected EmptyLeafNodesInSubtree to return 0, but it did not")
	}
	if in.InteriorNodesInSubtree() != 1 {
		t.Error("Expected InteriorNodesInSubtree to return 1, but it did not")
	}
	if in.NodesInSubtree() != 3 {
		t.Error("Expected NodesInSubtree to return 3, but it did not")
	}
	if in.NonEmptyLeafNodesInSubtree() != 2 {
		t.Error("Expected NonEmptyLeafNodesInSubtree to return 2, but it did not")
	}
}

func TestInteriorNodeChanged(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	k2, _ := hex.DecodeString("2134567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 2
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2
	dln, _ := NewDictionaryLeafNode(k1, v1)
	dln2, _ := NewDictionaryLeafNode(k2, v2)
	in, _ := NewInteriorNode(dln, dln2)

	if !in.Changed() {
		t.Error("Expected changed for a new node to be true, but it was not")
	}
	in.MarkUnchangedAll()
	if in.Changed() {
		t.Error("Expected changed after call to MarkUnchangedAll to be false, but it was not")
	}

	in.MarkChangedAll()
	if !in.Changed() {
		t.Error("Expected changed after call to MarkChangedAll to be true, but it was not")
	}

}

func TestInteriorNodeProperties(t *testing.T) {
	k1, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 1
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	k2, _ := hex.DecodeString("2134567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Key 2
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2
	dln, _ := NewDictionaryLeafNode(k1, v1)
	dln2, _ := NewDictionaryLeafNode(k2, v2)
	in, _ := NewInteriorNode(dln, dln2)

	if in.IsEmpty() {
		t.Error("Expected IsEmpty for an InteriorNode to be false, but it was not")
	}
	if in.IsStub() {
		t.Error("Expected IsStub for an InteriorNode to be false, but it was not")
	}
	if in.IsLeaf() {
		t.Error("Expected IsLeaf for an InteriorNode to be false, but it was not")
	}
}
