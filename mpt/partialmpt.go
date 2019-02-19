package mpt

import (
	"bytes"
	"fmt"

	"github.com/mit-dci/go-bverify/utils"
)

// PartialMPT or Partial Merkle Prefix Tries contain a subset of the information
// of the Full Merkle Prefix Trie. The omitted portions are tracked internally
// with "Stubs" which only store the hash of the omitted portion. Because
// they contain a subset of the authentication information,
// the partial Merkle Prefix Trie can only support some operations and
// may not have enough information for others.
type PartialMPT struct {
	root *InteriorNode
}

// NewPartialMPT creates a partial MPT from the full MPT. Since no keys are provided
// this just copies the root.
func NewPartialMPT(fm *FullMPT) (*PartialMPT, error) {
	left, _ := NewStub(fm.root.GetLeftChild().GetHash())
	right, _ := NewStub(fm.root.GetRightChild().GetHash())
	root, _ := NewInteriorNode(left, right)
	return &PartialMPT{root: root}, nil
}

// NewPartialMPTIncludingKey creates a partial MPT from the full MPT such that
// the partial contains specified key mapping
// (if the mapping exists and a path to a leaf if it
// does not) and authentication information from the
// full MPT.
func NewPartialMPTIncludingKey(fm *FullMPT, key []byte) (*PartialMPT, error) {
	// TODO: Assert key length
	root, _ := copyMultiplePaths([][]byte{key}, fm.root, -1)
	return &PartialMPT{root: root.(*InteriorNode)}, nil
}

// NewPartialMPTIncludingKeys creates a partial MPT from the full MPT such that
// the partial contains the specified key mappings
// (if the key exists and a path to a leaf if it does not)
// along with the required authentication information.
func NewPartialMPTIncludingKeys(fm *FullMPT, keys [][]byte) (*PartialMPT, error) {
	// TODO: Assert keys length
	root, _ := copyMultiplePaths(keys, fm.root, -1)
	return &PartialMPT{root: root.(*InteriorNode)}, nil
}

// newPartialMPTWithRoot create a Partial Merkle Prefix Trie with the root. This constructor is private
// because it assumes that the internal structure of root is correct. This is
// not safe to expose to clients.
func newPartialMPTWithRoot(root *InteriorNode) *PartialMPT {
	return &PartialMPT{root: root}
}

func copyMultiplePaths(matchingKeys [][]byte, copyNode Node, currentBitIndex int) (Node, error) {
	// case: if this is not on the path to the key hash
	if len(matchingKeys) == 0 {
		if copyNode.IsEmpty() {
			return NewEmptyLeafNode()
		}
		return NewStub(copyNode.GetHash())
	}

	// case: if this is on the path to a key hash
	// subcase: if we are at the end of a path
	if copyNode.IsLeaf() {
		if copyNode.IsEmpty() {
			return NewEmptyLeafNode()
		}
		return NewDictionaryLeafNodeCachedHash(copyNode.GetKey(), copyNode.GetValue(), copyNode.GetHash())
	}

	// subcase: intermediate node
	// divide up keys into those that match the right prefix (...1)
	// and those that match the left prefix (...0)
	matchLeft := make([][]byte, 0)
	matchRight := make([][]byte, 0)
	for _, key := range matchingKeys {
		bit := utils.GetBit(key, uint(currentBitIndex+1))
		if bit {
			matchRight = append(matchRight, key)
		} else {
			matchLeft = append(matchLeft, key)
		}
	}
	leftChild, _ := copyMultiplePaths(matchLeft, copyNode.GetLeftChild(), currentBitIndex+1)
	rightChild, _ := copyMultiplePaths(matchRight, copyNode.GetRightChild(), currentBitIndex+1)
	return NewInteriorNode(leftChild, rightChild)
}

// Get gets the value mapped to by key or null if the
// key is not mapped to anything.
// @param key - a fixed length byte array representing the key
// (e.g. the hash of some other string)
func (pm *PartialMPT) Get(key []byte) ([]byte, error) {
	// TODO: Assert correct key size?
	return partialGetHelper(pm.root, key, -1)
}

func partialGetHelper(currentNode Node, key []byte, currentBitIndex int) ([]byte, error) {
	if currentNode.IsStub() {
		return nil, fmt.Errorf("Stub encountered")
	}
	if currentNode.IsLeaf() {
		if !currentNode.IsEmpty() {
			// if the current node is NonEmpty and matches the Key
			if bytes.Equal(currentNode.GetKey(), key) {
				return currentNode.GetValue(), nil
			}
		}
		// otherwise key not in the MPT - return null;
		return nil, nil
	}
	bit := utils.GetBit(key, uint(currentBitIndex+1))
	if bit {
		return partialGetHelper(currentNode.GetRightChild(), key, currentBitIndex+1)
	}
	return partialGetHelper(currentNode.GetLeftChild(), key, currentBitIndex+1)
}

// Commitment gets a small cryptographic commitment to the authenticated
// dictionary. For any given set of (key,value) mappings,
// regardless of the order they inserted the commitment
// will be the same and it is computationally
// infeasible to find a different set of (key, value) mappings
// with the same commitment.
func (pm *PartialMPT) Commitment() []byte {
	return pm.root.GetHash()
}

// ProcessUpdatesFromBytes updates the authenticated dictionary
// to reflect changes from the passed byte slice (of a serialized PartialMPT).
// This will change the commitment as mappings have been inserted or removed
func (pm *PartialMPT) ProcessUpdatesFromBytes(b []byte) error {
	pm2, err := NewPartialMPTFromBytes(b)
	if err != nil {
		return err
	}
	return pm.ProcessUpdates(pm2)
}

// ProcessUpdates updates the authenticated dictionary to reflect changes from
// the passed PartialMPT. This will change the commitment as mappings have
// been inserted or removed
func (pm *PartialMPT) ProcessUpdates(pm2 *PartialMPT) error {
	newRoot, _ := UpdateNode(pm.root, pm2.root)
	pm.root = newRoot.(*InteriorNode)
	return nil
}

func (pm *PartialMPT) ByteSize() int {
	return pm.root.ByteSize()
}

// Bytes serializes the PartialMPT into a byte slice
func (pm *PartialMPT) Bytes() []byte {
	return pm.root.Bytes()
}

// NewPartialMPTFromBytes parses a byte slice into a Partial MPT
func NewPartialMPTFromBytes(b []byte) (*PartialMPT, error) {
	possibleRoot, err := NodeFromBytes(b)
	if err != nil {
		return nil, err
	}

	in, ok := possibleRoot.(*InteriorNode)
	if !ok {
		return nil, fmt.Errorf("The passed byte array is no valid tree")
	}

	// TODO: Should check if there's stub nodes in the tree we deserialized
	return newPartialMPTWithRoot(in), nil
}
