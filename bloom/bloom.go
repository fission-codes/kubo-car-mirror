package bloom

import (
	"math"

	"github.com/fission-codes/go-car-mirror/bitset"
	"github.com/fission-codes/go-car-mirror/util"
)

type Filter struct {
	bitCount  uint64         // filter size in bits
	hashCount uint64         // number of hash functions
	bitSet    *bitset.BitSet // bloom binary
}

// NewFilter returns a new Bloom filter with the specified number of bits and hash functions.
// bitCount and hashCount will be set to 1 if a number less than 1 is provided, to avoid panic.
func NewFilter(bitCount, hashCount uint64) *Filter {
	safeBitCount := max(1, bitCount)
	safeHashCount := max(1, hashCount)
	return &Filter{safeBitCount, safeHashCount, bitset.New(safeBitCount)}
}

// NewFilterFromBloomBytes returns a new Bloom filter with the specified number of bits and hash functions,
// and uses bloomBytes as the initial bytes of the Bloom binary.
// bitCount and hashCount will be set to 1 if a number less than 1 is provided, to avoid panic.
func NewFilterFromBloomBytes(bitCount, hashCount uint64, bloomBytes []byte) *Filter {
	safeBitCount := max(1, bitCount)
	safeHashCount := max(1, hashCount)
	return &Filter{safeBitCount, safeHashCount, bitset.NewFromBytes(safeBitCount, bloomBytes)}
}

// EstimateParameters returns estimates for bitCount and hashCount.
// Calculations are taken from the CAR Mirror spec.
// bitCount will be rounded to the next power of two, as recommended by the spec, to avoid resampling.
func EstimateParameters(n uint64, fpp float64) (bitCount, hashCount uint64) {
	bitCount = util.NextPowerOfTwo(uint64(math.Ceil(-1 * float64(n) * math.Log(fpp) / math.Pow(math.Log(2), 2))))
	hashCount = uint64(math.Ceil(float64(bitCount) / float64(n) * math.Log(2)))
	return
}

// NewFilterWithEstimates returns a new Bloom filter with estimated parameters based on the specified
// number of elements and false positive probability rate.
func NewFilterWithEstimates(n uint64, fpp float64) *Filter {
	m, k := EstimateParameters(n, fpp)
	return NewFilter(m, k)
}

// BitCount returns the filter size in bits.
func (f *Filter) BitCount() uint64 {
	return f.bitCount
}

// HashCount returns the number of hash functions.
func (f *Filter) HashCount() uint64 {
	return f.hashCount
}

// Bytes returns the Bloom binary as a byte slice.
func (f *Filter) Bytes() []byte {
	return f.bitSet.Bytes()
}

// Add sets hashCount bits of the Bloom filter, using the XXH3 hash with a seed.
// The seed starts at 1 and is incremented by 1 until hashCount bits have been set.
// Any hash that is higher than the bit count is thrown away and the seed is incremented by 1 and we try again.
func (f *Filter) Add(data []byte) *Filter {
	hasher := NewHasher(f.bitCount, f.hashCount, data)

	for hasher.Next() {
		nextHash := hasher.Value()
		// fmt.Printf("%v\n", nextHash)
		f.bitSet.Set(uint64(nextHash))
	}

	return f
}

// Returns true if all k bits of the Bloom filter are set for the specified data.  Otherwise false.
func (f *Filter) Test(data []byte) bool {
	hasher := NewHasher(f.bitCount, f.hashCount, data)

	for hasher.Next() {
		nextHash := hasher.Value()
		if !f.bitSet.Test(uint64(nextHash)) {
			return false
		}
	}

	return true
}

// FPP returns the false positive probability rate given the number of elements in the filter.
func (f *Filter) FPP(n uint64) float64 {
	// Taken from https://en.wikipedia.org/wiki/Bloom_filter#Optimal_number_of_hash_functions
	return math.Pow(1-math.Pow(math.E, -(((float64(f.BitCount())/float64(n))*math.Log(2))*(float64(n)/float64(f.BitCount())))), (float64(f.BitCount())/float64(n))*math.Log(2))
}

func max(x, y uint64) uint64 {
	if x > y {
		return x
	}
	return y
}
