package carmirror

import (
	"context"
	"fmt"
	"net/http"
	"time"

	ipld "github.com/ipfs/go-ipld-format"
	golog "github.com/ipfs/go-log"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
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
	HTTPRemoteAddress string
}

// Validate confirms the configuration is valid
func (cfg *Config) Validate() error {
	if cfg.HTTPRemoteAddress == "" {
		return fmt.Errorf("HTTPRemoteAddress is required")
	}

	return nil
}

// New creates a local CAR Mirror service.
//
// Its crucial that the NodeGetter passed to New be an offline-only getter.
// If using IPFS, this package defines a helper function: NewLocalNodeGetter
// to get an offline-only node getter from an IPFS CoreAPI interface.
func New(localNodes ipld.NodeGetter, blockStore coreiface.BlockAPI, opts ...func(cfg *Config)) (*CarMirror, error) {
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
		bapi: blockStore,
		// Spec: The Provider MAY garbage collect its session state when it has exhausted its graph, since false positives in the Bloom filter MAY lead to the Provider having an incorrect picture of the Requestor's store. In addition, further requests MAY come in for that session. Session state is an optimization, so treating this as a totally new session is acceptable. However, due to this fact, it is RECOMMENDED that the Provider maintain a session state TTL of at least 30 seconds since the last block is sent. Maintaining this cache for long periods can speed up future requests, so the Provider MAY keep this information around to aid future requests.
		sessionTTLDur: time.Second * 30,
	}

	if cfg.HTTPRemoteAddress != "" {
		m := http.NewServeMux()
		m.Handle("/dag", HTTPRemoteHandler(cm))

		cm.httpServer = &http.Server{
			Addr:    cfg.HTTPRemoteAddress,
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
