package bloom

import (
	"math"

	"github.com/fission-codes/go-car-mirror/util"
	"github.com/zeebo/xxh3"
)

// Hasher generates hashCount hashes as bit indices for the Bloom filter.
type Hasher struct {
	bitCount  uint64 // number of bits we need to index into
	hashCount uint64 // number of hash function calls that result in a bit being set
	data      []byte // the data sent through the hash function
	seed      uint64 // seed passed into the hash function, which starts at 0
	count     uint64 // number of hash function calls so far
	bitmask   uint64 // used for bitwise-AND to generate an index from the hash
}

// NewHasher returns a new Hasher
func NewHasher(bitCount, hashCount uint64, data []byte) *Hasher {
	return &Hasher{
		bitCount:  bitCount,
		hashCount: hashCount,
		data:      data,
		seed:      0,
		count:     0,
		bitmask:   bitmask(bitCount),
	}
}

// Next returns true if the Hasher has more hashes to generate.
func (h *Hasher) Next() bool {
	return h.count < h.hashCount
}

// Value returns the next hash from the Hasher.
func (h *Hasher) Value() uint64 {
	shiftSize := uint64(math.Log2(float64(h.bitmask)))
	var hash, index uint64

	// Attempt to convert hash into a usable index by taking shiftSize right bits and using as an index.
	// If bitCount is not a power of 2, this index may be out of bounds, so cycle through all bits in the
	// 64 bit hash to find an index that is in bounds.  If all bits are exhausted with no viable index,
	// generate a new hash and try again.
	for {
		// Generate hash with current seed
		hash = xxh3.HashSeed(h.data, h.seed)
		h.seed += 1

		for i := uint64(0); i < 64; i += shiftSize {
			index = hash & h.bitmask

			// Keep the index if in bounds.
			// If bitCount is a power of 2, we will always break here and thus avoid rejection sampling.
			if index < h.bitCount {
				// We used the hash to generate a valid index.
				// Bump hash count and return the index.
				h.count += 1
				return index
			}

			// index wasn't in bounds, so shift off the used bits and try again
			hash = hash >> shiftSize
			// fmt.Printf("Shifted: shiftSize=%v, bitmask=%v, i=%v, hash=%b\n", shiftSize, h.bitmask, i, hash)
		}
	}
}

// bitmask returns enough right bits set to 1 such that bitwise-AND with the hash will produce an index
// capable of indexing into all bitCount bits
func bitmask(bitCount uint64) uint64 {
	return util.NextPowerOfTwo(bitCount) - 1
}
