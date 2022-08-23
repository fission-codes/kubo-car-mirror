package plugin

import (
	"context"
	"net/http"

	carmirror "github.com/fission-codes/go-car-mirror/carmirror"
	golog "github.com/ipfs/go-log"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	plugin "github.com/ipfs/kubo/plugin"
)

var log = golog.Logger("car-mirror-plugin")

// Plugins is an exported list of plugins that will be loaded by go-ipfs
var Plugins = []plugin.Plugin{
	NewCarMirrorPlugin(),
}

// CarMirrorPlugin is exported struct IPFS will load & work with
type CarMirrorPlugin struct {
	// TODO: might need a config file if we have to write to it, like from allow and deny requests
	host *carmirror.CarMirror
	// log level, defaults to "info"
	LogLevel string
	// Address CAR Mirror will listen on for commands. This should be local only
	HTTPCommandsAddr string
	// Address CAR Mirror will listen on for performing CAR Mirror
	HTTPRemoteAddr string
}

// NewCarMirrorPlugin creates a CarMirrorPlugin with some sensible defaults
func NewCarMirrorPlugin() *CarMirrorPlugin {
	return &CarMirrorPlugin{
		LogLevel:         "info",
		HTTPRemoteAddr:   ":2503",          // for requests we expose remotely, which are protocol level concerns
		HTTPCommandsAddr: "127.0.0.1:2502", // for requests we only allow to be initiated locally, which are application level concerns
	}
}

// assert at compile time that CarMirrorPlugin support the PluginDaemon interface
var _ plugin.PluginDaemon = (*CarMirrorPlugin)(nil)

func (*CarMirrorPlugin) Name() string {
	return "car-mirror"
}

func (*CarMirrorPlugin) Version() string {
	return "0.1.0"
}

func (p *CarMirrorPlugin) Init(env *plugin.Environment) error {
	p.loadConfig(env.Config)
	// I don't like this because it overrides env vars.
	// golog.SetLogLevel("car-mirror-plugin", p.LogLevel)
	log.Debugf("%s: Init(%v), env.Config = %v\n", p.Name(), env, env.Config)
	return nil
}

func (p *CarMirrorPlugin) Start(capi coreiface.CoreAPI) error {

	log.Debugf("%s: Start\n", p.Name())

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

	log.Debugf("carmirror plugin started. listening for commands: %s\n", p.HTTPCommandsAddr)
	return nil
}

func (p *CarMirrorPlugin) Close() error {
	log.Debugf("%s: Close\n", p.Name())
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
	log.Debugf("loadConfig: cfg = %v\n", cfg)
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
