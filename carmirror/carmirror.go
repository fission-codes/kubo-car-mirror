package carmirror

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fission-codes/go-car-mirror/dag"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	golog "github.com/ipfs/go-log"
	mdag "github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-merkledag/traverse"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	carv1 "github.com/ipld/go-car"
)

var log = golog.Logger("car-mirror")

func init() {
	// golog.SetLogLevel("car-mirror", "debug")
}

const (
	// TODO: Should this be just a string to ditch the dependency on libp2p?  Only used in HTTP header.
	// CarMirrorProtocolID = protocol.ID("/car-mirror/0.1.0")

	// CarMirrorProtocolID is the CAR Mirror p2p Protocol Identifier & version tag
	CarMirrorProtocolID = "/car-mirror/0.1.0"
)

var (
	ErrSomething = fmt.Errorf("something")
)

// or pushable, pullable?
type CarMirrorable interface {
	// NewPushSession()
	// NewPullSession()
	// Push()
	// Pull()
	// Something with protocol verion? or not needed?
}

type CarMirror struct {
	// Local node getter
	lng ipld.NodeGetter

	// CoreAPI
	capi coreiface.CoreAPI

	// Local block API
	bapi coreiface.BlockAPI

	// HTTP server accepting CAR Mirror requests
	httpServer *http.Server

	// bools for various config options?

	// Mutex stuff
	// Session cache?
	sessionTTLDur time.Duration
}

var (
	// compile-time assertion that CarMirror satisfies the remote interface
	_ CarMirrorable = (*CarMirror)(nil)
)

// Config encapsulates optional CAR Mirror configuration
type Config struct {
	// Provide a listening address to have CarMirror spin up an HTTP server when
	// StartRemote(ctx) is called
	HTTPRemoteAddr string
}

// Validate confirms the configuration is valid
func (cfg *Config) Validate() error {
	if cfg.HTTPRemoteAddr == "" {
		return fmt.Errorf("HTTPRemoteAddr is required")
	}

	return nil
}

// New creates a local CAR Mirror service.
//
// Its crucial that the NodeGetter passed to New be an offline-only getter.
// If using IPFS, this package defines a helper function: NewLocalNodeGetter
// to get an offline-only node getter from an IPFS CoreAPI interface.
func New(localNodes ipld.NodeGetter, capi coreiface.CoreAPI, blockStore coreiface.BlockAPI, opts ...func(cfg *Config)) (*CarMirror, error) {
	// Add default stuff to the config
	cfg := &Config{}

	for _, opt := range opts {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cm := &CarMirror{
		lng:  localNodes,
		capi: capi,
		bapi: blockStore,
		// Spec: The Provider MAY garbage collect its session state when it has exhausted its graph, since false positives in the Bloom filter MAY lead to the Provider having an incorrect picture of the Requestor's store. In addition, further requests MAY come in for that session. Session state is an optimization, so treating this as a totally new session is acceptable. However, due to this fact, it is RECOMMENDED that the Provider maintain a session state TTL of at least 30 seconds since the last block is sent. Maintaining this cache for long periods can speed up future requests, so the Provider MAY keep this information around to aid future requests.
		sessionTTLDur: time.Second * 30,
	}

	if cfg.HTTPRemoteAddr != "" {
		m := http.NewServeMux()
		m.Handle("/dag", HTTPRemoteHandler(cm))

		cm.httpServer = &http.Server{
			Addr:    cfg.HTTPRemoteAddr,
			Handler: m,
		}
	}

	return cm, nil
}

// StartRemote makes car mirror available for remote requests, starting an HTTP
// server if a listening address is specified.
// StartRemote returns immediately. Stop remote service by cancelling
// the passed-in context.
func (cm *CarMirror) StartRemote(ctx context.Context) error {
	if cm.httpServer == nil {
		return fmt.Errorf("CAR Mirror is not configured as a remote")
	}

	go func() {
		<-ctx.Done()
		if cm.httpServer != nil {
			cm.httpServer.Close()
		}
	}()

	if cm.httpServer != nil {
		go cm.httpServer.ListenAndServe()
	}

	log.Debug("CAR Mirror remote started")
	return nil
}

// TODO: Add other methods below

func (cm *CarMirror) NewPushSession() {}
func (cm *CarMirror) NewPullSession() {

}

// GetLocalCids returns a unique list of `cid.CID`s underneath a given root CID, using an offline CoreAPI.
// The root CID is included in the returned list.
// In the case of an error, both the discovered CIDs thus far and the error are returned.
func (cm *CarMirror) GetLocalCids(ctx context.Context, rootCidStr string) ([]cid.Cid, error) {
	var cids []cid.Cid
	rootCid, err := dag.ParseCid(rootCidStr)
	if err != nil {
		return nil, err
	}
	cids = append(cids, *rootCid)

	rp, err := cm.capi.ResolvePath(ctx, path.New(rootCidStr))
	if err != nil {
		return cids, err
	}

	nodeGetter := mdag.NewSession(ctx, cm.lng)
	obj, err := nodeGetter.Get(ctx, rp.Cid())
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

func (cm *CarMirror) WriteCar(ctx context.Context, cids []cid.Cid, w io.Writer) error {
	return carv1.WriteCar(ctx, cm.lng, cids, w)
}
