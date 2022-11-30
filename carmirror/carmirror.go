package carmirror

import (
	"context"
	"fmt"
	"net/http"

	ipld "github.com/ipfs/go-ipld-format"
	golog "github.com/ipfs/go-log"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	options "github.com/ipfs/interface-go-ipfs-core/options"
)

const Version = "0.1.0"

var log = golog.Logger("kubo-car-mirror")

type CarMirror struct {
	// CAR Mirror config
	cfg *Config

	// Local node getter
	lng ipld.NodeGetter

	// CoreAPI
	capi coreiface.CoreAPI

	// Local block API
	bapi coreiface.BlockAPI

	// HTTP server accepting CAR Mirror requests
	httpServer *http.Server
}

// Config encapsulates CAR Mirror configuration
type Config struct {
	HTTPRemoteAddr       string
	MaxBlocksPerRound    int64
	MaxBlocksPerColdCall int64
}

// Validate confirms the configuration is valid
func (cfg *Config) Validate() error {
	if cfg.HTTPRemoteAddr == "" {
		return fmt.Errorf("HTTPRemoteAddr is required")
	}

	if cfg.MaxBlocksPerColdCall < 1 {
		return fmt.Errorf("MaxBlocksPerColdCall must be a positive number")
	}

	if cfg.MaxBlocksPerRound < 1 {
		return fmt.Errorf("MaxBlocksPerRound must be a positive number")
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
		cfg:  cfg,
		lng:  localNodes,
		capi: capi,
		bapi: blockStore,
	}

	if cfg.HTTPRemoteAddr != "" {
		m := http.NewServeMux()
		// m.Handle("/dag/push", cm.HTTPRemotePushHandler())
		// m.Handle("/dag/pull", cm.HTTPRemotePullHandler())

		cm.httpServer = &http.Server{
			Addr:    cfg.HTTPRemoteAddr,
			Handler: m,
		}
	}

	return cm, nil
}

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

// NewLocalNodeGetter creates a local (no fetch) NodeGetter from a CoreAPI.
func NewLocalNodeGetter(api coreiface.CoreAPI) (ipld.NodeGetter, error) {
	noFetchBlocks, err := api.WithOptions(options.Api.FetchBlocks(false))
	if err != nil {
		return nil, err
	}
	return noFetchBlocks.Dag(), nil
}
