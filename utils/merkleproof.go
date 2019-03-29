package utils

import (
	"bytes"
	"encoding/binary"

	"github.com/mit-dci/go-bverify/logging"

	"github.com/mit-dci/go-bverify/bitcoin/chainhash"
)

// MerkleProof contains the position of the hash whose inclusion you want to prove
// and then the chain of hashes you need to calculate the root hash
type MerkleProof struct {
	Position uint64
	Hashes   []*chainhash.Hash
}

// Bytes serializes the merkle proof into a byte slice
func (m MerkleProof) Bytes() []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, m.Position)
	for _, p := range m.Hashes {
		buf.Write(p[:])
	}
	return buf.Bytes()
}

// NewMerkleProof generates a MerkleProof from a merkle tree and a index of the
// element you want to prove. The merkle tree is expected to be in the form that
// is returned by BuildMerkleTreeStore
func NewMerkleProof(merkleTree []*chainhash.Hash, idx uint64) MerkleProof {
	logging.Debugf("Creating merkle proof for index [%d] - merkle tree:", idx)
	for i, h := range merkleTree {
		if h == nil {
			logging.Debugf("%03d : nil", i)
		} else {
			logging.Debugf("%03d : %x", i, h[:])
		}
	}

	treeHeight := calcTreeHeight(uint64((len(merkleTree) + 1) / 2))

	logging.Debugf("Treeheight: %d", treeHeight)

	proof := MerkleProof{Position: idx, Hashes: make([]*chainhash.Hash, treeHeight)}
	for i := uint(0); i < treeHeight; i++ {
		if merkleTree[idx^1] == nil {
			logging.Debugf("Adding hash %03d: nil [!!!]", idx^1)
		} else {
			logging.Debugf("Adding hash %03d: %x", idx^1, merkleTree[idx^1][:])
		}
		proof.Hashes[i] = merkleTree[idx^1]

		idx = (idx >> 1) | (1 << treeHeight)
	}
	return proof
}

// NewMerkleProofFromBytes will deserialize a merkle proof from a byte slice
func NewMerkleProofFromBytes(b []byte) MerkleProof {
	m := MerkleProof{}
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &m.Position)
	m.Hashes = make([]*chainhash.Hash, 0)
	for {
		if buf.Len() < 32 {
			break
		}
		hash, _ := chainhash.NewHash(buf.Next(32))
		m.Hashes = append(m.Hashes, hash)
	}
	return m
}

// Check will validate a merkle proof given the hash of the element to prove (hash)
// and the expected root hash (expectedRoot). Will return true when the merkle proof
// is valid, false otherwise.
func (proof MerkleProof) Check(hash, expectedRoot *chainhash.Hash) bool {

	logging.Debugf("Checking merkle proof: [%x] to [%x]", hash[:], expectedRoot[:])

	treeHeight := uint(len(proof.Hashes))

	logging.Debugf("treeHeight is %d", treeHeight)

	hashIdx := proof.Position

	logging.Debugf("Proof position is %d", hashIdx)
	for _, h := range proof.Hashes {
		logging.Debugf("Hash is nil: [%t]", h == nil)
		logging.Debugf("Adding hash [%x]", h[:])
		logging.Debugf("Adding to hash [%x]", hash[:])

		var newHash chainhash.Hash
		if hashIdx&1 == 1 {
			newHash = chainhash.DoubleHashH(append(h[:], hash[:]...))
		} else {
			newHash = chainhash.DoubleHashH(append(hash[:], h[:]...))
		}
		hash = &newHash
		hashIdx = (hashIdx >> 1) | (1 << treeHeight)
	}

	logging.Debugf("Final merkle proof: [%x] vs [%x]", hash[:], expectedRoot[:])

	return bytes.Equal(hash[:], expectedRoot[:])
}

// calcTreeHeight will return the height of a tree with n elements.
func calcTreeHeight(n uint64) (e uint) {
	for ; (1 << e) < n; e++ {
	}
	return
}
