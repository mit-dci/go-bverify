package mpt

import (
	"fmt"
	"io"

	"github.com/mit-dci/go-bverify/crypto/fastsha256"
)

var emptyLeafNodeHash []byte
var sharedEmptyLeafNode *EmptyLeafNode

// EmptyLeafNode represents an empty leaf in the tree. Empty leaves
// do not have associated values and use the special marker
// hash of all 0s.
//
type EmptyLeafNode struct {
}

// Compile time check if DictionaryLeafNode implements Node properly
var _ Node = &EmptyLeafNode{}

// NewEmptyLeafNode creates a new empty leaf node
func NewEmptyLeafNode() (*EmptyLeafNode, error) {
	return sharedEmptyLeafNode, nil
}

func (eln *EmptyLeafNode) Dispose() {
	eln = nil
}

// GetHash is the implementation of Node.GetHash
func (eln *EmptyLeafNode) GetHash() []byte {
	return emptyLeafNodeHash
}

// GetGraphHash is the implementation of Node.GetGraphHash
func (eln *EmptyLeafNode) GetGraphHash() []byte {
	hash := fastsha256.Sum256([]byte(fmt.Sprintf("%p", eln)))
	returnHash := make([]byte, len(hash))
	copy(returnHash, hash[:])
	return returnHash
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
	return false
}

// MarkChangedAll is the implementation of Node.MarkChangedAll
func (eln *EmptyLeafNode) MarkChangedAll() {

}

// MarkUnchangedAll is the implementation of Node.MarkUnchangedAll
func (eln *EmptyLeafNode) MarkUnchangedAll() {

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

// Bytes is the implementation of Node.Bytes
func (eln *EmptyLeafNode) Serialize(w io.Writer) {
	w.Write([]byte{byte(NodeTypeEmptyLeaf)})
}

func (eln *EmptyLeafNode) WriteGraphNodes(w io.Writer) {
	w.Write([]byte(fmt.Sprintf("\"%x\" [\n\tshape=box\n\tstyle=\"filled,solid\"\n\tfontcolor=gray50\n\tcolor=gray50\n\tfillcolor=white];\n", eln.GetGraphHash())))
}

func (eln *EmptyLeafNode) DeepCopy() (Node, error) {
	return sharedEmptyLeafNode, nil
}

func init() {
	emptyLeafNodeHash = make([]byte, 32)
	sharedEmptyLeafNode = &EmptyLeafNode{}
}
