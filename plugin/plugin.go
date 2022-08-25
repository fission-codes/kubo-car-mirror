package plugin

import (
	"context"
	"net/http"
	"os"

	carmirror "github.com/fission-codes/go-car-mirror/carmirror"
	golog "github.com/ipfs/go-log"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	plugin "github.com/ipfs/kubo/plugin"
)

var log = golog.Logger("car-mirror")

// Plugins is an exported list of plugins that will be loaded by kubo
var Plugins = []plugin.Plugin{
	NewCarMirrorPlugin(),
}

// CarMirrorPlugin is an exported struct IPFS will load & work with
type CarMirrorPlugin struct {
	host     *carmirror.CarMirror
	LogLevel string
	// HTTPCommandsAddr is the address CAR Mirror will listen on for local commands, which are application concerns.
	// Defaults to `127.0.0.1:2502`.
	HTTPCommandsAddr string
	// HTTPRemoteAddr is the address CAR Mirror will listen on for remote requests, which are protocol concerns.
	// Defaults to `:2503`.
	HTTPRemoteAddr string
}

// NewCarMirrorPlugin creates a CarMirrorPlugin with some sensible defaults
func NewCarMirrorPlugin() *CarMirrorPlugin {
	return &CarMirrorPlugin{
		LogLevel:         "info",
		HTTPRemoteAddr:   ":2503",
		HTTPCommandsAddr: "127.0.0.1:2502",
	}
}

// assert at compile time that CarMirrorPlugin support the PluginDaemon interface
var _ plugin.PluginDaemon = (*CarMirrorPlugin)(nil)

func (*CarMirrorPlugin) Name() string {
	return "car-mirror"
}

func (*CarMirrorPlugin) Version() string {
	return carmirror.Version
}

func (p *CarMirrorPlugin) Init(env *plugin.Environment) error {
	log.Debugf("Init")
	p.loadConfig(env.Config)

	// Only set default log level if env var isn't set
	if lvl := os.Getenv("GOLOG_LOG_LEVEL"); lvl == "" {
		golog.SetLogLevel("car-mirror", p.LogLevel)
	}

	return nil
}

func (p *CarMirrorPlugin) Start(capi coreiface.CoreAPI) error {
	log.Debugf("Start")

	lng, err := carmirror.NewLocalNodeGetter(capi)
	if err != nil {
		return err
	}

	p.host, err = carmirror.New(lng, capi, capi.Block(), func(cfg *carmirror.Config) {
		cfg.HTTPRemoteAddr = p.HTTPRemoteAddr
	})
	if err != nil {
		return err
	}

	// Start the CAR Mirror protocol server
	if err = p.host.StartRemote(context.Background()); err != nil {
		return err
	}

	// Start the application level server
	go p.listenLocalCommands()

	return nil
}

func (p *CarMirrorPlugin) Close() error {
	log.Debugf("Close")
	return nil
}

func (p *CarMirrorPlugin) listenLocalCommands() error {
	m := http.NewServeMux()
	// The CAR Mirror spec doesn't specify how a user initiates a new session.
	// That is an application concern, not protocol, and we've decided to initiate the request
	// via a request to the endpoints below.  Once a request for a new push or pull session has been received,
	// the running CAR Mirror server can then handle the protocol level concerns.
	m.Handle("/dag/push/new", carmirror.NewPushHandler(p.host))
	m.Handle("/dag/pull/new", carmirror.NewPullHandler(p.host))
	return http.ListenAndServe(p.HTTPCommandsAddr, m)
}

func (p *CarMirrorPlugin) loadConfig(cfg interface{}) {
	if v := getString(cfg, "HTTPRemoteAddr"); v != "" {
		p.HTTPRemoteAddr = v
	}
	if v := getString(cfg, "HTTPCommandsAddr"); v != "" {
		p.HTTPCommandsAddr = v
	}
	if v := getString(cfg, "LogLevel"); v != "" {
		p.LogLevel = v
	}
}

func getString(config interface{}, name string) string {
	if config == nil {
		return ""
	}
	mapIface, ok := config.(map[string]interface{})
	if !ok {
		return ""
	}
	rawValue, ok := mapIface[name]
	if !ok || rawValue == "" {
		return ""
	}
	value, ok := rawValue.(string)
	if !ok {
		return ""
	}
	return value
}
