package utils

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestMax(t *testing.T) {
	if Max(10, 11) != 11 {
		t.Error("Max function failed")
	}
	if Max(11, 10) != 11 {
		t.Error("Max function failed")
	}
}

func TestMin(t *testing.T) {
	if Min(10, 11) != 10 {
		t.Error("Min function failed")
	}
	if Min(11, 10) != 10 {
		t.Error("Min function failed")
	}
}

func TestCloneByteSlice(t *testing.T) {
	b, _ := hex.DecodeString("aabbccddeeff")
	b2 := CloneByteSlice(b)

	if &b == &b2 {
		t.Error("Memory addresses of cloned byteslice is equal")
	}

	if !bytes.Equal(b, b2) {
		t.Error("Byteslices have different content")
	}
}

func TestGetBit(t *testing.T) {
	b, _ := hex.DecodeString("aabbccddeeff")
	bits := "101010101011101111001100110111011110111011111111"

	for i, bit := range bits {
		if GetBit(b, uint(i)) != (bit == '1') {
			t.Errorf("Failed at bit %d - expected %t , got %t", i, (bit == '1'), GetBit(b, uint(i)))
		} else {
			t.Logf("Correct bit %d: %t", i, (bit == '1'))
		}
	}
}
