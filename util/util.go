package util

// NextPowerOfTwo returns i if it is a power of 2, otherwise the next power of two greater than i.
func NextPowerOfTwo(i uint64) uint64 {
	i--
	i |= i >> 1
	i |= i >> 2
	i |= i >> 4
	i |= i >> 8
	i |= i >> 16
	i++
	return i
}
