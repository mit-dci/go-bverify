package mpt

import (
	"bytes"
	"fmt"
	"io"

	"github.com/mit-dci/go-bverify/utils"
)

// DeltaMPT tracks the changes to a Merkle Prefix Trie (a delta).
// This delta contains ONLY the changed nodes. Nodes that have
// not been changed are represented as STUBS.
type DeltaMPT struct {
	root *InteriorNode
}

// NewDeltaMPT construct a MerklePrefixTrieDelta from a full MPT. It only copies
// the changes the from the MPT (where changes are defined as any nodes
// altered by inserts or deletes since the last call to Reset())
func NewDeltaMPT(fm *FullMPT) (*DeltaMPT, error) {
	leftChild, _ := copyChangesOnlyHelper(fm.root.GetLeftChild())
	rightChild, _ := copyChangesOnlyHelper(fm.root.GetRightChild())
	root, _ := NewInteriorNode(leftChild, rightChild)
	return &DeltaMPT{root: root}, nil
}

func (dm *DeltaMPT) Dispose() {
	dm.root.Dispose()
	dm = nil
}

// GetUpdatesForKey will, given a specific key, calculate
// the updates that should be sent to a client
// whose authenticated dictionary tracks this key.
//
// To reduce the size of the updates this method
// caches unchanged values on the client and
// avoids retransmitting them.
//
// The client can process this update and
// her view of the authenticated dictionary will
// now reflect the update.
func (dm *DeltaMPT) GetUpdatesForKey(key []byte) (*DeltaMPT, error) {
	return dm.GetUpdatesForKeys([][]byte{key})
}

// GetUpdatesForKeys will, given a set of keys, calculate
// the updates that should be sent to a client
// whose authenticated dictionary tracks these keys.
//
// To reduce the size of the updates this method
// caches unchanged values on the client and
// avoids retransmitting them.
//
// The client can process this update and
// her view of the authenticated dictionary will
// now reflect the update.
func (dm *DeltaMPT) GetUpdatesForKeys(keys [][]byte) (*DeltaMPT, error) {
	root, _ := getUpdatesHelper(keys, dm.root, -1)
	return &DeltaMPT{root: root.(*InteriorNode)}, nil
}

func getUpdatesHelper(keys [][]byte, currentNode Node, currentBitIndex int) (Node, error) {
	if currentNode.IsStub() {
		return nil, nil
	}

	// case: non-stub - this location has changed
	// subcase: no matching keys - value is not needed
	if len(keys) == 0 {
		// if empty, just send empty Node
		if currentNode.IsEmpty() {
			return NewEmptyLeafNode()
		}
		// if non-empty, send stub
		return NewStub(currentNode.GetHash())
	}

	// subcase: have a matching key and at end of path
	if currentNode.IsLeaf() {
		if currentNode.IsEmpty() {
			return NewEmptyLeafNode()
		}
		// if non-empty send entire leaf
		return NewDictionaryLeafNode(currentNode.GetKey(), currentNode.GetValue())
	}

	// subcase: intermediate node
	// divide up keys into those that match the right prefix (...1)
	// and those that match the left prefix (...0)
	matchLeft := make([][]byte, 0)
	matchRight := make([][]byte, 0)
	for _, key := range keys {
		bit := utils.GetBit(key, uint(currentBitIndex+1))
		if bit {
			matchRight = append(matchRight, key)
		} else {
			matchLeft = append(matchLeft, key)
		}
	}
	leftChild, _ := getUpdatesHelper(matchLeft, currentNode.GetLeftChild(), currentBitIndex+1)
	rightChild, _ := getUpdatesHelper(matchRight, currentNode.GetRightChild(), currentBitIndex+1)

	return NewInteriorNode(leftChild, rightChild)
}

func copyChangesOnlyHelper(currentNode Node) (Node, error) {
	if !currentNode.Changed() {
		return NewStub(currentNode.GetHash())
	}
	if currentNode.IsLeaf() {
		if currentNode.IsEmpty() {
			return NewEmptyLeafNode()
		}
		if currentNode.Changed() {
			return NewDictionaryLeafNode(currentNode.GetKey(), currentNode.GetValue())
		}
		// This statement below can never be reached. The node is _always_ changed because
		// otherwise it'd be caught by the above `!currentNode.Changed()` clause.
		// Removing it for bogus notice about it not being covered by a test case
		// (since it cannot)

		// return NewStub(currentNode.GetHash())
	}
	leftChild, _ := copyChangesOnlyHelper(currentNode.GetLeftChild())
	rightChild, _ := copyChangesOnlyHelper(currentNode.GetRightChild())

	return NewInteriorNode(leftChild, rightChild)
}

// ByteSize returns the length of Bytes()
func (dm *DeltaMPT) ByteSize() int {
	return dm.root.ByteSize()
}

// Bytes serializes the DeltaMPT into a byte slice
func (dm *DeltaMPT) Serialize(w io.Writer) {
	dm.root.Serialize(w)
}

func (dm *DeltaMPT) Bytes() []byte {
	b := make([]byte, dm.ByteSize())
	buf := bytes.NewBuffer(b)
	dm.Serialize(buf)
	return b
}

func (dm *DeltaMPT) Copy() (*DeltaMPT, error) {
	copiedRoot, err := dm.root.DeepCopy()
	if err != nil {
		return nil, err
	}
	return &DeltaMPT{root: copiedRoot.(*InteriorNode)}, nil
}

// NewDeltaMPTFromBytes parses a byte slice into a Delta MPT
func DeserializeNewDeltaMPT(r io.Reader) (*DeltaMPT, error) {
	possibleRoot, err := DeserializeNode(r)
	if err != nil {
		return nil, err
	}

	in, ok := possibleRoot.(*InteriorNode)
	if !ok {
		return nil, fmt.Errorf("The passed byte array is no valid tree")
	}

	return &DeltaMPT{root: in}, nil
}
