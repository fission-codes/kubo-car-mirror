package carmirror

import (
	"context"

	cmipld "github.com/fission-codes/go-car-mirror/ipfs"
	ipld "github.com/ipfs/go-ipld-format"
	ipfs "github.com/ipfs/interface-go-ipfs-core"
	opts "github.com/ipfs/interface-go-ipfs-core/options"
)

type KuboStore struct {
	store ipld.DAGService
	pins  ipfs.PinAPI
}

func (ks *KuboStore) Get(ctx context.Context, cid cmipld.Cid) (*cmipld.Block, error) {
	if node, err := ks.store.Get(ctx, cid.Unwrap()); err != nil {
		return nil, err
	} else {
		return cmipld.WrapBlock(node), nil
	}
}

func (ks *KuboStore) Has(ctx context.Context, cid cmipld.Cid) (bool, error) {
	if _, err := ks.store.Get(ctx, cid.Unwrap()); err != nil {
		return false, nil
	} else {
		return true, nil
	}
}

func (ks *KuboStore) Add(ctx context.Context, block *cmipld.RawBlock) (*cmipld.Block, error) {
	if node, err := ipld.DefaultBlockDecoder.Decode(block); err != nil {
		return nil, err
	} else {
		if err := ks.store.Add(ctx, node); err != nil {
			return nil, err
		} else {
			return cmipld.WrapBlock(node), nil
		}
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
