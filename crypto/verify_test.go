package crypto

import (
	"crypto/rand"
	"testing"

	"github.com/mit-dci/go-bverify/crypto/btcec"
	"github.com/mit-dci/go-bverify/crypto/fastsha256"
	"github.com/mit-dci/go-bverify/crypto/sig64"
)

func TestVerify(t *testing.T) {
	key := [32]byte{}
	rand.Read(key[:])
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key[:])
	var pk [33]byte
	copy(pk[:], pub.SerializeCompressed())

	hash := fastsha256.Sum256([]byte("hello world"))
	sig, err := priv.Sign(hash[:])
	if err != nil {
		t.Error(err)
		return
	}
	csig, err := sig64.SigCompress(sig.Serialize())
	if err != nil {
		t.Error(err)
		return
	}

	err = VerifySig([]byte("hello world"), pk, csig)
	if err != nil {
		t.Error(err)
		return
	}

	// damage signature
	damagedCsig := [64]byte{}
	copy(damagedCsig[:], csig[:])
	copy(damagedCsig[1:], []byte{0x00, 0x00, 0x00, 0x00})
	err = VerifySig([]byte("hello world"), pk, damagedCsig)
	if err == nil {
		t.Error("Invalid signature should error out but doesn't")
		return
	}

	// malformed signature
	err = VerifySig([]byte("hello world"), pk, [64]byte{0x00, 0x00, 0x00, 0x00})
	if err == nil {
		t.Error("Malformed signature should error out but doesn't")
		return
	}

	// malformed pubkey
	err = VerifySig([]byte("hello world"), [33]byte{0x00, 0x00, 0x00, 0x00}, csig)
	if err == nil {
		t.Error("Malformed pubkey should error out but doesn't")
		return
	}
}
