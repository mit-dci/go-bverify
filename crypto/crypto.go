package crypto

import "github.com/mit-dci/go-bverify/crypto/fastsha256"

// WitnessKeyAndValue commits to a key and a value using the following commitment:
// H(key||value)
func WitnessKeyAndValue(key, value [32]byte) [32]byte {
	hasher := fastsha256.New()
	hasher.Write(key[:])
	hasher.Write(value[:])
	hash := [32]byte{}
	copy(hash[:], hasher.Sum(nil))
	hasher = nil
	return hash
}
