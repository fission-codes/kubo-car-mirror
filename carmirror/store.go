package carmirror

import (
	"context"
	"strings"

	cm "github.com/fission-codes/go-car-mirror/core"
	errors "github.com/fission-codes/go-car-mirror/errors"
	cmipld "github.com/fission-codes/go-car-mirror/ipld"
	kubo "github.com/ipfs/boxo/coreiface"
	opts "github.com/ipfs/boxo/coreiface/options"
	blocks "github.com/ipfs/go-block-format"
	ipld "github.com/ipfs/go-ipld-format"
)

type KuboStore struct {
	store ipld.DAGService
	lng   ipld.NodeGetter
	pins  kubo.PinAPI
}

func NewKuboStore(core kubo.CoreAPI) *KuboStore {
	// TODO: pass errors back to caller instead of panicking
	lng, err := NewLocalNodeGetter(core)
	if err != nil {
		panic(err)
	}

	return &KuboStore{
		store: core.Dag(),
		lng:   lng,
		pins:  core.Pin(),
	}
}

func (ks *KuboStore) Get(ctx context.Context, cid cmipld.Cid) (cm.Block[cmipld.Cid], error) {
	if node, err := ks.lng.Get(ctx, cid.Unwrap()); err != nil {
		// TODO: don't rely on string matching
		if strings.Contains(err.Error(), "block was not found locally (offline)") {
			return nil, errors.ErrBlockNotFound
		}
		return nil, err
	} else {
		return cmipld.WrapBlock(node), nil
	}
}

func (ks *KuboStore) Has(ctx context.Context, cid cmipld.Cid) (bool, error) {
	if _, err := ks.lng.Get(ctx, cid.Unwrap()); err != nil {
		return false, nil
	} else {
		return true, nil
	}
}

func (ks *KuboStore) Add(ctx context.Context, block cm.RawBlock[cmipld.Cid]) (cm.Block[cmipld.Cid], error) {
	var ipfsBlock blocks.Block
	if cmBlock, ok := block.(*cmipld.RawBlock); ok {
		ipfsBlock = cmBlock.Unwrap()
	} else {
		if basicBlock, err := blocks.NewBlockWithCid(block.RawData(), block.Id().Unwrap()); err != nil {
			return nil, err
		} else {
			ipfsBlock = basicBlock
		}
	}
	if node, err := ipld.DefaultBlockDecoder.Decode(ipfsBlock); err != nil {
		return nil, err
	} else {
		if err := ks.store.Add(ctx, node); err != nil {
			return nil, err
		} else {
			return cmipld.WrapBlock(node), nil
		}
	}
}

func (ks *KuboStore) AddMany(ctx context.Context, rawBlocks []cm.RawBlock[cmipld.Cid]) ([]cm.Block[cmipld.Cid], error) {
	var ipfsBlock blocks.Block
	var nodes []ipld.Node
	var blks []cm.Block[cmipld.Cid]
	for _, block := range rawBlocks {
		if cmBlock, ok := block.(*cmipld.RawBlock); ok {
			ipfsBlock = cmBlock.Unwrap()
		} else {
			if basicBlock, err := blocks.NewBlockWithCid(block.RawData(), block.Id().Unwrap()); err != nil {
				return nil, err
			} else {
				ipfsBlock = basicBlock
			}
		}
		if node, err := ipld.DefaultBlockDecoder.Decode(ipfsBlock); err != nil {
			return nil, err
		} else {
			nodes = append(nodes, node)
			blks = append(blks, cmipld.WrapBlock(node))
		}
	}

	if err := ks.store.AddMany(ctx, nodes); err != nil {
		return nil, err
	} else {
		return blks, nil
	}
}

// There doesn't seem to be a clear way to list all the CIDs since the underlying
// blockstore is not exposed in the core Kubo API. This method will therefore list
// the cids of all pinned objects
func (ks *KuboStore) All(ctx context.Context) (<-chan cmipld.Cid, error) {
	if pins, err := ks.pins.Ls(ctx, opts.Pin.Ls.All()); err != nil {
		return nil, err
	} else {
		cids := make(chan cmipld.Cid)
		go func() {
			for pin := range pins {
				if pin.Err() == nil && pin.Path().IsValid() == nil {
					cids <- cmipld.WrapCid(pin.Path().Cid())
				}
			}
		}()
		return cids, nil
	}
}

// NewLocalNodeGetter creates a local (no fetch) NodeGetter from a CoreAPI.
func NewLocalNodeGetter(api kubo.CoreAPI) (ipld.NodeGetter, error) {
	noFetchBlocks, err := api.WithOptions(opts.Api.FetchBlocks(false))
	if err != nil {
		return nil, err
	}
	return noFetchBlocks.Dag(), nil
}
