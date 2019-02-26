package mpt

import (
	"bytes"
	"fmt"
	"io"
)

// SetLeafNode represents a leaf node in a Merkle Prefix Trie
// (MPT) dictionary. Set leaf nodes store a key and a value,
// both of which are fixed length byte arrays (usually
// the outputs of a cryptographic hash). The value of
// a leaf can be updated.
type SetLeafNode struct {
	value           []byte
	changed         bool
	recalculateHash bool
}

// Compile time check if SetLeafNode implements Node properly
var _ Node = &SetLeafNode{}

// NewSetLeafNode creates a new dictionary leaf node
func NewSetLeafNode(value []byte) (*SetLeafNode, error) {
	return &SetLeafNode{value: value, changed: true}, nil
}

// GetHash is the implementation of Node.GetHash
func (sln *SetLeafNode) GetHash() []byte {
	return sln.value
}

// GetGraphHash is the implementation of Node.GetGraphHash
func (sln *SetLeafNode) GetGraphHash() []byte {
	return sln.value
}

// SetLeftChild is the implementation of Node.SetLeftChild
func (sln *SetLeafNode) SetLeftChild(child Node) {
	panic("Cannot set children of a leaf node")
}

// SetRightChild is the implementation of Node.SetRightChild
func (sln *SetLeafNode) SetRightChild(child Node) {
	panic("Cannot set children of a leaf node")
}

// GetLeftChild is the implementation of Node.GetLeftChild
func (sln *SetLeafNode) GetLeftChild() Node {
	return nil
}

// GetRightChild is the implementation of Node.GetRightChild
func (sln *SetLeafNode) GetRightChild() Node {
	return nil
}

// SetValue is the implementation of Node.SetValue
func (sln *SetLeafNode) SetValue(value []byte) {
	panic("Cannot set the value of a set leaf node")
}

// GetValue is the implementation of Node.GetValue
func (sln *SetLeafNode) GetValue() []byte {
	return sln.value
}

// GetKey is the implementation of Node.GetKey
func (sln *SetLeafNode) GetKey() []byte {
	return sln.value
}

// IsEmpty is the implementation of Node.IsEmpty
func (sln *SetLeafNode) IsEmpty() bool {
	return false
}

// IsLeaf is the implementation of Node.IsLeaf
func (sln *SetLeafNode) IsLeaf() bool {
	return true
}

// IsStub is the implementation of Node.IsStub
func (sln *SetLeafNode) IsStub() bool {
	return false
}

// Changed is the implementation of Node.Changed
func (sln *SetLeafNode) Changed() bool {
	return sln.changed
}

// MarkChangedAll is the implementation of Node.MarkChangedAll
func (sln *SetLeafNode) MarkChangedAll() {
	sln.changed = true
}

// MarkUnchangedAll is the implementation of Node.MarkUnchangedAll
func (sln *SetLeafNode) MarkUnchangedAll() {
	sln.changed = false
}

// CountHashesRequiredForGetHash is the implementation of Node.CountHashesRequiredForGetHash
func (sln *SetLeafNode) CountHashesRequiredForGetHash() int {
	return 0
}

// NodesInSubtree is the implementation of Node.NodesInSubtree
func (sln *SetLeafNode) NodesInSubtree() int {
	return 1
}

// InteriorNodesInSubtree is the implementation of Node.InteriorNodesInSubtree
func (sln *SetLeafNode) InteriorNodesInSubtree() int {
	return 0
}

// EmptyLeafNodesInSubtree is the implementation of Node.EmptyLeafNodesInSubtree
func (sln *SetLeafNode) EmptyLeafNodesInSubtree() int {
	return 0
}

// NonEmptyLeafNodesInSubtree is the implementation of Node.NonEmptyLeafNodesInSubtree
func (sln *SetLeafNode) NonEmptyLeafNodesInSubtree() int {
	return 1
}

// Equals is the implementation of Node.Equals
func (sln *SetLeafNode) Equals(n Node) bool {
	sln2, ok := n.(*SetLeafNode)
	if ok {
		return bytes.Equal(sln2.GetValue(), sln.GetValue())
	}
	return false
}

// NewSetLeafNodeFromBytes deserializes the passed byteslice into a SetLeafNode
func NewSetLeafNodeFromBytes(b []byte) (*SetLeafNode, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("Passed byte slice should be more than 0 bytes")
	}
	return NewSetLeafNode(b[1:]) // Lob off type byte
}

// Bytes is the implementation of Node.Bytes
func (sln *SetLeafNode) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(NodeTypeSetLeaf))
	buf.Write(sln.value)
	return buf.Bytes()
}

func (sln *SetLeafNode) ByteSize() int {
	return 1 + len(sln.value)
}

// WriteGraphNodes is the implementation of Node.WriteGraphNodes
func (sln *SetLeafNode) WriteGraphNodes(w io.Writer) {
	w.Write([]byte(fmt.Sprintf("\"%x\" [\n\tshape=box\n\tstyle=\"filled,dashed\"\n\tcolor=red4\n\tfillcolor=salmon];\n", sln.GetGraphHash())))
}
