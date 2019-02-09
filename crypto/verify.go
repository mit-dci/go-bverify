package crypto

import (
	"fmt"

	"github.com/mit-dci/go-bverify/crypto/fastsha256"
	"github.com/mit-dci/lit/sig64"
	"github.com/mit-dci/zksigma/btcec"
)

var curve *btcec.KoblitzCurve

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
