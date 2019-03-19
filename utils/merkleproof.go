package utils

import (
	"bytes"
	"encoding/binary"

	"github.com/mit-dci/go-bverify/bitcoin/chainhash"
	"github.com/mit-dci/go-bverify/logging"
)

type MerkleProof struct {
	Position uint64
	Hashes   [][32]byte
}

func (m MerkleProof) Bytes() []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, m.Position)
	for _, p := range m.Hashes {
		buf.Write(p[:])
	}
	return buf.Bytes()
}

func MerkleProofFromBytes(b []byte) MerkleProof {
	m := MerkleProof{}
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &m.Position)
	m.Hashes = make([][32]byte, 0)
	for {
		if buf.Len() < 32 {
			break
		}
		hash32 := [32]byte{}
		copy(hash32[:], buf.Next(32))
		m.Hashes = append(m.Hashes, hash32)
	}
	return m
}

func (proof MerkleProof) Check(startHash []byte, expectedRoot []byte) bool {
	hash := [32]byte{}
	copy(hash[:], startHash)

	treeHeight := uint(len(proof.Hashes))
	hashIdx := proof.Position
	for _, h := range proof.Hashes {
		if hashIdx&1 == 1 {
			hash = chainhash.DoubleHashH(append(h[:], hash[:]...))
		} else {
			hash = chainhash.DoubleHashH(append(hash[:], h[:]...))
		}
		hashIdx = (hashIdx >> 1) | (1 << treeHeight)
	}

	return bytes.Equal(hash[:], expectedRoot[:])
}
