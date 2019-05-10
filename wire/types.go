package wire

import (
	"bytes"
	"encoding/binary"

	"github.com/mit-dci/go-bverify/mpt"

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

// ForeignStatement describes the parameters we need to know to follow an arbitrary
// outside statement that's being witnessed on our connected server.
type ForeignStatement struct {
	// Was this the initial statement of this log? If so, the hash calculation
	// differs.
	InitialStatement bool

	// The Log's ID - needed to fetch the proof from the server
	LogID [32]byte

	// This is the actual statement we're trying to prove validity for
	StatementPreimage string

	// This is the signature on the statement
	Signature [64]byte

	// This is the public key that signed the statement
	PubKey [33]byte

	// Index is the sequential index of the statement in the log
	Index uint64

	// Proof is optional, used for historic proofs (we can fetch the current
	// proof from the server if it's meant to keep live).
	Proof *mpt.PartialMPT
}

// Bytes serializes a ForeignStatement object into a byte slice
func (f *ForeignStatement) Bytes() []byte {
	var b bytes.Buffer

	if f.InitialStatement {
		b.Write([]byte{0x01})
	} else {
		b.Write([]byte{0x00})
	}

	b.Write(f.LogID[:])
	b.Write(f.Signature[:])
	b.Write(f.PubKey[:])
	binary.Write(&b, binary.BigEndian, f.Index)
	binary.Write(&b, binary.BigEndian, uint32(len(f.StatementPreimage)))
	b.Write([]byte(f.StatementPreimage))
	if f.Proof == nil {
		binary.Write(&b, binary.BigEndian, uint32(0))
	} else {
		binary.Write(&b, binary.BigEndian, uint32(f.Proof.ByteSize()))
		f.Proof.Serialize(&b)
	}

	return b.Bytes()
}

// ForeignStatementFromBytes deserializes a byte slice into a commitment object
func ForeignStatementFromBytes(b []byte) *ForeignStatement {
	f := ForeignStatement{}
	buf := bytes.NewBuffer(b)

	f.InitialStatement = bytes.Equal(buf.Next(1), []byte{0x01})
	copy(f.LogID[:], buf.Next(32))
	copy(f.Signature[:], buf.Next(64))
	copy(f.PubKey[:], buf.Next(33))

	binary.Read(buf, binary.BigEndian, &f.Index)

	iLen := uint32(0)
	binary.Read(buf, binary.BigEndian, &iLen)
	f.StatementPreimage = string(buf.Next(int(iLen)))

	binary.Read(buf, binary.BigEndian, &iLen)
	if iLen > 0 {
		f.Proof, _ = mpt.DeserializeNewPartialMPT(buf)
	}

	return &f
}
