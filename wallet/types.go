package wallet

import (
	"bytes"
	"encoding/binary"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

type Utxo struct {
	TxHash   chainhash.Hash
	Outpoint uint32
	Value    uint64
	PkScript []byte
}

func (u Utxo) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(u.TxHash[:])
	binary.Write(&buf, binary.BigEndian, u.Outpoint)
	binary.Write(&buf, binary.BigEndian, u.Value)
	buf.Write(u.PkScript)
	return buf.Bytes()
}

func UtxoFromBytes(b []byte) Utxo {
	buf := bytes.NewBuffer(b)
	u := Utxo{}
	hash, _ := chainhash.NewHash(buf.Next(32))
	u.TxHash = *hash
	binary.Read(buf, binary.BigEndian, &u.Outpoint)
	binary.Read(buf, binary.BigEndian, &u.Value)
	copy(u.PkScript, buf.Bytes())
	return u
}

type ChainIndex []*chainhash.Hash

func (activeChain ChainIndex) FindBlock(hash *chainhash.Hash) int {
	for i, b := range activeChain {
		if b.IsEqual(hash) {
			return i
		}
	}

	return -1
}
