package cm

// block "github.com/ipfs/go-block-format"

// BlockId represents a unique identifier for a Block.
// This interface only represents the identifier, not the Block.
type BlockId interface {
	// Bytes returns the BlockId as a byte slice.
	Bytes() []byte

	// String returns the BlockId as a string.
	// This is useful when the BlockId must be represented as a string (e.g. when used as a key in a map).
	String() string
}

// Block is an immutable data block referenced by a unique ID.
//
// What we call a Block, IPLD splits into both Block and Node.
//
//   Block
//     https://ipld.io/glossary/#block
//     https://github.com/ipfs/go-block-format/blob/master/blocks.go#L20
//
//   Node
//     https://ipld.io/glossary/#node
//     https://ipld.io/docs/data-model/node/#node
//     https://github.com/ipfs/go-ipld-format/blob/master/format.go#L26
//
//   Blocks vs Nodes
//     https://ipld.io/docs/data-model/node/#nodes-vs-blocks
//
// Node is the term used for each piece of data in a block (e.g. String, Float, Boolean, ...).
// Some Nodes have children that can be stored in a single block if they fit (e.g. for items in a list if the list is small).
// Other Nodes have children that are stored in other Blocks, either because they don't fit in one block or to take advantage of content IDs and deduplication.
// IPLD calls these content identified children `Links`.
//
// For purposes of transmitting blocks, we don't care about a child nested within the same block.
// We only care about a Block and its Links to other Blocks, in order to ensure that all nested blocks are transmitted.
// As such, we have collapsed the definition of Node and Block into just Block, and the block's links are represented as `Links` on the Block.
type Block interface {
	// Id returns the BlockId for the Block.
	Id() BlockId

	// RawBytes returns the bytes associated with the Block.
	RawBytes() []byte

	// Links returns a list of `BlockId`s linked to from the Block.
	Links() []BlockId
}

// ReadableBlockStore represents read operations for a store of blocks.
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

// BlockIdFilter is anything similar to a bloom filter that can efficiently (and without perfect accuracy) keep track of a list of `BlockId`s.
type BlockIdFilter interface {
	// Add adds a BlockId to the Filter.
	Add(id BlockId)

	// Has returns true (sometimes) if Add(BlockId) has been called..
	Has(id BlockId) bool

	// Merge merges two Filters together.
	Merge(other *BlockIdFilter) *BlockIdFilter

	// TODO: Does this need extra methods related to its sizing, saturation, etc?
}

// BlockSender is responsible for sending blocks - immediately and asynchronously, or via a buffer.
// The details are up to the implementor.
type BlockSender interface {
	Send(Block)
	Flush()
}

// BlockReceiver is responsible for receiving blocks.
type BlockReceiver interface {
	// Receive is called on receipt of a new block.
	Receive(Block)
}

// StatusAccumulator is responsible for collecting status.
type StatusAccumulator interface {
	Have(BlockId)
	Need(BlockId)
	Receive(BlockId)
	Send(StatusSender)
}

// StatusSender is responsible for sending status.
// The key intuition of CAR Mirror is that status can be sent efficiently using a lossy filter.
// The StatusSender will therefore usually batch reported information and send it in bulk to the ReceiverSession.
type StatusSender interface {
	Send(have BlockIdFilter, want []BlockId)
}

// StatusReceiver is responsible for receiving a status.
type StatusReceiver interface {
	HandleStatus(have BlockIdFilter, want []BlockId)
}

// Orchestrator is responsible for managing the flow of blocks and/or status.
type Orchestrator interface {
	BeginSend()
	EndSend()
	BeginReceipt()
	EndReceipt()
	BeginFlush()
	EndFlush()
}
