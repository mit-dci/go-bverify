package crypto

import "github.com/mit-dci/go-bverify/crypto/fastsha256"

// WitnessKeyAndValue commits to a key and a value using the following commitment:
// H(key||value)
func WitnessKeyAndValue(key, value []byte) []byte {
	data := make([]byte, len(key)+len(value))
	copy(data[:], key)
	copy(data[len(key):], value)
	hash := fastsha256.Sum256(data)
	data = make([]byte, len(hash))
	copy(data[:], hash[:])
	return data
}
