package mpt

import (
	"bytes"
)

// Stub represents an omitted path in a MPT. Stubs only store a
// hash commitment to that subtree. These can be used to construct proofs and
// can be swapped out for actual subtrees which match the hash
type Stub struct {
	hash []byte
}

// Compile time check if Stub implements Node properly
var _ Node = &Stub{}

// NewStub creates a new stub from a hash
func NewStub(hash []byte) (*Stub, error) {
	return &Stub{hash: hash}, nil
}

// GetHash is the implementation of Node.GetHash
func (s *Stub) GetHash() []byte {
	return s.hash
}

// SetLeftChild is the implementation of Node.SetLeftChild
func (s *Stub) SetLeftChild(child Node) {
	panic("Cannot set children of a stub")
}

// SetRightChild is the implementation of Node.SetRightChild
func (s *Stub) SetRightChild(child Node) {
	panic("Cannot set children of a stub")
}

// GetLeftChild is the implementation of Node.GetLeftChild
func (s *Stub) GetLeftChild() Node {
	return nil
}

// GetRightChild is the implementation of Node.GetRightChild
func (s *Stub) GetRightChild() Node {
	return nil
}

// SetValue is the implementation of Node.SetValue
func (s *Stub) SetValue(value []byte) {
	panic("Cannot set the value of a stub")
}

// GetValue is the implementation of Node.GetValue
func (s *Stub) GetValue() []byte {
	return nil
}

// GetKey is the implementation of Node.GetKey
func (s *Stub) GetKey() []byte {
	return nil
}

// IsEmpty is the implementation of Node.IsEmpty
func (s *Stub) IsEmpty() bool {
	return false
}

// IsLeaf is the implementation of Node.IsLeaf
func (s *Stub) IsLeaf() bool {
	return false
}

// IsStub is the implementation of Node.IsStub
func (s *Stub) IsStub() bool {
	return true
}

// Changed is the implementation of Node.Changed
func (s *Stub) Changed() bool {
	return false
}

// MarkChangedAll is the implementation of Node.MarkChangedAll
func (s *Stub) MarkChangedAll() {}

// MarkUnchangedAll is the implementation of Node.MarkUnchangedAll
func (s *Stub) MarkUnchangedAll() {}

// CountHashesRequiredForGetHash is the implementation of Node.CountHashesRequiredForGetHash
func (s *Stub) CountHashesRequiredForGetHash() int {
	panic("Cannot count hashes for a stub")
}

// NodesInSubtree is the implementation of Node.NodesInSubtree
func (s *Stub) NodesInSubtree() int {
	panic("cannot determine size of subtree rooted at a stub")
}

// InteriorNodesInSubtree is the implementation of Node.InteriorNodesInSubtree
func (s *Stub) InteriorNodesInSubtree() int {
	panic("cannot determine number of interior nodes in subtree rooted at a stub")
}

// EmptyLeafNodesInSubtree is the implementation of Node.EmptyLeafNodesInSubtree
func (s *Stub) EmptyLeafNodesInSubtree() int {
	panic("cannot determine number of empty leaf nodes in subtree rooted at a stub")
}

// NonEmptyLeafNodesInSubtree is the implementation of Node.NonEmptyLeafNodesInSubtree
func (s *Stub) NonEmptyLeafNodesInSubtree() int {
	panic("cannot determine number of non-empty leaf nodes in subtree rooted at a stub")
}

// Equals is the implementation of Node.Equals
func (s *Stub) Equals(s2 Node) bool {
	stub2, ok := s2.(*Stub)
	if ok {
		return bytes.Equal(stub2.GetHash(), s.GetHash())
	}
	return false
}

// NewStubFromBytes deserializes the passed byteslice into a Stub
func NewStubFromBytes(b []byte) (*Stub, error) {
	return NewStub(b[1:]) // Lob off the type byte
}

// Bytes is the implementation of Node.Bytes
func (s *Stub) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(NodeTypeStub))
	buf.Write(s.hash)
	return buf.Bytes()
}

// ByteSize returns the length of Bytes() without actually serializing it
func (s *Stub) ByteSize() int {
	return 1 + len(s.hash)
}
