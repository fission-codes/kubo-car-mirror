![](https://github.com/fission-codes/go-car-mirror/raw/master/assets/logo.png?sanitize=true)

# go-car-mirror

[![CI](https://github.com/fission-codes/go-car-mirror/actions/workflows/main.yml/badge.svg)](https://github.com/fission-codes/go-car-mirror/actions/workflows/main.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/fission-codes/blob/master/LICENSE)
[![Built by FISSION](https://img.shields.io/badge/⌘-Built_by_FISSION-purple.svg)](https://fission.codes)
[![Discord](https://img.shields.io/discord/478735028319158273.svg)](https://discord.gg/zAQBDEq)
[![Discourse](https://img.shields.io/discourse/https/talk.fission.codes/topics)](https://talk.fission.codes)


🚧 WIP 🚧

Go implementation of [CAR Mirror](https://github.com/fission-codes/spec/tree/main/car-pool).

## Building

go-car-mirror is implemented as a [Kubo Daemon Plugin](https://github.com/ipfs/kubo/blob/master/docs/plugins.md#daemon).
In order to avoid package versioning issues common with Go plugins, it needs to be built in-tree with Kubo.

We provide some make targets to simplify this process.

```
# Clone repos to same parent directory
git clone https://github.com/fission-codes/go-car-mirror
git clone https://github.com/ipfs/kubo

# Change to the go-car-mirror directory to run make targets
cd go-car-mirror

# Build
make build

# Build the kubo plugin
make build-plugin
```

## Testing

```
# Run Go tests
make test

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
./cmd/carmirror push ...

# Pull
./cmd/carmirror pull ...
```

During development, you might want to run in a testbed with [iptb](https://github.com/ipfs/iptb).  This is essentially what happens in sharness tests, but gives you more flexibility in trying things out.

```
# set up path, functions, etc
source test/lib/carmirror-lib.sh

# Create a 2 node testbed
iptb_new

# Start the daemons
iptb_start

# import test car file to node 0
ipfsi 0 dag import test/sharness/t0000-car-mirror-data/car-mirror.car

# confirm CID is on node 0
ipfsi 0 get QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W --offline -o /dev/null

# confirm CID is not on node 1
ipfsi 1 get QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W --offline -o /dev/null

# push CID from node 0 to node 1
carmirrori 0 push QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W $(cm_cli_remote_addr 1)

# OR pull CID from node 0 to node 1
carmirrori 1 pull QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W $(cm_cli_remote_addr 0)

# Confirm push in logs from node 0
iptb_logs 0

# Confirm push received in logs from node 1
iptb_logs 1

# confirm CID is on node 1 now
ipfsi 1 get QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W --offline -o /dev/null

# shutdown and cleanup
iptb_stop
iptb_remove
```

## Debugging

You can enable debugging in the logs using the `GOLOG_LOG_LEVEL` environment variable.  This can help with running the daemon, the carmirror CLI, verbose sharness tests, or manual operations in a local testbed.

```
# Turn on debugging
export GOLOG_LOG_LEVEL="car-mirror=debug,car-mirror-plugin=debug"

# Start daemon, start testbed, run commands, ...
```

## Configuration

CAR Mirror configuration currently resides in Kubo's plugin configuration.

```
# Configure port for locally accessible commands
../kubo/cmd/ipfs/ipfs config --json Plugins.Plugins.car-mirror.Config.HTTPCommandsAddr '"127.0.0.1:2502"'

# Configure port for remotely accessible commands (i.e. the actual protocol commands)
../kubo/cmd/ipfs/ipfs config --json Plugins.Plugins.car-mirror.Config.HTTPRemoteAddr '":2503"'

# Disable the plugin
../kubo/cmd/ipfs/ipfs config --json Plugins.Plugins.car-mirror.Disabled true
```

## Acknowledgements

A big thank you 🙏 to the [Qri](https://github.com/qri-io) team, whose [Dsync](https://github.com/qri-io/dag) project helped save a ton of time in getting this codebase started.