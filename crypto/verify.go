package crypto

import (
	"fmt"

	"github.com/mit-dci/go-bverify/crypto/btcec"
	"github.com/mit-dci/go-bverify/crypto/fastsha256"
	"github.com/mit-dci/go-bverify/crypto/sig64"
)

var curve *btcec.KoblitzCurve

// VerifySig verifies a signature against a given plaintext and pubKey
func VerifySig(plainText []byte, pubKey [33]byte, csig [64]byte) error {

	sig := sig64.SigDecompress(csig)

	psig, err := btcec.ParseSignature(sig, curve)
	if err != nil {
		return err
	}

	pk, err := btcec.ParsePubKey(pubKey[:], curve)
	if err != nil {
		return err
	}

	hash := fastsha256.Sum256(plainText)

	ok := psig.Verify(hash[:], pk)
	if !ok {
		return fmt.Errorf("Signature verification failed")
	}

	return nil
}

func init() {
	curve = btcec.S256()
}
