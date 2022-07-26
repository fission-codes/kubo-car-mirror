package bitset

import (
	"encoding/hex"
	"math"
	"math/bits"
)

type BitSet struct {
	bytes     []byte
	bitsCount uint64
}

const byteSize uint64 = 8

// New creates a new BitSet with the specified number of bits
func New(bitsCount uint64) *BitSet {
	bytesCount := int(math.Ceil(float64(bitsCount) / float64(byteSize)))
	bytes := make([]byte, bytesCount)
	return &BitSet{bytes, bitsCount}
}

// Set sets the bit at the specified index to true
func (b *BitSet) Set(bitsIndex uint64) {
	bytesIndex := bitsIndex / 8
	bitmask := uint8(1) << uint8(bitsIndex%8)
	b.bytes[bytesIndex] |= bitmask
}

// Test returns true if the bit at the specified index is true
func (b *BitSet) Test(bitsIndex uint64) bool {
	bytesIndex := bitsIndex / 8
	bitmask := uint8(1) << uint8(bitsIndex%8)
	return (b.bytes[bytesIndex] & bitmask) > 0
}

// Bytes returns the byte slice containing the BitSet
func (b *BitSet) Bytes() []byte {
	return b.bytes
}

// BitsCount returns the number of bits in the BitSet
func (b *BitSet) BitsCount() uint64 {
	return b.bitsCount
}

// BytesCount returns the number of bytes used to store the BitSet
func (b *BitSet) BytesCount() uint64 {
	return uint64(len(b.bytes))
}

// OnesCount returns the number of bits in the BitSet that are set to 1
func (b *BitSet) OnesCount() uint64 {
	var count uint64 = 0
	for _, bb := range b.Bytes() {
		count += uint64(bits.OnesCount8(bb))
	}
	return count
}

// HexEncode returns the bytes of the BitSet encoded as a hexadecimal string
func (b *BitSet) HexEncode() string {
	// should this be encoded in reverse?
	return hex.EncodeToString(b.bytes)
}
