package cm

// block "github.com/ipfs/go-block-format"

// Block is defined in go-block-format.
//   https://github.com/ipfs/go-block-format/blob/master/blocks.go
// Node is defined in go-ipld-format as Block interface plus other methods, including Links().
//   https://github.com/ipfs/go-ipld-format/blob/master/format.go#L26
// We're combining the two ideas into one?
// Or should we just stick with IPLD as an assumption and use Node?
// Is it that a block's binary data can include links, and IPLD's format includes how what we call links
// are encoded into the bytes in a block?  And technically if you can fit multiple nodes into a single Block you can do that?  i.e. even multiple Nodes that each have Links could be in a single Block?

// TODO: Should this have a Bytes() method, so we can use the raw bytes as keys in maps?
type BlockId interface {
	Bytes() []byte
	String() string
}

// Block is an immutable data block referenced by a unique ID.
//
// What we call a Block, IPLD splits into both Block and Node.
//   https://ipld.io/glossary/#block
//   https://ipld.io/docs/data-model/node/#nodes-vs-blocks
// Node is each thing in a block (e.g. String, Float, Boolean, ...).
// Some Nodes have children (e.g. List, Map).  These can be encoded in a single Block if they fit.
// Some Nodes have children that are other found in other Blocks.  IPLD calls these Links.
//
// For purposes of transmitting blocks, we don't care about a Node that represents items without links
// (e.g. String, ... that fit within a single block).  We only care about a Block and its Links to other Blocks.
// As such, we have collapsed the definition of Node and Block into just Block, and Links are represented as Children on Block.
type Block interface {
	// Id returns the BlockId for the Block.
	Id() BlockId

	// RawBytes returns the bytes associated with the Block.
	RawBytes() []byte

	// Links returns a list of `BlockId`s linked to from the Block.
	Links() []BlockId
}

type ReadableBlockStore interface {
	// Get gets the Block from the store with the given BlockId
	Get(BlockId) Block

	// Has returns true if the blockstore has a block with the given BlockId
	Has(BlockId) bool

	// All returns a lazy iterator over all block IDs in the blockstore.
	// All() iterator
	// AllKeysChan(ctx context.Context) (<-chan BlockId, error)
}

// See https://github.com/ipfs/go-ipfs-blockstore/blob/master/blockstore.go#L33 for comparison.
// Will need context.Context for some methods, so bake that into the API, at a higher level I guess.
type BlockStore interface {
	ReadableBlockStore

	Put(Block)
}

type MutablePointerResolver interface {
	// Resolve attempts to resolve ptr into a BlockId.
	Resolve(ptr string) (id BlockId, err error)
}

type Filter interface {
	// Add adds a BlockId to the Filter.
	Add(id BlockId)

	// Has returns true (sometimes) if Add(BlockId) has been called..
	Has(id BlockId) bool

	// Merge merges two Filters together.
	Merge(other *Filter) *Filter
}

type BlockSender interface {
	Send(Block)
	Flush()
}

type BlockReceiver interface {
	Receive(Block)
}

type StatusSender interface {
	Send(have Filter, want []BlockId)
}

type StatusAccumulator interface {
	Have(BlockId)
	Need(BlockId)
	Receive(BlockId)
	Send(StatusSender)
}

type Orchestrator interface {
	BeginSend()
	EndSend()
	BeginReceipt()
	EndReceipt()
	BeginFlush()
	EndFlush()
}
