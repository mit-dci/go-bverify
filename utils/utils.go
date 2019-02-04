package utils

// CloneByteSlice clones a byte slice and returns the clone
func CloneByteSlice(b []byte) []byte {
	clone := make([]byte, len(b))
	copy(clone[:], b[:])
	return clone
}
