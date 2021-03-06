package mpt

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
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
	stub := &Stub{hash: make([]byte, len(hash))}
	copy(stub.hash, hash)
	return stub, nil

}

func (s *Stub) Dispose() {
	s.hash = nil
	s = nil
}

// GetHash is the implementation of Node.GetHash
func (s *Stub) GetHash() []byte {
	return s.hash
}

// GetGraphHash is the implementation of Node.GetGraphHash
func (s *Stub) GetGraphHash() []byte {
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
func DeserializeNewStub(r io.Reader) (*Stub, error) {
	var stub []byte

	iLen := int32(0)
	err := binary.Read(r, binary.BigEndian, &iLen)
	if err != nil {
		return nil, err
	}
	if iLen > 0 {
		stub = make([]byte, iLen)
		i, err := r.Read(stub)
		if err != nil {
			return nil, err
		}
		if int32(i) != iLen {
			return nil, fmt.Errorf("Specified length of stub not present in buffer")
		}
	} else {
		return nil, fmt.Errorf("Dictionary leaf node needs a stub of at least 1 byte")
	}

	return NewStub(stub)
}

func (s *Stub) Serialize(w io.Writer) {
	w.Write([]byte{byte(NodeTypeStub)})
	binary.Write(w, binary.BigEndian, int32(len(s.hash)))
	w.Write(s.hash)
}

// ByteSize returns the length of Bytes() without actually serializing it
func (s *Stub) ByteSize() int {
	return 5 + len(s.hash)
}

// WriteGraphNodes is the implementation of Node.WriteGraphNodes
func (s *Stub) WriteGraphNodes(w io.Writer) {
	w.Write([]byte(fmt.Sprintf("\"%x\" [\n\tshape=box\n\tstyle=\"filled,dashed\"\n\ttextcolor=blue\n\tcolor=blue\n\tfillcolor=lightblue];\n", s.GetGraphHash())))
}

func (s *Stub) DeepCopy() (Node, error) {
	return NewStub(s.hash)
}
