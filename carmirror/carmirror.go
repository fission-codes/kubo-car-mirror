package carmirror

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	instrumented "github.com/fission-codes/go-car-mirror/core/instrumented"
	"github.com/fission-codes/go-car-mirror/filter"
	cmhttp "github.com/fission-codes/go-car-mirror/http"
	cmipld "github.com/fission-codes/go-car-mirror/ipld"
	gocid "github.com/ipfs/go-cid"
	golog "github.com/ipfs/go-log"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/zeebo/xxh3"
)

const Version = "0.1.0"

var log = golog.Logger("kubo-car-mirror")

const HASH_FUNCTION = 3

func init() {
	filter.RegisterHash(3, XX3HashBlockId)
}

func XX3HashBlockId(id cmipld.Cid, seed uint64) uint64 {
	return xxh3.HashSeed(id.Bytes(), seed)
}

type CarMirror struct {
	// CAR Mirror config
	cfg *Config

	// CoreAPI
	capi coreiface.CoreAPI

	// Client block store
	clientBlockStore *KuboStore

	// Server block store
	serverBlockStore *KuboStore

	// HTTP client for CAR Mirror requests
	client *cmhttp.Client[cmipld.Cid, *cmipld.Cid]

	// HTTP server for CAR Mirror requests
	server *cmhttp.Server[cmipld.Cid, *cmipld.Cid]
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
func New(capi coreiface.CoreAPI, clientBlockStore *KuboStore, serverBlockStore *KuboStore, opts ...func(cfg *Config)) (*CarMirror, error) {
	// Add default stuff to the config
	cfg := &Config{}

	for _, opt := range opts {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cmConfig := cmhttp.Config{
		MaxBatchSize:  cfg.MaxBatchSize,
		Address:       cfg.HTTPRemoteAddr,
		BloomFunction: HASH_FUNCTION,
		BloomCapacity: 1024,
		// TODO: Make this configurable via config file
		Instrument: instrumented.INSTRUMENT_ORCHESTRATOR | instrumented.INSTRUMENT_STORE | instrumented.INSTRUMENT_FILTER,
	}

	cm := &CarMirror{
		cfg:              cfg,
		capi:             capi,
		clientBlockStore: clientBlockStore,
		serverBlockStore: serverBlockStore,
		client:           cmhttp.NewClient[cmipld.Cid](clientBlockStore, cmConfig),
		server:           cmhttp.NewServer[cmipld.Cid](serverBlockStore, cmConfig),
	}

	return cm, nil
}

func (cm *CarMirror) StartRemote(ctx context.Context) error {
	if cm.server == nil {
		return fmt.Errorf("CAR Mirror is not configured as a remote")
	}

	go func() {
		<-ctx.Done()
		if cm.server != nil {
			cm.server.Stop()
		}
	}()

	if cm.server != nil {
		go cm.server.Start()
	}

	log.Debug("CAR Mirror remote started")
	return nil
}

type PushParams struct {
	Cid        string
	Addr       string
	Diff       string
	Stream     bool
	Background bool
}

func (cm *CarMirror) NewPushSessionHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			p := PushParams{
				Cid:        r.FormValue("cid"),
				Addr:       r.FormValue("addr"),
				Diff:       r.FormValue("diff"),
				Stream:     r.FormValue("stream") == "true",
				Background: r.FormValue("background") == "true",
			}
			log.Debugw("NewPushSessionHandler", "params", p)

			// Parse the CID
			cid, err := gocid.Parse(p.Cid)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(err.Error())
				// w.Write([]byte(err.Error()))
				return
			}

			// Initiate the push
			err = cm.client.Send(p.Addr, cmipld.WrapCid(cid))

			if err != nil {
				log.Debugw("NewPushSessionHandler", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(err.Error())
				// w.Write([]byte(err.Error()))
				return
			}

			if !p.Background {
				// TODO: This close is intentionally only being called if the job runs in the background.  It may make
				// more sense to not conflate background with not closing though.
				// Close the session and wait for the other end to close
				cm.client.CloseSource(p.Addr)

				info, err := cm.client.SourceInfo(p.Addr)
				for err == nil {
					log.Debugf("client info: %s", info.String())
					time.Sleep(100 * time.Millisecond)
					info, err = cm.client.SourceInfo(p.Addr)
				}

				if err != cmhttp.ErrInvalidSession {
					log.Debugw("Closed with unexpected error", "error", err)
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(err.Error())
					// w.Write([]byte(err.Error()))
					return
				}
			}
		}
	})
}

type PullParams struct {
	Cid        string
	Addr       string
	Stream     bool
	Background bool
}

func (cm *CarMirror) NewPullSessionHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			p := PullParams{
				Cid:        r.FormValue("cid"),
				Addr:       r.FormValue("addr"),
				Stream:     r.FormValue("stream") == "true",
				Background: r.FormValue("background") == "true",
			}
			log.Debugw("NewPullSessionHandler", "params", p)

			// Parse the CID
			cid, err := gocid.Parse(p.Cid)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(err.Error())
				// w.Write([]byte(err.Error()))
				return
			}

			// Initiate the pull
			err = cm.client.Receive(p.Addr, cmipld.WrapCid(cid))

			if err != nil {
				log.Debugw("NewPullSessionHandler", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(err.Error())
				// w.Write([]byte(err.Error()))
				return
			}

			if !p.Background {
				// Close the session and wait for the other end to close
				cm.client.CloseSink(p.Addr)

				info, err := cm.client.SinkInfo(p.Addr)
				for err == nil {
					log.Debugf("client info: %s", info.String())
					time.Sleep(100 * time.Millisecond)
					info, err = cm.client.SinkInfo(p.Addr)
				}

				if err != cmhttp.ErrInvalidSession {
					log.Debugw("Closed with unexpected error", "error", err)
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(err.Error())
					// w.Write([]byte(err.Error()))
					return
				}
			}
		}
	})
}

type LsParams struct {
}

func (cm *CarMirror) LsHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			p := LsParams{}
			log.Debugw("LsHandler", "params", p)
			sessions := make([]string, 0)

			log.Debugw("LsHandler", "server.sinkSessions", cm.server.SinkSessions())
			sessionTokens := cm.server.SinkSessions()
			for _, sessionToken := range sessionTokens {
				sessions = append(sessions, string(sessionToken))

				sessionInfo, err := cm.server.SinkInfo(sessionToken)
				if err != nil {
					log.Debugw("LsHandler", "error", err)
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(err.Error())
					// w.Write([]byte(err.Error()))
					return
				}
				log.Debugw("LsHandler", "sessionInfo", sessionInfo)
			}

			log.Debugw("LsHandler", "server.sourceSessions", cm.server.SourceSessions())
			sessionTokens = cm.server.SourceSessions()
			for _, sessionToken := range sessionTokens {
				sessions = append(sessions, string(sessionToken))

				sessionInfo, err := cm.server.SourceInfo(sessionToken)
				if err != nil {
					log.Debugw("LsHandler", "error", err)
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(err.Error())
					// w.Write([]byte(err.Error()))
					return
				}
				log.Debugw("LsHandler", "sessionInfo", sessionInfo)
			}

			log.Debugw("LsHandler", "client.sinkSessions", cm.client.SinkSessions())
			// TODO: update client.SinkSessions to return a slice of session tokens instead of strings
			clientSessionTokens := cm.client.SinkSessions()
			for _, sessionToken := range clientSessionTokens {
				sessions = append(sessions, string(sessionToken))

				sessionInfo, err := cm.client.SinkInfo(sessionToken)
				if err != nil {
					log.Debugw("LsHandler", "error", err)
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(err.Error())
					// w.Write([]byte(err.Error()))
					return
				}
				log.Debugw("LsHandler", "sessionInfo", sessionInfo)
			}

			log.Debugw("LsHandler", "client.sourceSessions", cm.client.SourceSessions())
			clientSessionTokens = cm.client.SourceSessions()
			for _, sessionToken := range clientSessionTokens {
				sessions = append(sessions, string(sessionToken))

				sessionInfo, err := cm.client.SourceInfo(sessionToken)
				if err != nil {
					log.Debugw("LsHandler", "error", err)
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(err.Error())
					// w.Write([]byte(err.Error()))
					return
				}
				log.Debugw("LsHandler", "sessionInfo", sessionInfo)
			}

			// Write the response
			json.NewEncoder(w).Encode(sessions)
			// w.WriteHeader(http.StatusOK)
			return
		}
	})
}

type CloseParams struct {
	Session string
}

func (cm *CarMirror) CloseHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			p := CloseParams{
				Session: r.FormValue("session"),
			}
			log.Debugw("CloseHandler", "params", p)

			for _, sessionToken := range cm.client.SinkSessions() {
				if string(sessionToken) == p.Session {
					if err := cm.client.CloseSink(sessionToken); err != nil {
						log.Debugw("CloseHandler", "error", err)
						w.WriteHeader(http.StatusInternalServerError)
						// TODO: encode in JSON
						e := map[string]string{
							"error": err.Error(),
						}
						json.NewEncoder(w).Encode(e)
						// w.Write([]byte(err.Error()))
						return
					}

					w.WriteHeader(http.StatusOK)
					e := map[string]string{
						"status": "OK",
					}
					json.NewEncoder(w).Encode(e)

					return
				}
			}

			for _, sessionToken := range cm.client.SourceSessions() {
				if string(sessionToken) == p.Session {
					if err := cm.client.CloseSource(sessionToken); err != nil {
						log.Debugw("CloseHandler", "error", err)
						w.WriteHeader(http.StatusInternalServerError)
						// TODO: encode in JSON
						e := map[string]string{
							"error": err.Error(),
						}
						json.NewEncoder(w).Encode(e)
						// w.Write([]byte(err.Error()))
						return
					}

					w.WriteHeader(http.StatusOK)
					e := map[string]string{
						"status": "OK",
					}
					json.NewEncoder(w).Encode(e)

					return
				}
			}

		}
	})
}
