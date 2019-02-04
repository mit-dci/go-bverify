package mpt

// NodeType is an enum for the various types of nodes we know.
// Used in serialization/deserialization
type NodeType byte

const (
	// NodeTypeStub indicates the node is of type Stub
	NodeTypeStub NodeType = 0x00

	// NodeTypeDictionaryLeaf indicates the node is of type DictionaryLeafNode
	NodeTypeDictionaryLeaf NodeType = 0x01

	// NodeTypeEmptyLeaf indicates the node is of type EmptyLeafNode
	NodeTypeEmptyLeaf NodeType = 0x02

	// NodeTypeInterior indicates the node is of type InteriorNode
	NodeTypeInterior NodeType = 0x03

	// NodeTypeSetLeaf indicates the node is of type SetLeafNode
	NodeTypeSetLeaf NodeType = 0x04
)

// Node is the building blocks of the MPT data structure.
// A node is MUTABLE - the children of interior nodes can change
// and the value stored at a leaf node can change. Nodes track
// these changes and re-calculate hashes accordingly.
type Node interface {
	// GetValue returns the value stored at this node, if it exists. This is only
	// applicable for a non-empty leaf.
	GetValue() []byte

	// SetValue sets the value stored at this node, if it exists. This is only
	// applicable for a non-empty leaf.
	SetValue(value []byte)

	// GetHash returns the hash of this node
	GetHash() []byte

	// CountHashesRequiredForGetHash will count the number of hashes required
	// to (re)calculate the hash of this Node
	CountHashesRequiredForGetHash() int

	// GetKey returns the key of this Node
	GetKey() []byte

	// IsLeaf will return true if this node is a (possibly empty) leaf
	IsLeaf() bool

	// IsEmpty returns true if this node is an empty leaf
	IsEmpty() bool

	// IsStub return true if this node is a stub
	IsStub() bool

	// GetLeftChild returns the left child of this node, if it exists
	// Only applicable if this is an interior node
	GetLeftChild() Node

	// GetRightChild returns the right child of this node, if it exists
	// Only applicable if this is an interior node
	GetRightChild() Node

	// SetLeftChild sets the left child of this node, if possible
	// Only applicable if this is an interior node
	SetLeftChild(child Node)

	// SetRightChild sets the right child of this node, if possible
	// Only applicable if this is an interior node
	SetRightChild(child Node)

	// Changed returns true if this node has been Changed
	Changed() bool

	// MarkChangedAll marks the entire (sub)tree rooted at this node as changed
	MarkChangedAll()

	// MarkUnchangedAll marks the entire (sub)tree rooted at this node as unchanged
	MarkUnchangedAll()

	// Bytes returns a serialized representation of this node
	Bytes() []byte

	// NodesInSubtree returns the number of nodes of any kind in the subtree rooted
	// at this node (includes this node).
	NodesInSubtree() int

	// InteriorNodesInSubtree returns the number of interior nodes in the subtree rooted
	// at this node (includes this node).
	InteriorNodesInSubtree() int

	// EmptyLeafNodesInSubtree returns the number of empty leaf nodes in the subtree rooted
	// at this node (includes this node).
	EmptyLeafNodesInSubtree() int

	// NonEmptyLeafNodesInSubtree returns the number of non-empty leaf nodes in the subtree rooted
	// at this node (includes this node).
	NonEmptyLeafNodesInSubtree() int

	// Returns true if the passed node is equal to the object the method is called on
	Equals(node Node) bool
}

// NodeFromBytes will deserialize the proper node type from a byte slice
func NodeFromBytes(b []byte) (Node, error) {
	if b[0] == byte(NodeTypeStub) {
		return NewStubFromBytes(b)
	}

	if b[0] == byte(NodeTypeDictionaryLeaf) {
		return NewDictionaryLeafNodeFromBytes(b)
	}

	if b[0] == byte(NodeTypeEmptyLeaf) {
		return NewEmptyLeafNodeFromBytes(b)
	}

	if b[0] == byte(NodeTypeInterior) {
		return NewInteriorNodeFromBytes(b)
	}

	if b[0] == byte(NodeTypeSetLeaf) {
		return NewSetLeafNodeFromBytes(b)
	}

	return nil, nil
}
