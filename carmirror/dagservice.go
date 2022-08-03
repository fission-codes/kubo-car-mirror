package carmirror

import (
	ipld "github.com/ipfs/go-ipld-format"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	options "github.com/ipfs/interface-go-ipfs-core/options"
)

// NewLocalNodeGetter creates a local (no fetch) NodeGetter from a CoreAPI.
func NewLocalNodeGetter(api coreiface.CoreAPI) (ipld.NodeGetter, error) {
	noFetchBlocks, err := api.WithOptions(options.Api.FetchBlocks(false))
	if err != nil {
		return nil, err
	}
	return noFetchBlocks.Dag(), nil
}
