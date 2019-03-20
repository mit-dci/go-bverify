package wire

import (
	"bytes"
	"encoding/binary"

	"github.com/mit-dci/go-bverify/utils"

	"github.com/mit-dci/go-bverify/bitcoin/chainhash"
)

type Commitment struct {
	Commitment             [32]byte
	TxHash                 *chainhash.Hash
	TriggeredAtBlockHeight int
	IncludedInBlock        *chainhash.Hash
	MerkleProof            utils.MerkleProof
	RawTx                  []byte
}

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

func NewCommitment(c [32]byte, txhash *chainhash.Hash, rawTx []byte, triggerHeight int) *Commitment {
	return &Commitment{Commitment: c, TxHash: txhash, RawTx: rawTx, TriggeredAtBlockHeight: triggerHeight}
}
