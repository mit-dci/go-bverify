package mpt

import (
	"bytes"
	"encoding/binary"
	"fmt"

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

// NewDictionaryLeafNode creates a new dictionary leaf node with already calculated hash
func NewDictionaryLeafNodeCachedHash(key, value, hash []byte) (*DictionaryLeafNode, error) {
	return &DictionaryLeafNode{key: key, value: value, changed: true, recalculateHash: false, commitmentHash: hash}, nil
}

// GetHash is the implementation of Node.GetHash
func (dln *DictionaryLeafNode) GetHash() []byte {
	if dln.recalculateHash {
		dln.commitmentHash = crypto.WitnessKeyAndValue(dln.key[:], dln.value[:])
		dln.recalculateHash = false
	}
	return dln.commitmentHash
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

// NewDictionaryLeafNodeFromBytes deserializes the passed byteslice into a DictionaryLeafNode
func NewDictionaryLeafNodeFromBytes(b []byte) (*DictionaryLeafNode, error) {
	var key, value []byte
	if len(b) == 0 {
		return nil, fmt.Errorf("Need at least 1 byte")
	}

	buf := bytes.NewBuffer(b[1:]) // lob off type byte

	iLen := int32(0)
	err := binary.Read(buf, binary.BigEndian, &iLen)
	if err != nil {
		return nil, err
	}
	if iLen > 0 {
		if buf.Len() < int(iLen) {
			return nil, fmt.Errorf("Specified length of key not present in buffer")
		}
		key = buf.Next(int(iLen))
	} else {
		return nil, fmt.Errorf("Dictionary leaf node needs a key of at least 1 byte")
	}
	iLen = 0
	err = binary.Read(buf, binary.BigEndian, &iLen)
	if err != nil {
		return nil, err
	}
	if iLen > 0 {
		if buf.Len() < int(iLen) {
			return nil, fmt.Errorf("Specified length of value not present in buffer")
		}
		value = buf.Next(int(iLen))
	} else {
		return nil, fmt.Errorf("Dictionary leaf node needs a value of at least 1 byte")
	}

	return NewDictionaryLeafNode(key, value)
}

// Bytes is the implementation of Node.Bytes
func (dln *DictionaryLeafNode) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(NodeTypeDictionaryLeaf))
	binary.Write(&buf, binary.BigEndian, int32(len(dln.key)))
	buf.Write(dln.key)
	binary.Write(&buf, binary.BigEndian, int32(len(dln.value)))
	buf.Write(dln.value)
	return buf.Bytes()
}

func (dln *DictionaryLeafNode) ByteSize() int {
	return 9 + len(dln.key) + len(dln.value)
}
