package carmirror

import (
	"context"
	"fmt"
	"net/http"

	cmhttp "github.com/fission-codes/go-car-mirror/http"
	cmipld "github.com/fission-codes/go-car-mirror/ipld"
	golog "github.com/ipfs/go-log"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
)

const Version = "0.1.0"

var log = golog.Logger("kubo-car-mirror")

type CarMirror struct {
	// CAR Mirror config
	cfg *Config

	// CoreAPI
	capi coreiface.CoreAPI

	// Block store
	blockStore *KuboStore

	// HTTP client for CAR Mirror requests
	client *cmhttp.Client[cmipld.Cid, *cmipld.Cid]

	// HTTP server for CAR Mirror requests
	server *cmhttp.Server[cmipld.Cid, *cmipld.Cid]

	// HTTP server accepting CAR Mirror requests
	httpServer *http.Server
}

// Config encapsulates CAR Mirror configuration
type Config struct {
	HTTPRemoteAddr string
	MaxBatchSize   uint32
}

// Validate confirms the configuration is valid
func (cfg *Config) Validate() error {
	if cfg.HTTPRemoteAddr == "" {
		return fmt.Errorf("HTTPRemoteAddr is required")
	}

	if cfg.MaxBatchSize < 1 {
		return fmt.Errorf("MaxBatchSize must be a positive number")
	}

	return nil
}

// New creates a local CAR Mirror service.
func New(capi coreiface.CoreAPI, blockStore *KuboStore, opts ...func(cfg *Config)) (*CarMirror, error) {
	// Add default stuff to the config
	cfg := &Config{}

	for _, opt := range opts {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cmConfig := cmhttp.Config{
		MaxBatchSize: cfg.MaxBatchSize,
		Address:      cfg.HTTPRemoteAddr,
	}

	cm := &CarMirror{
		cfg:        cfg,
		capi:       capi,
		blockStore: blockStore,
		client:     cmhttp.NewClient[cmipld.Cid](blockStore, cmConfig),
		server:     cmhttp.NewServer[cmipld.Cid](blockStore, cmConfig),
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
