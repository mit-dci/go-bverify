package mpt

import (
	"bytes"
	"fmt"

	"github.com/mit-dci/lit/crypto/fastsha256"
	"github.com/mit-dci/lit/wire"
)

// InteriorNode represents an interior node in the MPT. An interior node has
// two children, a left child and right child. Interior nodes do not store
// keys or values. The hash of the interior node is H(left.getHash()||right.getHash())
// where left.getHash() (resp. right.getHash()) is the hash of the left (resp right)
// child.
//
// The children of the interior node may be changed. Whenever the children are changed
// the node is marked "changed" until reset() is called. Hashes are calculated
// lazily, only when getHash() is called.
type InteriorNode struct {
	hash            []byte
	recalculateHash bool
	changed         bool
	leftChild       Node
	rightChild      Node
}

// Compile time check if DictionaryLeafNode implements Node properly
var _ Node = &InteriorNode{}

// NewInteriorNode creates a new empty leaf node
func NewInteriorNode(leftChild, rightChild Node) (*InteriorNode, error) {
	return &InteriorNode{leftChild: leftChild, rightChild: rightChild, changed: true, recalculateHash: true, hash: make([]byte, 32)}, nil
}

// GetHash is the implementation of Node.GetHash
func (i *InteriorNode) GetHash() []byte {
	if i.recalculateHash {
		leftChildHash := i.leftChild.GetHash()
		rightChildHash := i.rightChild.GetHash()
		commitment := make([]byte, len(leftChildHash)+len(rightChildHash))
		copy(commitment[:], leftChildHash[:])
		copy(commitment[len(leftChildHash):], rightChildHash[:])
		hash := fastsha256.Sum256(commitment)
		copy(i.hash[:], hash[:])
		i.recalculateHash = false
	}
	return i.hash
}

// SetLeftChild is the implementation of Node.SetLeftChild
func (i *InteriorNode) SetLeftChild(child Node) {
	i.leftChild = child
	i.changed = true
	i.recalculateHash = true
}

// SetRightChild is the implementation of Node.SetRightChild
func (i *InteriorNode) SetRightChild(child Node) {
	i.rightChild = child
	i.changed = true
	i.recalculateHash = true
}

// GetLeftChild is the implementation of Node.GetLeftChild
func (i *InteriorNode) GetLeftChild() Node {
	return i.leftChild
}

// GetRightChild is the implementation of Node.GetRightChild
func (i *InteriorNode) GetRightChild() Node {
	return i.rightChild
}

// HasLeft returns true if the left child of this node is not nil
func (i *InteriorNode) HasLeft() bool {
	return i.leftChild != nil
}

// HasRight returns true if the left child of this node is not nil
func (i *InteriorNode) HasRight() bool {
	return i.rightChild != nil
}

// SetValue is the implementation of Node.SetValue
func (i *InteriorNode) SetValue(value []byte) {
	panic("Cannot set value of an interior node")
}

// GetValue is the implementation of Node.GetValue
func (i *InteriorNode) GetValue() []byte {
	return nil
}

// GetKey is the implementation of Node.GetKey
func (i *InteriorNode) GetKey() []byte {
	return nil
}

// IsEmpty is the implementation of Node.IsEmpty
func (i *InteriorNode) IsEmpty() bool {
	return false
}

// IsLeaf is the implementation of Node.IsLeaf
func (i *InteriorNode) IsLeaf() bool {
	return false
}

// IsStub is the implementation of Node.IsStub
func (i *InteriorNode) IsStub() bool {
	return false
}

// Changed is the implementation of Node.Changed
func (i *InteriorNode) Changed() bool {
	return i.changed
}

// MarkChangedAll is the implementation of Node.MarkChangedAll
func (i *InteriorNode) MarkChangedAll() {
	if !i.leftChild.Changed() {
		i.leftChild.MarkChangedAll()
	}
	if !i.rightChild.Changed() {
		i.rightChild.MarkChangedAll()
	}
	i.changed = true
}

// MarkUnchangedAll is the implementation of Node.MarkUnchangedAll
func (i *InteriorNode) MarkUnchangedAll() {
	if i.leftChild.Changed() {
		i.leftChild.MarkUnchangedAll()
	}
	if i.rightChild.Changed() {
		i.rightChild.MarkUnchangedAll()
	}
	i.changed = false
}

// CountHashesRequiredForGetHash is the implementation of Node.CountHashesRequiredForGetHash
func (i *InteriorNode) CountHashesRequiredForGetHash() int {
	if i.recalculateHash {
		total := 1
		total += i.leftChild.CountHashesRequiredForGetHash()
		total += i.rightChild.CountHashesRequiredForGetHash()
		return total
	}
	return 0
}

// NodesInSubtree is the implementation of Node.NodesInSubtree
func (i *InteriorNode) NodesInSubtree() int {
	return 1 + i.rightChild.NodesInSubtree() + i.leftChild.NodesInSubtree()
}

// InteriorNodesInSubtree is the implementation of Node.InteriorNodesInSubtree
func (i *InteriorNode) InteriorNodesInSubtree() int {
	return 1 + i.rightChild.InteriorNodesInSubtree() + i.leftChild.InteriorNodesInSubtree()
}

// EmptyLeafNodesInSubtree is the implementation of Node.EmptyLeafNodesInSubtree
func (i *InteriorNode) EmptyLeafNodesInSubtree() int {
	return i.rightChild.EmptyLeafNodesInSubtree() + i.leftChild.EmptyLeafNodesInSubtree()
}

// NonEmptyLeafNodesInSubtree is the implementation of Node.NonEmptyLeafNodesInSubtree
func (i *InteriorNode) NonEmptyLeafNodesInSubtree() int {
	return i.rightChild.NonEmptyLeafNodesInSubtree() + i.leftChild.NonEmptyLeafNodesInSubtree()
}

// Equals is the implementation of Node.Equals
func (i *InteriorNode) Equals(n Node) bool {
	i2, ok := n.(*InteriorNode)
	if ok {
		if i.leftChild == nil && i2.leftChild != nil {
			return false
		}
		if i.rightChild == nil && i2.rightChild != nil {
			return false
		}
		if i.leftChild != nil && !i.leftChild.Equals(i2.leftChild) {
			return false
		}
		if i.rightChild != nil && !i.rightChild.Equals(i2.rightChild) {
			return false
		}
		return true
	}
	return false
}

// NewInteriorNodeFromBytes deserializes the passed byteslice into a DictionaryLeafNode
func NewInteriorNodeFromBytes(b []byte) (*InteriorNode, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("Need at least one byte in slice")
	}
	buf := bytes.NewBuffer(b[1:]) // Lob off type byte
	left, err := wire.ReadVarBytes(buf, 0, 256, "key")
	if err != nil {
		return nil, err
	}
	right, err := wire.ReadVarBytes(buf, 0, 256, "key")
	if err != nil {
		return nil, err
	}
	var leftNode, rightNode Node
	if len(left) > 0 {
		leftNode, err = NodeFromBytes(left)
		if err != nil {
			return nil, err
		}
	}
	if len(right) > 0 {
		rightNode, err = NodeFromBytes(right)
		if err != nil {
			return nil, err
		}
	}
	return NewInteriorNode(leftNode, rightNode)
}

// Bytes is the implementation of Node.Bytes
func (i *InteriorNode) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(NodeTypeInterior))
	if i.leftChild != nil {
		wire.WriteVarBytes(&buf, 0, i.leftChild.Bytes())
	} else {
		wire.WriteVarBytes(&buf, 0, []byte{})
	}
	if i.rightChild != nil {
		wire.WriteVarBytes(&buf, 0, i.rightChild.Bytes())
	} else {
		wire.WriteVarBytes(&buf, 0, []byte{})
	}
	return buf.Bytes()
}
