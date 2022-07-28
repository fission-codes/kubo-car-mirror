package bloom

import (
	"github.com/fission-codes/go-car-mirror/util"
	"github.com/zeebo/xxh3"
)

// Hasher generates hashCount hashes as bit indices for the Bloom filter.
// Approach taken from Philipp - https://github.com/matheus23/rust-set-reconciliation/blob/main/src/ibf.rs#L128.
type Hasher struct {
	bitCount  uint64
	hashCount uint64
	data      []byte
	seed      uint64
	count     uint64
	bitmask   uint64
}

func NewHasher(bitCount, hashCount uint64, data []byte) *Hasher {
	return &Hasher{
		bitCount:  bitCount,
		hashCount: hashCount,
		data:      data,
		seed:      0,
		count:     0,
		bitmask:   util.NextPowerOfTwo(bitCount) - 1,
	}
}

// Next returns true if the Hasher has more hashes to generate.
func (h *Hasher) Next() bool {
	return h.count < h.hashCount
}

// Value returns the next hash from the Hasher.
func (h *Hasher) Value() uint64 {
	var hash uint64

	for {
		hash = xxh3.HashSeed(h.data, h.seed) & h.bitmask
		h.seed += 1

		// Keep the hash if in bounds
		if hash < h.bitCount {
			break
		}
	}

	// Good hash.  Bump hash count and return it.
	h.count += 1

	return hash
}
