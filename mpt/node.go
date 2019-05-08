package mpt

import (
	"fmt"
	"io"

	"github.com/mit-dci/go-bverify/utils"
)

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

	// GetGraphHash returns the hash of this node when presenting in a graph
	// for most node types this'll be equal - but for EmptyLeafNodes this will
	// hash the address of the object to make sure empty leaf nodes have unique
	// hashes
	GetGraphHash() []byte

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

	// ByteSize returns the length of Bytes() without actually serializing first
	ByteSize() int

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

	// Writes a visualization of the node and its children to the writer in DOT format
	WriteGraphNodes(w io.Writer)

	// Dispose sets all used memory to nil and disposes children
	Dispose()
}

// GetNodeHeight returns the height of a node in the MPT, calculated bottom (leaves) up.
func GetNodeHeight(node Node) int {
	if node.IsLeaf() {
		// each leaf is at height zero
		return 0
	}

	return utils.Max(GetNodeHeight(node.GetLeftChild()), GetNodeHeight(node.GetRightChild())) + 1
}

// NodeFromBytes will deserialize the proper node type from a byte slice
func NodeFromBytes(b []byte) (Node, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("Need at least one byte in slice")
	}
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

	return nil, fmt.Errorf("Unknown leaf type %x", b[0])
}

// UpdateNodeFromBytes tries updating from the passed byte slice creating
// new nodes where it's not needed, preserving ones known in n and not in the update.
func UpdateNodeFromBytes(n Node, b []byte) (Node, error) {
	n2, err := NodeFromBytes(b)
	if err != nil {
		return nil, err
	}
	return UpdateNode(n, n2)
}

// UpdateNode tries updating from the passed node creating
// new nodes where it's not needed, preserving ones known in n and not in the update.
func UpdateNode(n, n2 Node) (Node, error) {
	//	var err error
	in, ok := n2.(*InteriorNode)
	if ok {
		var left Node
		var right Node
		if n != nil {
			left = n.GetLeftChild()
			right = n.GetRightChild()
		}
		if in.HasLeft() {
			left, _ = UpdateNode(left, in.GetLeftChild())
			// Error handling only needed when there's actual errors possible
			// Not now, maybe remove error return?
			/*if err != nil {
				return nil, err
			}*/
		}
		if in.HasRight() {
			right, _ = UpdateNode(right, in.GetRightChild())
			// Error handling only needed when there's actual errors possible
			// Not now, maybe remove error return?
			/*if err != nil {
				return nil, err
			}*/
		}
		return NewInteriorNode(left, right)
	}
	return n2, nil
}
