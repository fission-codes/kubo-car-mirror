package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fission-codes/go-car-mirror/dag"
	"github.com/ipfs/go-cid"
	ipfs "github.com/ipfs/go-ipfs-http-client"
	golog "github.com/ipfs/go-log"
	mdag "github.com/ipfs/go-merkledag"
	traverse "github.com/ipfs/go-merkledag/traverse"
	iface "github.com/ipfs/interface-go-ipfs-core"
	options "github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

var log = golog.Logger("car-mirror")

type Client struct {
	ctx      context.Context
	api      iface.CoreAPI
	shutdown context.CancelFunc
}

// NewClient returns a client
func NewClient(ctx context.Context) (c *Client, err error) {
	ctx, cancel := context.WithCancel(ctx)

	api, err := ipfs.NewURLApiWithClient("http://localhost:5001", http.DefaultClient)
	if err != nil {
		log.Fatal(err)
	}

	cli := &Client{
		api:      api,
		ctx:      ctx,
		shutdown: cancel,
	}

	// TODO: proper cancelation and context stuff
	// go func() {
	// 	err := operation1(ctx)
	// 	// If this operation returns an error
	// 	// cancel all operations using this context
	// 	if err != nil {
	// 		cancel()
	// 	}
	// }()

	return cli, nil
}

// API returns a CoreAPI
func (c *Client) Api() iface.CoreAPI {
	return c.api
}

// OfflineApi returns an offline CoreAPI
func (c *Client) OfflineApi() iface.CoreAPI {
	api, err := c.api.WithOptions(options.Api.Offline(true))
	if err != nil {
		panic(err)
	}
	return api
}

// GetLocalCids returns a unique list of `cid.CID`s underneath a given root CID, using an offline CoreAPI.
// The root CID is included in the returned list.
// In the case of an error, both the discovered CIDs thus far and the error are returned.
func (c *Client) GetLocalCids(rootCidStr string) ([]cid.Cid, error) {
	var cids []cid.Cid
	rootCid, err := dag.ParseCid(rootCidStr)
	if err != nil {
		return nil, err
	}
	cids = append(cids, *rootCid)

	rp, err := c.OfflineApi().ResolvePath(c.ctx, path.New(rootCidStr))
	if err != nil {
		return cids, err
	}
	nodeGetter := mdag.NewSession(c.ctx, c.OfflineApi().Dag())
	obj, err := nodeGetter.Get(c.ctx, rp.Cid())
	if err != nil {
		return cids, err
	}
	err = traverse.Traverse(obj, traverse.Options{
		DAG:   nodeGetter,
		Order: traverse.DFSPre,
		Func: func(current traverse.State) error {
			cids = append(cids, current.Node.Cid())
			return nil
		},
		ErrFunc:        nil,
		SkipDuplicates: true,
	})
	if err != nil {
		return cids, fmt.Errorf("error traversing DAG: %w", err)
	}

	return cids, nil
}
