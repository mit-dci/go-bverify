package mpt

import (
	"testing"
)

func TestStubLeftChild(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected SetLeftChild to panic, but it did not")
		}
	}()

	s, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}
	s2, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}

	n := s.GetLeftChild()
	if n != nil {
		t.Error("Expected left child to return nil, but it did not")
	}

	s.SetLeftChild(s2)
}

func TestStubRightChild(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected SetRightChild to panic, but it did not")
		}
	}()

	s, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}
	s2, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}

	n := s.GetRightChild()
	if n != nil {
		t.Error("Expected right child to return nil, but it did not")
	}

	s.SetRightChild(s2)
}

func TestStubKeyValue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected SetValue to panic, but it did not")
		}
	}()

	s, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}
	n := s.GetKey()
	if n != nil {
		t.Error("Expected GetKey to return nil, but it did not")
	}
	n = s.GetValue()
	if n != nil {
		t.Error("Expected GetValue to return nil, but it did not")
	}

	s.SetValue([]byte{})
}

func TestStubEquals(t *testing.T) {
	s, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}
	s2, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}
	s3, err := NewStub([]byte{0x02})
	if err != nil {
		t.Error(err.Error())
	}
	eln, err := NewEmptyLeafNode()
	if err != nil {
		t.Error(err.Error())
	}

	if !s.Equals(s2) {
		t.Error("Expected two stubs with same 'hash' to return n1.Equal(n2) true, but got false")
	}
	if s.Equals(s3) {
		t.Error("Expected two stubs with different 'hash' to return n1.Equal(n2) false, but got true")
	}
	if s.Equals(eln) {
		t.Error("Expected a stub and empty leaf node to return n1.Equal(n2) false, but got true")
	}
}

func TestStubProperties(t *testing.T) {
	s, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}
	if s.IsEmpty() {
		t.Error("Expected IsEmpty for an Stub to be false, but it was not")
	}
	if !s.IsStub() {
		t.Error("Expected IsStub for an Stub to be true, but it was not")
	}
	if s.IsLeaf() {
		t.Error("Expected IsLeaf for an Stub to be false, but it was not")
	}
	if s.Changed() {
		t.Error("Expected Changed for an Stub to be false, but it was not")
	}
}

func TestStubSerialize(t *testing.T) {
	s, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}
	b := s.Bytes()
	n, err := NodeFromBytes(b)
	if err != nil {
		t.Error(err.Error())
	}
	s2, ok := n.(*Stub)
	if !ok {
		t.Error("Failed to deserialize Stub")
	}
	if !s.Equals(s2) {
		t.Error("Deserialized node did not equal input")
	}
}

func TestStubCountHashesRequiredForGetHash(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected CountHashesRequiredForGetHash to panic, but it did not")
		}
	}()

	s, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}
	s.CountHashesRequiredForGetHash()
}

func TestStubNodesInSubtree(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected NodesInSubtree to panic, but it did not")
		}
	}()

	s, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}
	s.NodesInSubtree()
}

func TestStubInteriorNodesInSubtree(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected InteriorNodesInSubtree to panic, but it did not")
		}
	}()

	s, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}
	s.InteriorNodesInSubtree()
}

func TestStubEmptyLeafNodesInSubtree(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected EmptyLeafNodesInSubtree to panic, but it did not")
		}
	}()

	s, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}
	s.EmptyLeafNodesInSubtree()
}

func TestStubNonEmptyLeafNodesInSubtree(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected NonEmptyLeafNodesInSubtree to panic, but it did not")
		}
	}()

	s, err := NewStub([]byte{0x01})
	if err != nil {
		t.Error(err.Error())
	}
	s.NonEmptyLeafNodesInSubtree()
}
