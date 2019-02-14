package fastsha256

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestFastSha256(t *testing.T) {
	test1 := []byte("hello world")
	expectedHash1, _ := hex.DecodeString("b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9")
	test2 := []byte("hello world 2")
	expectedHash2, _ := hex.DecodeString("ed12932f3ef94c0792fbc55263968006e867e522cf9faa88274340a2671d4441")
	test3 := []byte("hello world 3")
	expectedHash3, _ := hex.DecodeString("4ffabbab4e763202462df1f59811944121588f0567f55bce581a0e99ebcf6606")

	calculatedHash1 := Sum256(test1)
	calculatedHash2 := Sum256(test2)
	calculatedHash3 := Sum256(test3)

	if !bytes.Equal(calculatedHash1[:], expectedHash1) {
		t.Error("Mismatching hash 1")
		return
	}

	if !bytes.Equal(calculatedHash2[:], expectedHash2) {
		t.Error("Mismatching hash 2")
		return
	}

	if !bytes.Equal(calculatedHash3[:], expectedHash3) {
		t.Error("Mismatching hash 3")
		return
	}

}
