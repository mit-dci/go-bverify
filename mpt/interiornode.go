package mpt

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/mit-dci/go-bverify/crypto/fastsha256"
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

// Compile time check if InteriorNode implements Node properly
var _ Node = &InteriorNode{}

// NewInteriorNode creates a new interior node
func NewInteriorNode(leftChild, rightChild Node) (*InteriorNode, error) {
	return &InteriorNode{leftChild: leftChild, rightChild: rightChild, changed: true, hash: make([]byte, 32), recalculateHash: true, hash: nil}, nil
}

// NewInteriorNodeWithCachedHash creates a new interior node with a cached hash.
// This is useful when creating proof trees - since we're just substituting parts
// of the tree with stubs, the resulting hashes are equal. Rehashing is a lot of overhead
func NewInteriorNodeWithCachedHash(leftChild, rightChild Node, hash []byte) (*InteriorNode, error) {
	node := &InteriorNode{leftChild: leftChild, rightChild: rightChild, changed: true, recalculateHash: false, hash: make([]byte, 32)}
	copy(node.hash, hash)
	return node, nil

}

func (i *InteriorNode) Dispose() {
	i.hash = nil
	i.recalculateHash = false
	i.changed = false
	i.leftChild.Dispose()
	i.rightChild.Dispose()
	i = nil
}

func (i *InteriorNode) CalculateAllHashes() {

}

// GetHash is the implementation of Node.GetHash
func (i *InteriorNode) GetHash() []byte {
	if i.recalculateHash {
		payload := make([]byte, 0)
		if i.leftChild != nil {
			payload = append(payload, i.leftChild.GetHash()...)
		}
		if i.rightChild != nil {
			payload = append(payload, i.rightChild.GetHash()...)
		}
		hash := fastsha256.Sum256(payload)
		i.hash = make([]byte, len(hash))
		copy(i.hash, hash[:])
		i.recalculateHash = false
	}
	return i.hash
}

// GetGraphHash is the implementation of Node.GetGraphHash
func (i *InteriorNode) GetGraphHash() []byte {
	return i.GetHash()
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
	total := 1
	if i.leftChild != nil {
		total += i.leftChild.NodesInSubtree()
	}
	if i.rightChild != nil {
		total += i.rightChild.NodesInSubtree()
	}
	return total
}

// InteriorNodesInSubtree is the implementation of Node.InteriorNodesInSubtree
func (i *InteriorNode) InteriorNodesInSubtree() int {
	total := 1
	if i.leftChild != nil {
		total += i.leftChild.InteriorNodesInSubtree()
	}
	if i.rightChild != nil {
		total += i.rightChild.InteriorNodesInSubtree()
	}
	return total
}

// EmptyLeafNodesInSubtree is the implementation of Node.EmptyLeafNodesInSubtree
func (i *InteriorNode) EmptyLeafNodesInSubtree() int {
	total := 0
	if i.leftChild != nil {
		total += i.leftChild.EmptyLeafNodesInSubtree()
	}
	if i.rightChild != nil {
		total += i.rightChild.EmptyLeafNodesInSubtree()
	}
	return total
}

// NonEmptyLeafNodesInSubtree is the implementation of Node.NonEmptyLeafNodesInSubtree
func (i *InteriorNode) NonEmptyLeafNodesInSubtree() int {
	total := 0
	if i.leftChild != nil {
		total += i.leftChild.NonEmptyLeafNodesInSubtree()
	}
	if i.rightChild != nil {
		total += i.rightChild.NonEmptyLeafNodesInSubtree()
	}
	return total
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

// NewInteriorNodeFromBytes deserializes the passed byteslice into a InteriorNode
func DeserializeNewInteriorNode(r io.Reader) (*InteriorNode, error) {
	var err error

	var leftNode, rightNode Node
	iLen := int32(0)
	err = binary.Read(r, binary.BigEndian, &iLen)
	if err != nil {
		return nil, err
	}
	if iLen > 0 {
		leftNode, err = DeserializeNode(r)
		if err != nil {
			return nil, err
		}
	}
	err = binary.Read(r, binary.BigEndian, &iLen)
	if err != nil {
		return nil, err
	}
	if iLen > 0 {
		rightNode, err = DeserializeNode(r)
		if err != nil {
			return nil, err
		}
	}

	return NewInteriorNode(leftNode, rightNode)
}

func (i *InteriorNode) ByteSize() int {
	// 1 (Type) + 4 (leftChild size) + leftChild size + 4 (rightChild size) + rightChildSize
	size := 9
	if i.leftChild != nil {
		size += i.leftChild.ByteSize()
	}
	if i.rightChild != nil {
		size += i.rightChild.ByteSize()
	}
	return size
}

func (i *InteriorNode) Serialize(w io.Writer) {
	w.Write([]byte{byte(NodeTypeInterior)})
	if i.leftChild != nil {
		binary.Write(w, binary.BigEndian, int32(i.leftChild.ByteSize()))
		i.leftChild.Serialize(w)
	} else {
		binary.Write(w, binary.BigEndian, int32(0))
	}

	if i.rightChild != nil {
		binary.Write(w, binary.BigEndian, int32(i.rightChild.ByteSize()))
		i.rightChild.Serialize(w)
	} else {
		binary.Write(w, binary.BigEndian, int32(0))
	}
}

// WriteGraphNodes is the implementation of Node.WriteGraphNodes
func (i *InteriorNode) WriteGraphNodes(w io.Writer) {
	w.Write([]byte(fmt.Sprintf("\"%x\" [\n\tshape=box\n\tstyle=\"filled,dashed\"\n\tcolor=black\n\tfillcolor=gray68];\n", i.GetGraphHash())))
	if i.leftChild != nil {
		i.leftChild.WriteGraphNodes(w)
		w.Write([]byte(fmt.Sprintf("\"%x\" -> \"%x\";\n", i.GetGraphHash(), i.leftChild.GetGraphHash())))
	}
	if i.rightChild != nil {
		i.rightChild.WriteGraphNodes(w)
		w.Write([]byte(fmt.Sprintf("\"%x\" -> \"%x\";\n", i.GetGraphHash(), i.rightChild.GetGraphHash())))
	}

}
