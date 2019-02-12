package mpt

import (
	"bytes"
	"testing"
)

func TestEmptyLeafNodeLeftChild(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected SetLeftChild to panic, but it did not")
		}
	}()

	eln, err := NewEmptyLeafNode()
	if err != nil {
		t.Error(err.Error())
	}
	eln2, err := NewEmptyLeafNode()
	if err != nil {
		t.Error(err.Error())
	}

	n := eln.GetLeftChild()
	if n != nil {
		t.Error("Expected left child to return nil, but it did not")
	}

	eln.SetLeftChild(eln2)
}

func TestEmptyLeafNodeRightChild(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected SetRightChild to panic, but it did not")
		}
	}()

	eln, err := NewEmptyLeafNode()
	if err != nil {
		t.Error(err.Error())
	}
	eln2, err := NewEmptyLeafNode()
	if err != nil {
		t.Error(err.Error())
	}

	n := eln.GetRightChild()
	if n != nil {
		t.Error("Expected right child to return nil, but it did not")
	}

	eln.SetRightChild(eln2)
}

func TestEmptyLeafNodeKeyValue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected SetValue to panic, but it did not")
		}
	}()

	eln, err := NewEmptyLeafNode()
	if err != nil {
		t.Error(err.Error())
	}

	n := eln.GetValue()
	if n != nil {
		t.Error("Expected GetValue to return nil, but it did not")
	}

	n = eln.GetKey()
	if n != nil {
		t.Error("Expected GetKey to return nil, but it did not")
	}

	eln.SetValue([]byte{})
}

func TestEmptyLeafNodeEquals(t *testing.T) {
	eln, err := NewEmptyLeafNode()
	if err != nil {
		t.Error(err.Error())
	}
	eln2, err := NewEmptyLeafNode()
	if err != nil {
		t.Error(err.Error())
	}
	if !eln.Equals(eln2) {
		t.Error("Expected two empty leaf nodes to return n1.Equal(n2) true, but got false")
	}
}

func TestEmptyLeafNodeEmptyLeafNodesInSubtree(t *testing.T) {
	eln, err := NewEmptyLeafNode()
	if err != nil {
		t.Error(err.Error())
	}
	if eln.EmptyLeafNodesInSubtree() != 1 {
		t.Error("Expected EmptyLeafNodesInSubtree to return 1, but it did not")
	}
}

func TestEmptyLeafNodeNodesInSubtree(t *testing.T) {
	eln, err := NewEmptyLeafNode()
	if err != nil {
		t.Error(err.Error())
	}
	if eln.NodesInSubtree() != 1 {
		t.Error("Expected NodesInSubtree to return 1, but it did not")
	}
}

func TestEmptyLeafNodeNonEmptyLeafNodesInSubtree(t *testing.T) {
	eln, err := NewEmptyLeafNode()
	if err != nil {
		t.Error(err.Error())
	}
	if eln.NonEmptyLeafNodesInSubtree() != 0 {
		t.Error("Expected NonEmptyLeafNodesInSubtree to return 0, but it did not")
	}
}

func TestEmptyLeafNodeChanged(t *testing.T) {
	eln, err := NewEmptyLeafNode()
	if err != nil {
		t.Error(err.Error())
	}
	if !eln.Changed() {
		t.Error("Expected changed for a new node to be true, but it was not")
	}
	eln.MarkUnchangedAll()
	if eln.Changed() {
		t.Error("Expected changed after call to MarkUnchangedAll to be false, but it was not")
	}
	eln.MarkChangedAll()
	if !eln.Changed() {
		t.Error("Expected changed after call to MarkChangedAll to be true, but it was not")
	}
}

func TestEmptyLeafNodeProperties(t *testing.T) {
	eln, err := NewEmptyLeafNode()
	if err != nil {
		t.Error(err.Error())
	}
	if !eln.IsEmpty() {
		t.Error("Expected IsEmpty for an EmptyLeafNode to be true, but it was not")
	}
	if eln.IsStub() {
		t.Error("Expected IsStub for an EmptyLeafNode to be false, but it was not")
	}
	if !eln.IsLeaf() {
		t.Error("Expected IsLeaf for an EmptyLeafNode to be true, but it was not")
	}
	if eln.CountHashesRequiredForGetHash() != 0 {
		t.Error("Expected CountHashesRequiredForGetHash for an EmptyLeafNode to be 0, but it was not")
	}
	if eln.InteriorNodesInSubtree() != 0 {
		t.Error("Expected InteriorNodesInSubtree for an EmptyLeafNode to be 0, but it was not")
	}

	emptyHash := make([]byte, 32)
	elnHash := eln.GetHash()
	if !bytes.Equal(emptyHash, elnHash) {
		t.Errorf("Expected EmptyLeafNode.GetHash() to return %x but got %x", emptyHash, elnHash)
	}
}

func TestEmptyLeafNodeSerialize(t *testing.T) {
	eln, err := NewEmptyLeafNode()
	if err != nil {
		t.Error(err.Error())
	}
	b := eln.Bytes()
	n, err := NodeFromBytes(b)
	if err != nil {
		t.Error(err.Error())
	}
	eln2, ok := n.(*EmptyLeafNode)
	if !ok {
		t.Error("Failed to deserialize EmptyLeafNode")
	}
	if !eln.Equals(eln2) {
		t.Error("Deserialized node did not equal input")
	}
}
