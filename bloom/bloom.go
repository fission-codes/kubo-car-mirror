package bloom

import (
	"math"

	"github.com/bits-and-blooms/bitset" // TODO: replace with bitset that supports uint64?
	"github.com/zeebo/xxh3"
)

type Filter struct {
	m uint64         // filter size in bits
	k uint64         // hash count
	b *bitset.BitSet // bloom BitSet
}

// New creates a new Bloom filter with _m_ bits and _k_ hashing functions
// We force _m_ and _k_ to be at least one to avoid panics.
func New(m, k uint64) *Filter {
	// TODO: Should we initialize bitset to 0 instead of size?
	return &Filter{max(1, m), max(1, k), bitset.New(uint(m))}
}

// From creates a new Bloom filter with len(_data_) * 64 bits and _k_ hashing
// functions. The data slice is not going to be reset.
func From(data []uint64, k uint64) *Filter {
	// TODO: times 64?
	m := uint64(len(data) * 64)
	return FromWithM(data, m, k)
}

// FromWithM creates a new Bloom filter with _m_ length, _k_ hashing functions.
// The data slice is not going to be reset.
func FromWithM(data []uint64, m, k uint64) *Filter {
	return &Filter{m, k, bitset.From(data)}
}

// Calculations taken from the spec, which in turn were taken from Wikipedia
func EstimateParameters(n uint64, fpp float64) (m, k uint64) {
	m = uint64(math.Ceil(-1 * float64(n) * math.Log(fpp) / math.Pow(math.Log(2), 2)))
	k = uint64(math.Ceil(float64(m) / float64(n) * math.Log(2)))

	return
}

func NewWithEstimates(n uint64, fpp float64) *Filter {
	m, k := EstimateParameters(n, fpp)
	return New(m, k)
}

// M returns the filter size in bits
func (f *Filter) M() uint64 {
	return f.m
}

// K returns the number of hash functions
func (f *Filter) K() uint64 {
	return f.k
}

// B returns the Bloom binary as a `bitset.BitSet`
func (f *Filter) B() *bitset.BitSet {
	return f.b
}

// Sets _k_ bits of the Bloom filter, using the XXH3 hash and _i_ through _k_ as the seed.
func (f *Filter) Add(data []byte) *Filter {
	hasher := NewHasher(f.m, f.k, data)

	for hasher.Next() {
		nextHash := hasher.Value()
		f.b.Set(uint(nextHash))
	}

	return f
}

// Returns true if all k bits of the Bloom filter are set for the specified data.  Otherwise false.
func (f *Filter) Has(data []byte) bool {
	hasher := NewHasher(f.m, f.k, data)

	for hasher.Next() {
		nextHash := hasher.Value()
		if !f.b.Test(uint(nextHash)) {
			return false
		}
	}

	return true
}

func (f *Filter) ApproximatedSize() uint32 {
	x := float64(f.b.Count())
	m := float64(f.M())
	k := float64(f.K())
	size := -1 * m / k * math.Log(1-x/m) / math.Log(math.E)
	return uint32(math.Floor(size + 0.5)) // round
}

// FPP returns the false positive probability rate given n
func (f *Filter) FPP(n uint64) float64 {
	// Taken from https://en.wikipedia.org/wiki/Bloom_filter#Optimal_number_of_hash_functions
	return math.Pow(1-math.Pow(math.E, -(((float64(f.M())/float64(n))*math.Log(2))*(float64(n)/float64(f.M())))), (float64(f.M())/float64(n))*math.Log(2))
}

// Hasher generates k hashes as bit indices for the Bloom filter
// Approach taken from Philipp - https://github.com/matheus23/rust-set-reconciliation/blob/main/src/ibf.rs#L128
type Hasher struct {
	m       uint64
	k       uint64
	data    []byte
	seed    uint64
	count   uint64
	bitmask uint64
}

func NewHasher(m, k uint64, data []byte) *Hasher {
	return &Hasher{
		m:       m,
		k:       k,
		data:    data,
		seed:    0,
		count:   0,
		bitmask: NextPowerOfTwo(m) - 1,
	}
}

// Next returns true if the Hasher has more hashes to generate
func (h *Hasher) Next() bool {
	return h.count < h.k
}

// Value returns the next hash from the Hasher
func (h *Hasher) Value() uint64 {
	var hash uint64

	for {
		hash = xxh3.HashSeed(h.data, h.seed) & h.bitmask
		h.seed += 1

		// Keep the hash if in bounds
		if hash < h.m {
			break
		}
	}

	// Good hash.  Bump hash count and return it.
	h.count += 1

	return hash
}

// NextPowerOfTwo returns _v_ if it is a power of 2, otherwise the next power of two greater than _v_.
func NextPowerOfTwo(v uint64) uint64 {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}

func max(x, y uint64) uint64 {
	if x > y {
		return x
	}
	return y
}
