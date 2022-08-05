package plugin

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	carmirror "github.com/fission-codes/go-car-mirror/carmirror"
	"github.com/fission-codes/go-car-mirror/payload"
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
	// configPath string
	host *carmirror.CarMirror
	// log level, defaults to "info"
	LogLevel string
	// Address CAR Mirror will listen on for commands. This should be local only
	HTTPCommandsAddr string
	// Address CAR Mirror will listen on for performing CAR Mirror
	HTTPRemoteAddr string
	// allow-list of peerIDs to accept DAG pushes
	AllowAddrs []string
}

// NewDsyncPlugin creates a CarMirrorPlugin with some sensible defaults
// at least one address will need to be explicitly added to the AllowAddrs
// list before anyone can push to this node
func NewCarMirrorPlugin() *CarMirrorPlugin {
	// cfgPath, err := configPath()
	// if err != nil {
	// 	panic(err)
	// }

	return &CarMirrorPlugin{
		// configPath:       cfgPath,
		LogLevel: "info",
		// 5001 is ipfs API
		// 5002 for car mirror?
		// Looks like both of these are different
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
	return "0.1.0"
}

func (p *CarMirrorPlugin) Init(env *plugin.Environment) error {
	// TODO: Load config from file
	log.Debugf("%s: Init\n", p.Name())
	return nil
}

func (p *CarMirrorPlugin) Start(capi coreiface.CoreAPI) error {

	log.Debugf("%s: Start\n", p.Name())

	lng, err := carmirror.NewLocalNodeGetter(capi)
	if err != nil {
		return err
	}

	p.host, err = carmirror.New(lng, capi.Block(), func(cfg *carmirror.Config) {
		cfg.HTTPRemoteAddress = p.HTTPRemoteAddr
	})
	if err != nil {
		return err
	}

	if err = p.host.StartRemote(context.Background()); err != nil {
		return err
	}

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
	m.Handle("/dag/push", newPushHandler(p.host))
	m.Handle("/dag/pull", newPullHandler(p.host))
	return http.ListenAndServe(p.HTTPCommandsAddr, m)
}

func newPushHandler(cm *carmirror.CarMirror) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream, err := strconv.ParseBool(r.URL.Query().Get("stream"))
		if err != nil {
			stream = false
		}

		diff := r.URL.Query().Get("diff")

		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			errStr := fmt.Sprintf("Failed to read request body. Error=%v", err.Error())
			http.Error(w, errStr, 500)
			return
		}

		var pushRequest payload.PushRequestor
		if err := payload.CborDecode(body, &pushRequest); err != nil {
			errStr := fmt.Sprintf("Failed to decode CBOR. Error=%v", err.Error())
			http.Error(w, errStr, 500)
			return
		}

		switch r.Method {
		case "POST":
			fmt.Fprintf(w, "/dag/push, stream=%v, diff=%v, request=%v\n", stream, diff, pushRequest)
			log.Debug("push")
		}
	})
}

func newPullHandler(cm *carmirror.CarMirror) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream, err := strconv.ParseBool(r.URL.Query().Get("stream"))
		if err != nil {
			stream = false
		}

		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			errStr := fmt.Sprintf("Failed to read request body. Error=%v", err.Error())
			http.Error(w, errStr, 500)
			return
		}

		var pullRequest payload.PullRequestor
		if err := payload.CborDecode(body, &pullRequest); err != nil {
			errStr := fmt.Sprintf("Failed to decode CBOR. Error=%v", err.Error())
			http.Error(w, errStr, 500)
			return
		}

		switch r.Method {
		case "POST":
			log.Debug("pull")
			fmt.Fprintf(w, "/dag/pull, stream=%v, request=%v\n", stream, pullRequest)
		}
	})
}
