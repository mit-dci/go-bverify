package mpt

import (
	"bytes"
	"fmt"

	"github.com/mit-dci/go-bverify/crypto"
	"github.com/mit-dci/go-bverify/utils"
	"github.com/mit-dci/go-bverify/wire"
)

// DictionaryLeafNode represents a leaf node in a Merkle Prefix Trie
// (MPT) dictionary. Dictionary leaf nodes store a key and a value,
// both of which are fixed length byte arrays (usually
// the outputs of a cryptographic hash). The value of
// a leaf can be updated.
type DictionaryLeafNode struct {
	key             []byte
	value           []byte
	changed         bool
	commitmentHash  []byte
	recalculateHash bool
}

// Compile time check if DictionaryLeafNode implements Node properly
var _ Node = &DictionaryLeafNode{}

// NewDictionaryLeafNode creates a new dictionary leaf node
func NewDictionaryLeafNode(key, value []byte) (*DictionaryLeafNode, error) {
	return &DictionaryLeafNode{key: key, value: value, changed: true, recalculateHash: true}, nil
}

// GetHash is the implementation of Node.GetHash
func (dln *DictionaryLeafNode) GetHash() []byte {
	if dln.recalculateHash {
		dln.commitmentHash = crypto.WitnessKeyAndValue(dln.key, dln.value)
		dln.recalculateHash = false
	}
	return utils.CloneByteSlice(dln.commitmentHash)
}

// SetLeftChild is the implementation of Node.SetLeftChild
func (dln *DictionaryLeafNode) SetLeftChild(child Node) {
	panic("Cannot set children of a leaf node")
}

// SetRightChild is the implementation of Node.SetRightChild
func (dln *DictionaryLeafNode) SetRightChild(child Node) {
	panic("Cannot set children of a leaf node")
}

// GetLeftChild is the implementation of Node.GetLeftChild
func (dln *DictionaryLeafNode) GetLeftChild() Node {
	return nil
}

// GetRightChild is the implementation of Node.GetRightChild
func (dln *DictionaryLeafNode) GetRightChild() Node {
	return nil
}

// SetValue is the implementation of Node.SetValue
func (dln *DictionaryLeafNode) SetValue(value []byte) {
	if !bytes.Equal(dln.value, value) {
		dln.value = utils.CloneByteSlice(value)
		dln.changed = true
		dln.recalculateHash = true
	}
}

// GetValue is the implementation of Node.GetValue
func (dln *DictionaryLeafNode) GetValue() []byte {
	return utils.CloneByteSlice(dln.value)
}

// GetKey is the implementation of Node.GetKey
func (dln *DictionaryLeafNode) GetKey() []byte {
	return utils.CloneByteSlice(dln.key)
}

// IsEmpty is the implementation of Node.IsEmpty
func (dln *DictionaryLeafNode) IsEmpty() bool {
	return false
}

// IsLeaf is the implementation of Node.IsLeaf
func (dln *DictionaryLeafNode) IsLeaf() bool {
	return true
}

// IsStub is the implementation of Node.IsStub
func (dln *DictionaryLeafNode) IsStub() bool {
	return false
}

// Changed is the implementation of Node.Changed
func (dln *DictionaryLeafNode) Changed() bool {
	return dln.changed
}

// MarkChangedAll is the implementation of Node.MarkChangedAll
func (dln *DictionaryLeafNode) MarkChangedAll() {
	dln.changed = true
}

// MarkUnchangedAll is the implementation of Node.MarkUnchangedAll
func (dln *DictionaryLeafNode) MarkUnchangedAll() {
	dln.changed = false
}

// CountHashesRequiredForGetHash is the implementation of Node.CountHashesRequiredForGetHash
func (dln *DictionaryLeafNode) CountHashesRequiredForGetHash() int {
	if dln.recalculateHash {
		return 1
	}
	return 0
}

// NodesInSubtree is the implementation of Node.NodesInSubtree
func (dln *DictionaryLeafNode) NodesInSubtree() int {
	return 1
}

// InteriorNodesInSubtree is the implementation of Node.InteriorNodesInSubtree
func (dln *DictionaryLeafNode) InteriorNodesInSubtree() int {
	return 0
}

// EmptyLeafNodesInSubtree is the implementation of Node.EmptyLeafNodesInSubtree
func (dln *DictionaryLeafNode) EmptyLeafNodesInSubtree() int {
	return 0
}

// NonEmptyLeafNodesInSubtree is the implementation of Node.NonEmptyLeafNodesInSubtree
func (dln *DictionaryLeafNode) NonEmptyLeafNodesInSubtree() int {
	return 1
}

// Equals is the implementation of Node.Equals
func (dln *DictionaryLeafNode) Equals(n Node) bool {
	dln2, ok := n.(*DictionaryLeafNode)
	if ok {
		return bytes.Equal(dln2.GetKey(), dln.GetKey()) &&
			bytes.Equal(dln2.GetValue(), dln.GetValue())

	}
	return false
}

// NewDictionaryLeafNodeFromBytes deserializes the passed byteslice into a DictionaryLeafNode
func NewDictionaryLeafNodeFromBytes(b []byte) (*DictionaryLeafNode, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("Need at least 1 byte")
	}
	buf := bytes.NewBuffer(b[1:]) // Lob off type byte
	key, err := wire.ReadVarBytes(buf, 256, "key")
	if err != nil {
		return nil, err
	}
	if len(key) == 0 {
		return nil, fmt.Errorf("Key should be at least 1 byte")
	}
	value, err := wire.ReadVarBytes(buf, 256, "value")
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, fmt.Errorf("Value should be at least 1 byte")
	}
	return NewDictionaryLeafNode(key, value)
}

// Bytes is the implementation of Node.Bytes
func (dln *DictionaryLeafNode) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(NodeTypeDictionaryLeaf))
	wire.WriteVarBytes(&buf, dln.key)
	wire.WriteVarBytes(&buf, dln.value)
	return buf.Bytes()
}
