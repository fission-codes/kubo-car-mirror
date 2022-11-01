package cm

import (
	"testing"

	blocksutil "github.com/ipfs/go-ipfs-blocksutil"
	"gotest.tools/assert"
)

func TestMemoroyIpldBlockStore(t *testing.T) {
	bs := NewMemoryIpldBlockStore()

	gen := blocksutil.NewBlockGenerator()
	b1 := gen.Next()

	// Added
	ib1 := NewIpldBlock(b1.Cid(), b1.RawData(), nil)
	bs.Put(ib1)
	assert.Assert(t, bs.Has(ib1.Cid()))
	assert.Equal(t, bs.Get(ib1.Cid()), ib1)

	// Not added
	b2 := gen.Next()
	ib2 := NewIpldBlock(b2.Cid(), b1.RawData(), nil)
	assert.Assert(t, !bs.Has(ib2.Cid()))
}
