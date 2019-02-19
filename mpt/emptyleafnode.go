package mpt

import (
	"bytes"
)

// EmptyLeafNode represents an empty leaf in the tree. Empty leaves
// do not have associated values and use the special marker
// hash of all 0s.
//
type EmptyLeafNode struct {
	changed bool
	hash    []byte
}

// Compile time check if DictionaryLeafNode implements Node properly
var _ Node = &EmptyLeafNode{}

// NewEmptyLeafNode creates a new empty leaf node
func NewEmptyLeafNode() (*EmptyLeafNode, error) {
	return &EmptyLeafNode{changed: true, hash: make([]byte, 32)}, nil
}

// GetHash is the implementation of Node.GetHash
func (eln *EmptyLeafNode) GetHash() []byte {
	return eln.hash
}

// SetLeftChild is the implementation of Node.SetLeftChild
func (eln *EmptyLeafNode) SetLeftChild(child Node) {
	panic("Cannot set children of an empty leaf node")
}

// SetRightChild is the implementation of Node.SetRightChild
func (eln *EmptyLeafNode) SetRightChild(child Node) {
	panic("Cannot set children of an empty leaf node")
}

// GetLeftChild is the implementation of Node.GetLeftChild
func (eln *EmptyLeafNode) GetLeftChild() Node {
	return nil
}

// GetRightChild is the implementation of Node.GetRightChild
func (eln *EmptyLeafNode) GetRightChild() Node {
	return nil
}

// SetValue is the implementation of Node.SetValue
func (eln *EmptyLeafNode) SetValue(value []byte) {
	panic("Cannot set value of an empty leaf node")
}

// GetValue is the implementation of Node.GetValue
func (eln *EmptyLeafNode) GetValue() []byte {
	return nil
}

// GetKey is the implementation of Node.GetKey
func (eln *EmptyLeafNode) GetKey() []byte {
	return nil
}

// IsEmpty is the implementation of Node.IsEmpty
func (eln *EmptyLeafNode) IsEmpty() bool {
	return true
}

// IsLeaf is the implementation of Node.IsLeaf
func (eln *EmptyLeafNode) IsLeaf() bool {
	return true
}

// IsStub is the implementation of Node.IsStub
func (eln *EmptyLeafNode) IsStub() bool {
	return false
}

// Changed is the implementation of Node.Changed
func (eln *EmptyLeafNode) Changed() bool {
	return eln.changed
}

// MarkChangedAll is the implementation of Node.MarkChangedAll
func (eln *EmptyLeafNode) MarkChangedAll() {
	eln.changed = true
}

// MarkUnchangedAll is the implementation of Node.MarkUnchangedAll
func (eln *EmptyLeafNode) MarkUnchangedAll() {
	eln.changed = false
}

// CountHashesRequiredForGetHash is the implementation of Node.CountHashesRequiredForGetHash
func (eln *EmptyLeafNode) CountHashesRequiredForGetHash() int {
	return 0
}

// NodesInSubtree is the implementation of Node.NodesInSubtree
func (eln *EmptyLeafNode) NodesInSubtree() int {
	return 1
}

// InteriorNodesInSubtree is the implementation of Node.InteriorNodesInSubtree
func (eln *EmptyLeafNode) InteriorNodesInSubtree() int {
	return 0
}

// EmptyLeafNodesInSubtree is the implementation of Node.EmptyLeafNodesInSubtree
func (eln *EmptyLeafNode) EmptyLeafNodesInSubtree() int {
	return 1
}

// NonEmptyLeafNodesInSubtree is the implementation of Node.NonEmptyLeafNodesInSubtree
func (eln *EmptyLeafNode) NonEmptyLeafNodesInSubtree() int {
	return 0
}

// Equals is the implementation of Node.Equals
func (eln *EmptyLeafNode) Equals(n Node) bool {
	_, ok := n.(*EmptyLeafNode)
	return ok
}

func (eln *EmptyLeafNode) ByteSize() int {
	return 1
}

// NewEmptyLeafNodeFromBytes deserializes the passed byteslice into a DictionaryLeafNode
func NewEmptyLeafNodeFromBytes(b []byte) (*EmptyLeafNode, error) {
	return NewEmptyLeafNode()
}

// Bytes is the implementation of Node.Bytes
func (eln *EmptyLeafNode) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(NodeTypeEmptyLeaf))
	return buf.Bytes()
}
