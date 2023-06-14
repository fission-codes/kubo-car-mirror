![](https://github.com/fission-codes/kubo-car-mirror/raw/master/assets/logo.png?sanitize=true)

# kubo-car-mirror

[![CI](https://github.com/fission-codes/kubo-car-mirror/actions/workflows/main.yml/badge.svg)](https://github.com/fission-codes/kubo-car-mirror/actions/workflows/main.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/fission-codes/blob/master/LICENSE)
[![Built by FISSION](https://img.shields.io/badge/⌘-Built_by_FISSION-purple.svg)](https://fission.codes)
[![Discord](https://img.shields.io/discord/478735028319158273.svg)](https://discord.gg/zAQBDEq)
[![Discourse](https://img.shields.io/discourse/https/talk.fission.codes/topics)](https://talk.fission.codes)

🚧 WIP 🚧

Go implementation of [CAR Mirror](https://github.com/fission-codes/spec/tree/main/car-pool).

Presentation at IPFS Thing 2023, by @walkah https://www.youtube.com/watch?v=UeSb7vC0K7Y

## Building

kubo-car-mirror is implemented as a [Kubo Daemon Plugin](https://github.com/ipfs/kubo/blob/master/docs/plugins.md#daemon).
In order to avoid package versioning issues common with Go plugins, it needs to be built in-tree with Kubo.

We provide some make targets to simplify this process.

```
# Clone repos to same parent directory
git clone https://github.com/fission-codes/kubo-car-mirror
git clone https://github.com/ipfs/kubo

# Change to the kubo-car-mirror directory to run make targets
cd kubo-car-mirror

# Build everything, including the kubo plugin
make build

# Build everything, using a local go-car-mirror dependency in a sibling clone
make build-local
```

## Building from kubo without sibling repo

If you want to create a branch in your kubo fork that can be used to build kubo-car-mirror, first clone your kubo fork and checkout the branch you want to use.
Then do the following.

```
cd kubo-car-mirror
make setup-kubo-build
cd ../kubo
make build-carmirror
git add .
git commit -m "Add build-carmirror make target"
```

Push your changes to your branch.  Now you will be able to clone the branch of your kubo fork and just run `make build-carmirror` to build kubo with the kubo-car-mirror plugin and then your normal kubo make targets, like `make build` and `make install`.

The build will also install the `carmirror` CLI to `kubo/carmirror/cmd/carmirror/carmirror`.  This binary will be gitignore'd, similar to how the `ipfs` CLI's binary is gitignore'd, so you don't accidentally add a platform specific binary to Git.

By default the latest version of the main branch in kubo-car-mirror will be built.  If you want to build a specific version, you can set the `KUBO_CAR_MIRROR_GIT_VERSION` environment variable before building.

## Updating to the latest version of go-car-mirror
First make sure you have a sibling repo of go-car-mirror with the latest version on main pulled.  Then run the following command.

```
make update-go-car-mirror
```

Check in the changes to go.mod and go.sum.

## Testing

```
# Run unit and sharness tests
make test

# Run unit tests
make test-unit

# Run sharness tests
make sharness

# Run sharness tests verbosely
make sharness-v

# Run sharness tests without downloading deps
make sharness-no-deps

# Run sharness tests verbosely without downloading deps
make sharness-no-deps-v
```

## Running

Once built, the kubo daemon has the CAR Mirror plugin baked in.

```
# Start kubo daemon
../kubo/cmd/ipfs/ipfs daemon
```

You can interact with local CAR Mirror APIs using the `carmirror` CLI.

```
# Push
./cmd/carmirror/carmirror push -c CID -a ADDR

# Pull
./cmd/carmirror/carmirror pull -c CID -a ADDR
```

During development, you might want to run in a testbed with [iptb](https://github.com/ipfs/iptb). This is essentially what happens in sharness tests, but gives you more flexibility in trying things out.

```
# Set up path, functions, etc
source test/lib/carmirror-lib.sh

# Create a temp dir to redirect downloads to with -o.
# Note, if you redirect to /dev/null instead, error messages will be silently swallowed.
DATE=$(date +"%Y-%m-%dT%H:%M:%SZ")
CM_TMP=$(mktemp -d "/tmp/carmirror_tests.$DATE.XXXXXX") || die "could not 'mktemp -d /tmp/carmirror_tests.$DATE.XXXXXX'"

# Stop and remove current testbed if started
iptb_stop
iptb_remove

# Create a 2 node testbed
iptb_new

# Start the daemons
iptb_start

# import test car file to node 0
ipfsi 0 dag import test/sharness/t0000-car-mirror-data/car-mirror.car

# confirm CID is on node 0
ipfsi 0 get QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W --offline -o $CM_TMP

# confirm CID is not on node 1
ipfsi 1 get QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W --offline -o $CM_TMP

# push CID from node 0 to node 1, in background so we can see session with ls output
carmirrori 0 push -c QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W -a $(cm_cli_remote_addr 1) -b

# OR pull CID from node 0 to node 1
carmirrori 1 pull -c QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W -a $(cm_cli_remote_addr 0) -b

# Confirm push in logs from node 0
iptb_logs 0

# Confirm push received in logs from node 1
iptb_logs 1

# confirm CID is on node 1 now
ipfsi 1 get QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W --offline -o $CM_TMP

# See running sessions
carmirrori 0 ls
carmirrori 1 ls

# Get all stats
carmirrori 0 stats

# Get stats for specific session
carmirrori 0 stats -s http://localhost:2505

# Close session
carmirrori 0 close -s http://localhost:2505

# Cancel session, for forcibly closing
carmirrori 0 cancel -s http://localhost:2505

# shutdown and cleanup
iptb_stop
iptb_remove
```

## Debugging

You can enable debugging in the logs using the `GOLOG_LOG_LEVEL` environment variable. This can help with running the daemon, the carmirror CLI, verbose sharness tests, or manual operations in a local testbed.

```
# Turn on debugging
export GOLOG_LOG_LEVEL="error,kubo-car-mirror=debug,go-car-mirror=debug"

# Start daemon, start testbed, run commands, ...
```

## Configuration

CAR Mirror configuration currently resides in Kubo's plugin configuration.

```
# Configure port for locally accessible commands
../kubo/cmd/ipfs/ipfs config --json Plugins.Plugins.car-mirror.Config.HTTPCommandsAddr '"127.0.0.1:2502"'

# Configure port for remotely accessible commands (i.e. the actual protocol commands)
../kubo/cmd/ipfs/ipfs config --json Plugins.Plugins.car-mirror.Config.HTTPRemoteAddr '":2503"'

# Configure max batch size
../kubo/cmd/ipfs/ipfs config --json Plugins.Plugins.car-mirror.Config.MaxBlocksPerRound 32

# Configure max batch size for cold call push
../kubo/cmd/ipfs/ipfs config --json Plugins.Plugins.car-mirror.Config.MaxBlocksPerColdCall 32

# Disable the plugin
../kubo/cmd/ipfs/ipfs config --json Plugins.Plugins.car-mirror.Disabled true
```

## Acknowledgements

Huge thanks 🙏 to [Jonathan Essex](https://github.com/softwareplumber), whose design and implementation of [go-car-mirror](https://github.com/fission-codes/go-car-mirror) does all the heavy lifting for this project.

A big thank you to the [Qri](https://github.com/qri-io) team, whose [Dsync](https://github.com/qri-io/dag) project helped save a ton of time in getting a Kubo plugin project like this up and running.
