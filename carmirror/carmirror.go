package carmirror

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	instrumented "github.com/fission-codes/go-car-mirror/core/instrumented"
	"github.com/fission-codes/go-car-mirror/filter"
	cmhttp "github.com/fission-codes/go-car-mirror/http"
	cmipld "github.com/fission-codes/go-car-mirror/ipld"
	stats "github.com/fission-codes/go-car-mirror/stats"
	gocid "github.com/ipfs/go-cid"
	golog "github.com/ipfs/go-log"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/pkg/errors"
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

	// The block store
	blockStore *KuboStore

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
		MaxBatchSize:  cfg.MaxBatchSize,
		Address:       cfg.HTTPRemoteAddr,
		BloomFunction: HASH_FUNCTION,
		BloomCapacity: 1024,
		// TODO: Make this configurable via config file
		Instrument: instrumented.INSTRUMENT_ORCHESTRATOR | instrumented.INSTRUMENT_STORE | instrumented.INSTRUMENT_FILTER,
	}

	cm := &CarMirror{
		cfg:        cfg,
		capi:       capi,
		blockStore: blockStore,
		client:     cmhttp.NewClient[cmipld.Cid](blockStore, cmConfig),
		server:     cmhttp.NewServer[cmipld.Cid](blockStore, cmConfig),
	}

	return cm, nil
}

func (cm *CarMirror) StartRemote(ctx context.Context) error {
	log.Debugw("enter", "object", "CarMirror", "method", "StartRemote")
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
				WriteError(w, errors.Wrap(err, "failed to parse CID"))
				return
			}

			session := cm.client.GetSourceSession(p.Addr)

			go func() {
				if err := session.Enqueue(cmipld.WrapCid(cid)); err != nil {
					log.Debugw("NewPushSessionHandler", "error", err)
					WriteError(w, err)
					return
				}

				// Close the source
				if err := cm.client.CloseSource(p.Addr); err != nil {
					log.Debugw("NewPushSessionHandler", "error", err)
					WriteError(w, err)
					return
				}
			}()

			if !p.Background {
				select {
				case err := <-session.Done():
					log.Debugw("NewPushSessionHandler", "session", "done")
					if err != nil {
						WriteError(w, err)
					}
					return
				case <-time.After(10 * time.Minute):
					// TODO: Unless we handle timeouts in a different manner, maybe make this default configurable plus overrideable per request
					log.Debugw("NewPushSessionHandler", "session", "timeout")
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
				WriteError(w, errors.Wrap(err, "failed to parse CID"))
				return
			}
			// Initiate the pull
			log.Debugw("before receive", "object", "CarMirror", "method", "NewPullSessionHandler", "cid", cid.String(), "addr", p.Addr)

			session := cm.client.GetSinkSession(p.Addr)

			go func() {
				if err := session.Enqueue(cmipld.WrapCid(cid)); err != nil {
					log.Debugw("NewPullSessionHandler", "error", err)
					WriteError(w, err)
					return
				}

				// Close the sink
				if err := cm.client.CloseSink(p.Addr); err != nil {
					log.Debugw("NewPullSessionHandler", "error", err)
					WriteError(w, err)
					return
				}
			}()

			if !p.Background {
				select {
				case err := <-session.Done():
					log.Debugw("NewPullSessionHandler", "session", "done")
					if err != nil {
						WriteError(w, err)
					}
					return
				case <-time.After(10 * time.Minute):
					// TODO: Unless we handle timeouts in a different manner, maybe make this default configurable plus overrideable per request
					log.Debugw("NewPullSessionHandler", "session", "timeout")
				}
			}
		}
	})
}

// TODO: Any params?
type LsParams struct {
}

type LsResponse struct {
	SessionId   string
	SessionInfo string
}

func (cm *CarMirror) LsHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			p := LsParams{}
			log.Debugw("LsHandler", "params", p)

			sessionMap := make(map[string]LsResponse)
			sessions := make([]LsResponse, 0)

			// Start off with the list of sessions from the stats, so they are returned even if closed.
			// TODO: This is error prone since we're assuming prefixes (i.e. keys) in the snapshot are session ids.
			// Currently that is true, but it may not always be true.
			for _, key := range stats.GLOBAL_REPORTING.Snapshot().Keys() {
				session := strings.Split(key, ".")[0]
				sessionMap[session] = LsResponse{SessionId: session, SessionInfo: "unknown"}
			}

			log.Debugw("LsHandler", "server.sinkSessions", cm.server.SinkSessions())
			sessionTokens := cm.server.SinkSessions()
			for _, sessionToken := range sessionTokens {
				sessionInfo, err := cm.server.SinkInfo(sessionToken)
				if err != nil {
					log.Debugw("LsHandler", "error", err)
					WriteError(w, err)
					return
				}
				sessionMap[string(sessionToken)] = LsResponse{SessionId: string(sessionToken), SessionInfo: sessionInfo.String()}
				log.Debugw("LsHandler", "sessionInfo", sessionInfo)
			}

			log.Debugw("LsHandler", "server.sourceSessions", cm.server.SourceSessions())
			sessionTokens = cm.server.SourceSessions()
			for _, sessionToken := range sessionTokens {
				sessionInfo, err := cm.server.SourceInfo(sessionToken)
				if err != nil {
					log.Debugw("LsHandler", "error", err)
					WriteError(w, err)
					return
				}
				sessionMap[string(sessionToken)] = LsResponse{SessionId: string(sessionToken), SessionInfo: sessionInfo.String()}
				log.Debugw("LsHandler", "sessionInfo", sessionInfo)
			}

			log.Debugw("LsHandler", "client.sinkSessions", cm.client.SinkSessions())
			// TODO: update client.SinkSessions to return a slice of session tokens instead of strings
			clientSessionTokens := cm.client.SinkSessions()
			for _, sessionToken := range clientSessionTokens {
				sessionInfo, err := cm.client.SinkInfo(sessionToken)
				if err != nil {
					log.Debugw("LsHandler", "error", err)
					WriteError(w, err)
					return
				}
				sessionMap[string(sessionToken)] = LsResponse{SessionId: string(sessionToken), SessionInfo: sessionInfo.String()}
				log.Debugw("LsHandler", "sessionInfo", sessionInfo)
			}

			log.Debugw("LsHandler", "client.sourceSessions", cm.client.SourceSessions())
			clientSessionTokens = cm.client.SourceSessions()
			for _, sessionToken := range clientSessionTokens {
				sessionInfo, err := cm.client.SourceInfo(sessionToken)
				if err != nil {
					log.Debugw("LsHandler", "error", err)
					WriteError(w, err)
					return
				}
				sessionMap[string(sessionToken)] = LsResponse{SessionId: string(sessionToken), SessionInfo: sessionInfo.String()}
				log.Debugw("LsHandler", "sessionInfo", sessionInfo)
			}

			for _, session := range sessionMap {
				sessions = append(sessions, session)
			}

			// Write the response
			// TODO: Sort them?
			json.NewEncoder(w).Encode(sessions)
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
						WriteError(w, err)
						return
					}

					WriteSuccess(w)

					return
				}
			}

			for _, sessionToken := range cm.client.SourceSessions() {
				if string(sessionToken) == p.Session {
					if err := cm.client.CloseSource(sessionToken); err != nil {
						log.Debugw("CloseHandler", "error", err)
						WriteError(w, err)
						return
					}

					WriteSuccess(w)
					return
				}
			}

			// If we get here, we didn't find the session
			WriteError(w, fmt.Errorf("session not found"))
		}
	})
}

type CancelParams struct {
	Session string
}

func (cm *CarMirror) CancelHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			p := CancelParams{
				Session: r.FormValue("session"),
			}
			log.Debugw("CancelHandler", "params", p)

			for _, sessionToken := range cm.client.SinkSessions() {
				if string(sessionToken) == p.Session {
					if err := cm.client.CancelSink(sessionToken); err != nil {
						log.Debugw("CancelHandler", "error", err)
						WriteError(w, err)
						return
					}

					WriteSuccess(w)

					return
				}
			}

			for _, sessionToken := range cm.client.SourceSessions() {
				if string(sessionToken) == p.Session {
					if err := cm.client.CancelSource(sessionToken); err != nil {
						log.Debugw("CancelHandler", "error", err)
						WriteError(w, err)
						return
					}

					WriteSuccess(w)
					return
				}
			}

			// If we get here, we didn't find the session
			WriteError(w, fmt.Errorf("session not found"))
		}
	})
}

type StatsParams struct {
	Session string
}

func (cm *CarMirror) StatsHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			p := StatsParams{
				Session: r.FormValue("session"),
			}
			log.Debugw("StatsHandler", "params", p)

			snapshot := stats.GLOBAL_REPORTING.Snapshot()
			if p.Session != "" {
				snapshot = snapshot.Filter(p.Session)
			}

			b, err := snapshot.MarshalJSON()
			if err != nil {
				log.Debugw("StatsHandler", "error", err)
				WriteError(w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}
	})
}

func WriteSuccess(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	res := map[string]string{
		"status": "OK",
	}
	json.NewEncoder(w).Encode(res)
}

func WriteError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	e := map[string]string{
		"error": err.Error(),
	}
	json.NewEncoder(w).Encode(e)
}
