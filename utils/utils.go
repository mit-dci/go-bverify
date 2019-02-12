package utils

// CloneByteSlice clones a byte slice and returns the clone
func CloneByteSlice(b []byte) []byte {
	clone := make([]byte, len(b))
	copy(clone[:], b[:])
	return clone
}

// Max returns the larger of x or y.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// Min returns the smaller of x or y.
func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

// GetBit gets the bit at index in a byte array. byte array: byte[0]|| byte[1] ||
// byte[2] || byte[3] index [0...7] [8...15] [16...23] [24...31]
func GetBit(b []byte, idx uint) bool {
	bitIdx := uint(idx % 8)
	byteIdx := (idx - bitIdx) / 8
	return (b[byteIdx] & (1 << (7 - bitIdx))) > 0
}
