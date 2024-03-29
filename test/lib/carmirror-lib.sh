# Must be sourced, not executed
BASE_DIR=$(dirname "$0")
KUBO_DIR=$(readlink -f $BASE_DIR/../../../kubo)
IPFS_CMD_DIR=$KUBO_DIR/cmd/ipfs
IPFS_CMD_PATH=$IPFS_CMD_DIR/ipfs

# If our ipfs cmd isn't in the path, iptb will use the default one
export PATH=$IPFS_CMD_DIR:$KUBO_DIR/test/bin:$PATH
export IPTB_ROOT="$HOME/.iptb"
DATE=$(date +"%Y-%m-%dT%H:%M:%SZ")
export CM_TMP=$(mktemp -d "/tmp/carmirror_tests.$DATE.XXXXXX") || die "could not 'mktemp -d /tmp/carmirror_tests.$DATE.XXXXXX'"


ipfsi() {
  dir="$1"
  shift
  IPFS_PATH="$IPTB_ROOT/testbeds/default/$dir" $KUBO_DIR/cmd/ipfs/ipfs "$@"
}

default_commands_port=2502

cm_commands_port() {
  node=$1
  echo $(( default_commands_port + ($node * 2) ))
}

cm_commands_addr() {
  node=$1
  echo "127.0.0.1:$(cm_commands_port $node)"
}

cm_remote_port() {
  node=$1
  commands_port=$(cm_commands_port $node)
  echo $(($commands_port + 1))
}

cm_remote_addr() {
  node=$1
  echo ":$(cm_remote_port $node)"
}

# TODO: Simplify if config should include http:// as well
cm_cli_commands_addr() {
  node=$1
  echo "http://$(cm_commands_addr $node)"
}

cm_cli_remote_addr() {
  node=$1
  echo "http://localhost$(cm_remote_addr $node)"
}

configure_cm() {
  num_nodes=$1
  for ((node=0; node<$num_nodes; node++)); do
    ipfsi $node config --json Plugins.Plugins.car-mirror.Config.HTTPCommandsAddr "\"$(cm_commands_addr $node)\""
    ipfsi $node config --json Plugins.Plugins.car-mirror.Config.HTTPRemoteAddr "\"$(cm_remote_addr $node)\""
    ipfsi $node config --json Plugins.Plugins.car-mirror.Config.LogLevel "\"debug\""
  done
}

# carmirror equivalent of ipfsi, allowing us to call the carmirror cli for a given node
carmirrori() {
  node="$1"
  shift

  time ./cmd/carmirror/carmirror --commands-address "$(cm_cli_commands_addr $node)" "$@"
}

iptb_new() {
  iptb testbed create -type localipfs -count 2 -force -init
  configure_cm 2
}

iptb_start() {
  export GOLOG_LOG_LEVEL="error,kubo-car-mirror=debug,go-car-mirror=debug"
  iptb start
  unset GOLOG_LOG_LEVEL
}

iptb_wait_stop() {
  while ! iptb run -- sh -c '! { test -e "$IPFS_PATH/repo.lock" && fuser -f "$IPFS_PATH/repo.lock" >/dev/null 2>&1; }'; do
      go-sleep 10ms
  done
}

iptb_restart() {
  iptb_stop
  iptb_wait_stop
  iptb_start
}


iptb_stop() {
  iptb stop
}

iptb_remove() {
  # TODO: if ipfs is still running, kill it
  rm -rf $IPTB_ROOT
}

iptb_fresh() {
  iptb_stop
  # iptb_wait_stop
  iptb_remove
  iptb_new
  iptb_start
  ps -ef | grep ipfs | grep -v grep
}

iptb_fresh_and_tail() {
  iptb_fresh
  iptb_tail
}

iptb_nuke() {
  iptb_stop
  iptb_remove
}

iptb_logs() {
  node="$1"
  shift

  iptb logs $node "$@"
}

iptb_tail() {
  if [ $# -eq 0 ]; then
    files=( $(ls ~/.iptb/testbeds/default/*/daemon.std*) )
  else
    files=( $(ls ~/.iptb/testbeds/default/$1/daemon.std*) )
  fi

  tail -f ${files[@]}
}

get_cid() {
  rm -rf $CM_TMP/ciddir
  random-files $CM_TMP/ciddir > /dev/null
  CID=$(ipfsi 0 add -Q --pin=false -r $CM_TMP/ciddir)
  echo $CID
}

echo "*** See README.md for instructions on setting up a testbed and running tests locally. ***"
