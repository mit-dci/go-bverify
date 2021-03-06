package mpt

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/mit-dci/go-bverify/crypto"
)

// DictionaryLeafNode represents a leaf node in a Merkle Prefix Trie
// (MPT) dictionary. Dictionary leaf nodes store a key and a value,
// both of which are fixed length byte arrays (usually
// the outputs of a cryptographic hash). The value of
// a leaf can be updated.
type DictionaryLeafNode struct {
	key             []byte
	value           []byte
	commitmentHash  []byte
	changed         bool
	recalculateHash bool
}

// Compile time check if DictionaryLeafNode implements Node properly
var _ Node = &DictionaryLeafNode{}

// NewDictionaryLeafNode creates a new dictionary leaf node
func NewDictionaryLeafNode(key, value []byte) (*DictionaryLeafNode, error) {
	node := &DictionaryLeafNode{key: make([]byte, len(key)), value: make([]byte, len(value)), commitmentHash: make([]byte, 32), changed: true, recalculateHash: true}
	copy(node.key, key)
	copy(node.value, value)
	return node, nil
}

func (dln *DictionaryLeafNode) Dispose() {
	dln.key = nil
	dln.value = nil
	dln.commitmentHash = nil

	dln = nil
}

// NewDictionaryLeafNode creates a new dictionary leaf node with already calculated hash
func NewDictionaryLeafNodeCachedHash(key, value, hash []byte) (*DictionaryLeafNode, error) {
	node := &DictionaryLeafNode{key: make([]byte, len(key)), value: make([]byte, len(value)), commitmentHash: make([]byte, 32), changed: true, recalculateHash: false}
	copy(node.key, key)
	copy(node.value, value)
	copy(node.commitmentHash, hash)
	return node, nil
}

// GetHash is the implementation of Node.GetHash
func (dln *DictionaryLeafNode) GetHash() []byte {
	if dln.recalculateHash {
		copy(dln.commitmentHash, crypto.WitnessKeyAndValue(dln.key[:], dln.value[:]))
		dln.recalculateHash = false
	}
	return dln.commitmentHash
}

// GetGraphHash is the implementation of Node.GetGraphHash
func (dln *DictionaryLeafNode) GetGraphHash() []byte {
	return dln.GetHash()
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
		dln.value = value
		dln.changed = true
		dln.recalculateHash = true
	}
}

// GetValue is the implementation of Node.GetValue
func (dln *DictionaryLeafNode) GetValue() []byte {
	return dln.value
}

// GetKey is the implementation of Node.GetKey
func (dln *DictionaryLeafNode) GetKey() []byte {
	return dln.key
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

// DeserializeNewDictionaryLeafNode deserializes a DictionaryLeafNode from the passed in reader
func DeserializeNewDictionaryLeafNode(r io.Reader) (*DictionaryLeafNode, error) {
	var key, value []byte

	iLen := int32(0)
	err := binary.Read(r, binary.BigEndian, &iLen)
	if err != nil {
		return nil, err
	}
	if iLen > 0 {
		key = make([]byte, iLen)
		i, err := r.Read(key)
		if err != nil {
			return nil, err
		}
		if int32(i) != iLen {
			return nil, fmt.Errorf("Specified length of key not present in buffer")
		}
	} else {
		return nil, fmt.Errorf("Dictionary leaf node needs a key of at least 1 byte")
	}
	iLen = 0
	err = binary.Read(r, binary.BigEndian, &iLen)
	if err != nil {
		return nil, err
	}
	if iLen > 0 {
		value = make([]byte, iLen)
		i, err := r.Read(value)
		if err != nil {
			return nil, err
		}
		if int32(i) != iLen {
			return nil, fmt.Errorf("Specified length of value not present in buffer")
		}
	} else {
		return nil, fmt.Errorf("Dictionary leaf node needs a value of at least 1 byte")
	}

	return NewDictionaryLeafNode(key, value)
}

// Bytes is the implementation of Node.Bytes
func (dln *DictionaryLeafNode) Serialize(w io.Writer) {
	w.Write([]byte{byte(NodeTypeDictionaryLeaf)})
	binary.Write(w, binary.BigEndian, int32(len(dln.key)))
	w.Write(dln.key)
	binary.Write(w, binary.BigEndian, int32(len(dln.value)))
	w.Write(dln.value)
}

func (dln *DictionaryLeafNode) ByteSize() int {
	return 9 + len(dln.key) + len(dln.value)
}

func (dln *DictionaryLeafNode) WriteGraphNodes(w io.Writer) {
	w.Write([]byte(fmt.Sprintf("\"%x\" [\n\tshape=box\n\tstyle=\"filled,solid\"\n\tfontcolor=blue\n\tcolor=blue\n\tfillcolor=lightblue];\n", dln.GetGraphHash())))
}

func (dln *DictionaryLeafNode) DeepCopy() (Node, error) {
	return NewDictionaryLeafNodeCachedHash(dln.key, dln.value, dln.commitmentHash)
}
