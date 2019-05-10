package mpt

import (
	"bytes"
	"fmt"
	"io"

	"github.com/mit-dci/go-bverify/utils"
)

// FullMPT is a Full Merkle Prefix Trie. This stores
// all mappings and authentication information.
//
// Internally it contains NO STUBs and each node
// tracks if it has been changed. Tracking changes
// allow for lazy recalculation of hashes and
// to keep track of updates.
//
// MPT use structural equality
type FullMPT struct {
	root *InteriorNode
}

// NewFullMPT creates an empty Merkle Prefix Trie
func NewFullMPT() (*FullMPT, error) {
	left, _ := NewEmptyLeafNode()
	right, _ := NewEmptyLeafNode()
	root, _ := NewInteriorNode(left, right)
	return &FullMPT{root: root}, nil
}

// newFullMPTWithRoot create a Merkle Prefix Trie with the root. This constructor is private
// because it assumes that the internal structure of root is correct. This is
// not safe to expose to clients.
func newFullMPTWithRoot(root *InteriorNode) *FullMPT {
	return &FullMPT{root: root}
}

// Insert inserts a (key,value) mapping into the dictionary.
// If the key is currently mapped to some other value,
// the value is updated. Reinserting a (key, value) mapping
// that already is in the dictionary still counts as a change
// made to the dictionary.
// Authentication information
// is updated//lazily* - meaning that calculation
// of hashes is delayed until this.commitment()
// is called!
//
// Additionally the dictionary records all insertions
// as changes and tracks which nodes have been changed
// for the purpose of calculating updates.
//
// key - a fixed length byte array representing the key
// (e.g. the hash of some other string)
// value - a fixed length byte array representing the value
// (e.g. the hash of some other string)
//
func (fm *FullMPT) Insert(key, value []byte) {
	// TODO Assert lengths
	insertHelper(key, value, -1, fm.root)
}

func (fm *FullMPT) Dispose() {
	fm.root.Dispose()
	fm = nil
}

func insertHelper(key, value []byte, currentBitIndex int, currentNode Node) (Node, error) {
	if currentNode.IsLeaf() {
		if bytes.Equal(currentNode.GetKey(), key) {
			// this key is already in the tree, update existing mappings
			currentNode.SetValue(value)
			return currentNode, nil
		}

		// If the key is not in the tree, add it
		nodeToAdd, _ := NewDictionaryLeafNode(key, value)
		if currentNode.IsEmpty() {
			// If the current leaf is empty, just replace it
			return nodeToAdd, nil
		}
		// Otherwise we need to split
		currentNode.MarkChangedAll()
		return split(currentNode.(*DictionaryLeafNode), nodeToAdd, currentBitIndex)
	}
	bit := utils.GetBit(key, uint(currentBitIndex+1))
	if bit {
		newRightChild, _ := insertHelper(key, value, currentBitIndex+1, currentNode.GetRightChild())
		currentNode.SetRightChild(newRightChild)
		return currentNode, nil
	}
	newLeftChild, _ := insertHelper(key, value, currentBitIndex+1, currentNode.GetLeftChild())
	currentNode.SetLeftChild(newLeftChild)
	return currentNode, nil

}

func split(a, b *DictionaryLeafNode, currentBitIndex int) (Node, error) {
	bitA := utils.GetBit(a.GetKey(), uint(currentBitIndex+1))
	bitB := utils.GetBit(b.GetKey(), uint(currentBitIndex+1))
	// Still collision, split again
	if bitA == bitB {
		// Recursively split
		res, _ := split(a, b, currentBitIndex+1)
		empty, _ := NewEmptyLeafNode()

		if bitA {
			return NewInteriorNode(empty, res)
		}
		return NewInteriorNode(res, empty)
	}
	// no collision
	if bitA {
		return NewInteriorNode(b, a)
	}
	return NewInteriorNode(a, b)
}

// Get gets the value mapped to by key or null if the
// key is not mapped to anything.
// @param key - a fixed length byte array representing the key
// (e.g. the hash of some other string)
func (fm *FullMPT) Get(key []byte) []byte {
	// TODO: Assert correct key size?
	return getHelper(fm.root, key, -1)
}

func getHelper(currentNode Node, key []byte, currentBitIndex int) []byte {
	if currentNode.IsLeaf() {
		if !currentNode.IsEmpty() {
			// if the current node is NonEmpty and matches the Key
			if bytes.Equal(currentNode.GetKey(), key) {
				return currentNode.GetValue()
			}
		}
		// otherwise key not in the MPT - return null;
		return nil
	}
	bit := utils.GetBit(key, uint(currentBitIndex+1))
	if bit {
		return getHelper(currentNode.GetRightChild(), key, currentBitIndex+1)
	}
	return getHelper(currentNode.GetLeftChild(), key, currentBitIndex+1)
}

// Delete removes the key and its associated mapping,
// if it exists, from the dictionary.
//
// Additionally the dictionary records all deletions
// as changes and tracks which nodes have been changed
// for the purpose of calculating updates.
// @param key - a fixed length byte array representing the key
// (e.g. the hash of some other string)
func (fm *FullMPT) Delete(key []byte) {
	// TODO: Assert correct key size?
	deleteHelper(key, -1, fm.root, true)
}

func deleteHelper(key []byte, currentBitIndex int, currentNode Node, isRoot bool) (Node, error) {
	if currentNode.IsLeaf() {
		if !currentNode.IsEmpty() {
			if bytes.Equal(currentNode.GetKey(), key) {
				return NewEmptyLeafNode()
			}
		}
		// otherwise the key is not in the tree and nothing needs to be done
		return currentNode, nil
	}

	// we have to watch out to make sure that if this is the root node
	// that we return an InteriorNode and don't propagate up an empty node
	bit := utils.GetBit(key, uint(currentBitIndex+1))
	leftChild := currentNode.GetLeftChild()
	rightChild := currentNode.GetRightChild()
	if bit {
		// delete key from the right subtree
		newRightChild, _ := deleteHelper(key, currentBitIndex+1, rightChild, false)
		// if left subtree is empty, and rightChild is leaf
		// we push the newRightChild back up the MPT
		if leftChild.IsEmpty() && newRightChild.IsLeaf() && !isRoot {
			return newRightChild, nil
		}

		// if newRightChild is empty, and leftChild is a leaf
		// we push the leftChild back up the MPT
		if newRightChild.IsEmpty() && leftChild.IsLeaf() && !isRoot {
			// we also mark the left subtree as changed
			// since its entire position has changed
			leftChild.MarkChangedAll()
			return leftChild, nil
		}
		// otherwise just update current (interior) node's
		// right child
		currentNode.SetRightChild(newRightChild)
		return currentNode, nil
	}
	newLeftChild, _ := deleteHelper(key, currentBitIndex+1, leftChild, false)
	if rightChild.IsEmpty() && newLeftChild.IsLeaf() && !isRoot {
		return newLeftChild, nil
	}
	if newLeftChild.IsEmpty() && rightChild.IsLeaf() && !isRoot {
		rightChild.MarkChangedAll()
		return rightChild, nil
	}
	currentNode.SetLeftChild(newLeftChild)
	return currentNode, nil
}

// Commitment gets a small cryptographic commitment to the authenticated
// dictionary. For any given set of (key,value) mappings,
// regardless of the order they inserted the commitment
// will be the same and it is computationally
// infeasible to find a different set of (key, value) mappings
// with the same commitment.
func (fm *FullMPT) Commitment() []byte {

	return fm.root.GetHash()
}

// Reset resets the current state of the authenticated dictionary
// to have no changes. Changes all nodes
// currently marked as "changed" to "unchanged"
func (fm *FullMPT) Reset() {
	fm.root.MarkUnchangedAll()
}

// MaxHeight returns the height of the tree. Height is defined as the maximum possible
// distance from the leaf to the root node (TODO: I'm not sure this should be a
// public method - only really useful for benchmarking purposes)
func (fm *FullMPT) MaxHeight() int {
	return GetNodeHeight(fm.root)
}

// CountNodes returns the total number of nodes
// in the MPT
func (fm *FullMPT) CountNodes() int {
	return fm.root.NodesInSubtree()
}

// CountInteriorNodes returns the total number of interior nodes
// in the MPT
func (fm *FullMPT) CountInteriorNodes() int {
	return fm.root.InteriorNodesInSubtree()
}

// CountEmptyLeafNodes returns the total number of
// empty nodes in the MPT
func (fm *FullMPT) CountEmptyLeafNodes() int {
	return fm.root.EmptyLeafNodesInSubtree()
}

// Size  returns the number of distinct (key,value) entries
// in the dictionary.
func (fm *FullMPT) Size() int {
	return fm.root.NonEmptyLeafNodesInSubtree()
}

// ByteSize returns the size of Bytes() without actually serializing
func (fm *FullMPT) ByteSize() int {
	return fm.root.ByteSize()
}

func (fm *FullMPT) Bytes() []byte {
	b := make([]byte, fm.ByteSize())
	buf := bytes.NewBuffer(b)
	fm.Serialize(buf)
	return b
}

// Bytes serializes the FullMPT into a byte slice
func (fm *FullMPT) Serialize(w io.Writer) {
	fm.root.Serialize(w)
}

// NewFullMPTFromBytes parses a byte slice into a Full MPT
func DeserializeNewFullMPT(r io.Reader) (*FullMPT, error) {
	possibleRoot, err := DeserializeNode(r)
	if err != nil {
		return nil, err
	}

	in, ok := possibleRoot.(*InteriorNode)
	if !ok {
		return nil, fmt.Errorf("The passed byte array is no valid tree")
	}

	// TODO: Should check if there's stub nodes in the tree we deserialized
	return newFullMPTWithRoot(in), nil
}

func (fm *FullMPT) Copy() (*FullMPT, error) {
	r, w := io.Pipe()
	go func() {
		fm.Serialize(w)
	}()

	return DeserializeNewFullMPT(r)
}

func (fm *FullMPT) Graph() []byte {
	var buf bytes.Buffer
	buf.Write([]byte("digraph fmpt {\n"))
	fm.root.WriteGraphNodes(&buf)
	buf.Write([]byte("\n}\n"))
	return buf.Bytes()
}
