package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestWitnessKeyAndValue(t *testing.T) {
	wit := WitnessKeyAndValue([]byte("Hello"), []byte("World"))
	expectedWit, _ := hex.DecodeString("872e4e50ce9990d8b041330c47c9ddd11bec6b503ae9386a99da8584e9bb12c4")
	if !bytes.Equal(wit[:], expectedWit) {
		t.Errorf("Expected witness to be [%x] but found [%x]", expectedWit, wit)
	}
}
