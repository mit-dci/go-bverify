package mpt

import (
	"encoding/hex"
	"testing"
)

func TestGetNodeHeight(t *testing.T) {
	el1, _ := NewEmptyLeafNode()
	el2, _ := NewEmptyLeafNode()

	in1, _ := NewInteriorNode(el1, el2)

	el3, _ := NewEmptyLeafNode()
	in2, _ := NewInteriorNode(in1, el3)

	el4, _ := NewEmptyLeafNode()
	in3, _ := NewInteriorNode(in2, el4)

	if GetNodeHeight(el1) != 0 {
		t.Error("Expected node height of leaf to be 0, it was not")
	}

	if GetNodeHeight(in1) != 1 {
		t.Error("Expected node height of in1 to be 1, it was not")
	}

	if GetNodeHeight(in2) != 2 {
		t.Error("Expected node height of in2 to be 2, it was not")
	}

	if GetNodeHeight(in3) != 3 {
		t.Error("Expected node height of in3 to be 3, it was not")
	}
}

func TestNodeSerialize(t *testing.T) {
	_, err := NodeFromBytes([]byte{}) // empty slice
	if err == nil {
		t.Error("Expected error deserializing empty slice, got none")
	}
}

func TestUpdateNode(t *testing.T) {
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2

	s1, _ := NewStub(v1)
	s2, _ := NewStub(v2)

	n, _ := UpdateNode(nil, s1)
	if !n.Equals(s1) {
		t.Error("Update node failed")
	}

	in1, _ := NewInteriorNode(s1, nil)
	in2, _ := NewInteriorNode(nil, s2)

	n, _ = UpdateNode(in1, in2)
	if !n.GetLeftChild().Equals(s1) || !n.GetRightChild().Equals(s2) {
		t.Error("Update node failed")
	}

	n, _ = UpdateNode(in2, in1)
	if !n.GetLeftChild().Equals(s1) || !n.GetRightChild().Equals(s2) {
		t.Error("Update node failed")
	}

	b := in2.Bytes()
	n, _ = UpdateNodeFromBytes(in1, b)
	if !n.GetLeftChild().Equals(s1) || !n.GetRightChild().Equals(s2) {
		t.Error("Update node failed")
	}

	_, err := UpdateNodeFromBytes(in1, []byte{})
	if err == nil {
		t.Error("Expected error because of empty byteslice but got none")
	}
}
