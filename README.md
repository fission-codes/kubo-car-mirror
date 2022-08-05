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
In order to avoid package versioning issues common with Go plugins, it needs to be built in-tree with Kubo, per the instructions below.

```
# Use go modules, not needed if running Go 1.17 or later.
export GO111MODULE=on

# Clone kubo.
git clone https://github.com/ipfs/kubo
cd kubo

# Pull in the plugin (you can specify a version other than latest if you'd like, such as a GitHub branch).
go get github.com/fission-codes/go-car-mirror/plugin@latest

# Or if building against local copy...
cat <<EOF>>go.mod

require github.com/fission-codes/go-car-mirror latest

replace github.com/fission-codes/go-car-mirror => ../go-car-mirror
EOF

# Tidy up and download deps
go mod tidy

# Add the plugin to the preload list.
echo "carmirror github.com/fission-codes/go-car-mirror/plugin *" >> plugin/loader/preload_list

# Build kubo with the plugin.
make build

# This might fail saying we're missing `github.com/fission-codes/go-car-mirror/plugin` or `github.com/fission-codes/go-car-mirror`.  For some reason I sometimes have to rerun the `go get` above, or the `go mod tidy` and it will add it back.

# Install kubo.
make install
```

Once built and installed, the ipfs daemon has the CAR Mirror plugin baked in.  You can see it start by setting your log level to `debug`.

```
GOLOG_LOG_LEVEL="car-mirror-plugin=debug" ipfs daemon
```