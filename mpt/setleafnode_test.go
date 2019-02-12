package mpt

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestSetLeafNodeLeftChild(t *testing.T) {
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected SetLeftChild to panic, but it did not")
		}
	}()

	sln, err := NewSetLeafNode(v1)
	if err != nil {
		t.Error(err.Error())
	}
	sln2, err := NewSetLeafNode(v2)
	if err != nil {
		t.Error(err.Error())
	}

	n := sln.GetLeftChild()
	if n != nil {
		t.Error("Expected left child to return nil, but it did not")
	}

	sln.SetLeftChild(sln2)
}

func TestSetLeafNodeRightChild(t *testing.T) {
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected SetRightChild to panic, but it did not")
		}
	}()

	sln, err := NewSetLeafNode(v1)
	if err != nil {
		t.Error(err.Error())
	}
	sln2, err := NewSetLeafNode(v2)
	if err != nil {
		t.Error(err.Error())
	}

	n := sln.GetRightChild()
	if n != nil {
		t.Error("Expected right child to return nil, but it did not")
	}

	sln.SetRightChild(sln2)
}

func TestSetLeafNodeKeyValue(t *testing.T) {
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected SetValue to panic, but it did not")
		}
	}()
	sln, err := NewSetLeafNode(v1)
	if err != nil {
		t.Error(err.Error())
	}

	n := sln.GetValue()
	if !bytes.Equal(n, v1) {
		t.Errorf("Expected GetValue to return %x, but it returned %x", v1, n)
	}

	n = sln.GetKey()
	if !bytes.Equal(n, v1) {
		t.Errorf("Expected GetKey to return %x, but it returned %x", v1, n)
	}

	sln.SetValue(v1)
}

func TestSetLeafNodeEquals(t *testing.T) {
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	v2, _ := hex.DecodeString("efdcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 2

	sln, err := NewSetLeafNode(v1)
	if err != nil {
		t.Error(err.Error())
	}
	sln2, err := NewSetLeafNode(v1)
	if err != nil {
		t.Error(err.Error())
	}
	sln3, err := NewSetLeafNode(v2)
	if err != nil {
		t.Error(err.Error())
	}

	s, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}

	if !sln.Equals(sln2) {
		t.Error("Expected Equals for two SetLeafNodes with the same  values to be true, but it was not")
	}

	if sln.Equals(sln3) {
		t.Error("Expected Equals for two SetLeafNodes with different values to be false, but it was not")
	}

	if sln.Equals(s) {
		t.Error("Expected Equals for a SetLeafNode and Stub to be false, but it was not")
	}
}

func TestSetLeafNodeGetHash(t *testing.T) {
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	sln, err := NewSetLeafNode(v1)
	if err != nil {
		t.Error(err)
	}
	if sln.CountHashesRequiredForGetHash() != 0 { // Hash == value for a SetLeafNode so no calculation needed
		t.Error("Expected CountHashesRequiredForGetHash to return 0, but it did not")
	}
	dh := sln.GetHash()
	if !bytes.Equal(dh, v1) { // Hash == value for a SetLeafNode
		t.Errorf("GetHash should return %x, but got %x", v1, dh)
	}
}

func TestSetLeafNodeSerialize(t *testing.T) {
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	sln, err := NewSetLeafNode(v1)
	if err != nil {
		t.Error(err.Error())
	}
	b := sln.Bytes()
	n, err := NodeFromBytes(b)
	if err != nil {
		t.Error(err.Error())
	}
	sln2, ok := n.(*SetLeafNode)
	if !ok {
		t.Error("Failed to deserialize SetLeafNode")
	}
	if !sln.Equals(sln2) {
		t.Error("Deserialized node did not equal input")
	}

	n, err = NewSetLeafNodeFromBytes([]byte{}) // Empty byte slice
	if err == nil {
		t.Error("NodeFromBytes with invalid data should have returned an error, but did not")
	}

	n, err = NewSetLeafNodeFromBytes([]byte{0xFF, 0xAB}) // Length for key, but no bytes with actual data
	if err == nil {
		t.Error("NodeFromBytes with invalid data should have returned an error, but did not")
	}

	n, err = NewSetLeafNodeFromBytes([]byte{0xFF}) // No data for key nor value
	if err == nil {
		t.Error("NodeFromBytes with invalid data should have returned an error, but did not")
	}

	n, err = NewSetLeafNodeFromBytes([]byte{0xFF, 0x00}) // zero-length key
	if err == nil {
		t.Error("NodeFromBytes with invalid data should have returned an error, but did not")
	}
}

func TestSetLeafNodeCounts(t *testing.T) {
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	sln, err := NewSetLeafNode(v1)
	if err != nil {
		t.Error(err.Error())
	}

	if sln.EmptyLeafNodesInSubtree() != 0 {
		t.Error("Expected EmptyLeafNodesInSubtree to return 0, but it did not")
	}
	if sln.InteriorNodesInSubtree() != 0 {
		t.Error("Expected InteriorNodesInSubtree to return 0, but it did not")
	}
	if sln.NodesInSubtree() != 1 {
		t.Error("Expected NodesInSubtree to return 1, but it did not")
	}
	if sln.NonEmptyLeafNodesInSubtree() != 1 {
		t.Error("Expected NonEmptyLeafNodesInSubtree to return 1, but it did not")
	}
}

func TestSetLeafNodeChanged(t *testing.T) {
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	sln, err := NewSetLeafNode(v1)
	if err != nil {
		t.Error(err.Error())
	}
	if err != nil {
		t.Error(err.Error())
	}
	if !sln.Changed() {
		t.Error("Expected changed for a new node to be true, but it was not")
	}
	sln.MarkUnchangedAll()
	if sln.Changed() {
		t.Error("Expected changed after call to MarkUnchangedAll to be false, but it was not")
	}
	sln.MarkChangedAll()
	if !sln.Changed() {
		t.Error("Expected changed after call to MarkChangedAll to be true, but it was not")
	}
}

func TestSetLeafNodeProperties(t *testing.T) {
	v1, _ := hex.DecodeString("fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321") // Value 1
	sln, err := NewSetLeafNode(v1)
	if err != nil {
		t.Error(err.Error())
	}
	if sln.IsEmpty() {
		t.Error("Expected IsEmpty for an SetLeafNode to be false, but it was not")
	}
	if sln.IsStub() {
		t.Error("Expected IsStub for an SetLeafNode to be false, but it was not")
	}
	if !sln.IsLeaf() {
		t.Error("Expected IsLeaf for an SetLeafNode to be true, but it was not")
	}
}
