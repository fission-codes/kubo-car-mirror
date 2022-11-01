package cm

import (
	"github.com/fission-codes/go-car-mirror/bloom"
	gocid "github.com/ipfs/go-cid"
)

// --- Interface implementations ---

// BlockId
var _ BlockId = (*gocid.Cid)(nil)

// Block
type IpldBlock struct {
	id       gocid.Cid
	rawBytes []byte
	links    []BlockId
}

func NewIpldBlock(id gocid.Cid, rawBytes []byte, links []BlockId) *IpldBlock {
	return &IpldBlock{
		id:       id,
		rawBytes: rawBytes,
		links:    links,
	}
}

func (b *IpldBlock) Id() BlockId {
	return b.id
}

func (b *IpldBlock) RawBytes() []byte {
	return b.rawBytes
}

func (b *IpldBlock) Links() []BlockId {
	return b.links
}

var _ Block = (*IpldBlock)(nil)

// BlockStore
type MemoryIpldBlockStore struct {
	// Mapping from Cid string to IpldBlock
	blocks map[string]*Block
}

func NewMemoryIpldBlockStore() *MemoryIpldBlockStore {
	return &MemoryIpldBlockStore{
		blocks: make(map[string]*Block),
	}
}

func (bs *MemoryIpldBlockStore) Get(cid BlockId) Block {
	return *bs.blocks[cid.String()]
}

func (bs *MemoryIpldBlockStore) Has(cid BlockId) bool {
	_, ok := bs.blocks[cid.String()]
	return ok
}

func (bs *MemoryIpldBlockStore) Put(block Block) {
	cid := block.Id().String()
	bs.blocks[cid] = &block
}

var _ BlockStore = (*MemoryIpldBlockStore)(nil)

// MutablePointerResolver

// type IpfsMutablePointerResolver struct { ... }
// func (mpr *...) Resolve(ptr string) (id BlockId, err error) {}

// var _ MutablePointerResolver = (...)(nil)

// Filter

type BloomFilter struct {
	filter *bloom.Filter
}

// TODO: Add New* methods to mirror those in bloom.Filter

func (f *BloomFilter) Add(id BlockId) {
	f.filter.Add(id.Bytes())
}

func (f *BloomFilter) Has(id BlockId) bool {
	return f.filter.Test(id.Bytes())
}

func (f *BloomFilter) Merge(other *Filter) *BloomFilter {
	// TODO: Merge bloom filters together
	return f
}

// var _ Filter = (*BloomFilter)(nil)

// BlockSender

// BlockReceiver

// StatusSender

// StatusAccumulator

// Orchestrator

// --- Structs and their methods not left up to implementors ---

// SenderSession

type SenderSession struct {
	blockSender  BlockSender
	orchestrator Orchestrator
	filter       Filter
	sentCids     []BlockId // change to cid set if we don't need order
	// is peer needed?  Or is this global, with peerId as a key for sentCids, like spec says
}

func (ss *SenderSession) SendBlock(BlockId)                         {}
func (ss *SenderSession) HandleStatus(have *Filter, wanted BlockId) {}
func (ss *SenderSession) Flush()                                    {}

// ReceiverSession

type ReceiverSession struct {
	statusAccumulator StatusAccumulator
	statusSender      StatusSender
	orchestrator      Orchestrator
}

func (rs *ReceiverSession) HandleBlock(Block)        {}
func (rs *ReceiverSession) AccumulateStatus(BlockId) {}
func (rs *ReceiverSession) Flush()                   {}

// SimpleStatusAccumulator

// --- Round based implementations ---

// ListSender

// ListReceiver

// BatchSendOrchestrator

// BatchBlockSender

// BatchReceiveOrchestrator

// BatchBlockReceiver
