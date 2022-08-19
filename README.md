![](https://github.com/fission-codes/go-car-mirror/raw/master/assets/logo.png?sanitize=true)

# go-car-mirror

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
```

## Running

Once built, the kubo daemon has the CAR Mirror plugin baked in.

```
# Start kubo daemon
../kubo/cmd/ipfs/ipfs daemon

# Or start with debug
GOLOG_LOG_LEVEL="car-mirror=debug,car-mirror-plugin=debug" ../kubo/cmd/ipfs daemon
```

As a convenience, you can interact with CAR Mirror APIs using the `carmirror` CLI.

```
# Push
./cmd/carmirror push ...

# Pull
./cmd/carmirror pull ...
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

A big thank you to the [qri-io](https://github.com/qri-io) team, whose [dsync](https://github.com/qri-io/dag) project helped save a ton of time in getting this codebase organized. 🙏