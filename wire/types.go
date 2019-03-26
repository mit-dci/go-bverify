package wire

import (
	"bytes"
	"encoding/binary"

	"github.com/mit-dci/go-bverify/utils"

	"github.com/mit-dci/go-bverify/bitcoin/chainhash"
)

// Commitment contains all the properties of a commitment the server
// made and can be used to verify the validity of the commitment
type Commitment struct {
	// The hash we're committing
	Commitment [32]byte
	// The hash of the transaction containing our commitment
	TxHash *chainhash.Hash
	// The blockheight on the server at the time of commitment
	TriggeredAtBlockHeight int
	// The hash of the block in which the commitment was included
	IncludedInBlock *chainhash.Hash
	// The merkle proof of inclusion of the commitment transaction in the block
	MerkleProof utils.MerkleProof
	// The full transaction (client can recalculate the hash to ensure it is
	// the right transaction and it actually contains the commitment)
	RawTx []byte
}

// Bytes serializes a commitment object into a byte slice
func (c *Commitment) Bytes() []byte {
	var b bytes.Buffer
	b.Write(c.Commitment[:])
	if c.TxHash != nil {
		b.Write(c.TxHash[:])
	} else {
		b.Write(make([]byte, 32))
	}

	binary.Write(&b, binary.BigEndian, int32(c.TriggeredAtBlockHeight))
	if c.IncludedInBlock != nil {
		b.Write(c.IncludedInBlock[:])
	} else {
		b.Write(make([]byte, 32))
	}
	merkleProofBytes := c.MerkleProof.Bytes()
	binary.Write(&b, binary.BigEndian, int32(len(merkleProofBytes)))
	b.Write(merkleProofBytes)
	b.Write(c.RawTx)
	return b.Bytes()
}

// CommitmentFromBytes deserializes a byte slice into a commitment object
func CommitmentFromBytes(b []byte) *Commitment {
	c := Commitment{}
	buf := bytes.NewBuffer(b)
	copy(c.Commitment[:], buf.Next(32))
	nullBytes := make([]byte, 32)
	txhash := make([]byte, 32)
	copy(txhash, buf.Next(32))
	if !bytes.Equal(nullBytes, txhash) {
		c.TxHash, _ = chainhash.NewHash(txhash)
	}
	var i int32
	binary.Read(buf, binary.BigEndian, &i)
	c.TriggeredAtBlockHeight = int(i)
	includedInBlock := make([]byte, 32)
	copy(includedInBlock, buf.Next(32))
	if !bytes.Equal(nullBytes, includedInBlock) {
		c.IncludedInBlock, _ = chainhash.NewHash(includedInBlock)
	}
	binary.Read(buf, binary.BigEndian, &i)
	c.MerkleProof = utils.NewMerkleProofFromBytes(buf.Next(int(i)))
	c.RawTx = buf.Bytes()
	return &c
}

// NewCommitment is a convenience function for creating a new commitment that isn't
// mined yet (hence the absence of parameters for blockhash)
func NewCommitment(c [32]byte, txhash *chainhash.Hash, rawTx []byte, triggerHeight int) *Commitment {
	return &Commitment{Commitment: c, TxHash: txhash, RawTx: rawTx, TriggeredAtBlockHeight: triggerHeight}
}
