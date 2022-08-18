#!/usr/bin/env bash

test_description="Test CAR Mirror"

. lib/test-lib.sh

default_ipfs_remote_port=4001
default_cm_port=2504

cm_port() {
  node=$1
  echo $((default_cm_port + $node))
}

cm_command_addr() {
  node=$1
  echo "127.0.0.1:$(cm_port $node)"
}

ipfs_remote_port() {
  node=$1
  echo $((default_ipfs_remote_port + $node))
}

ipfs_remote_addr() {
  node=$1
  echo "127.0.0.1:$(ipfs_remote_port $node)"
}

configure_cm_ports() {
  num_nodes=$1
  for ((node=0; node<$num_nodes; node++)); do
    test_expect_success "configure car mirror port for node $node" "
      ipfsi $node config --json Plugins.Plugins.car-mirror.Config.HTTPCommandsAddr '\""$(cm_command_addr $node)"\"' &&
      ipfsi $node config --json Plugins.Plugins.car-mirror.Config.HTTPRemoteAddr '\""$(ipfs_remote_addr $node)"\"' &&
      ipfsi $node config Plugins.Plugins.car-mirror.Config.HTTPCommandsAddr > node_config &&
      test_should_contain \""$(cm_command_addr $node)"\" node_config &&
      ipfsi $node config Plugins.Plugins.car-mirror.Config.HTTPRemoteAddr > node_config &&
      test_should_contain \""$(ipfs_remote_addr $node)"\" node_config
    "
  done
}

# car mirror equivalent of ipfsi, allowing us to call the car mirror cli for a given node

cmi() {
  node="$1"
  dir=$node
  shift
  IPFS_PATH="$IPTB_ROOT/testbeds/default/$dir" ipfs "$@"

  cm --command-address $(cm_command_addr $node) $@
}

# Don't connect nodes together.
# Shouldn't matter since we run commands with --offline, but just to be safe...
startup_cluster_disconnected() {
  num_nodes="$1"
  shift
  other_args="$@"
  bound=$(expr "$num_nodes" - 1)

  if test -n "$other_args"; then
    test_expect_success "start up nodes with additional args" "
      iptb start -wait [0-$bound] -- ${other_args[@]}
    "
  else
    test_expect_success "start up nodes" '
      iptb start -wait [0-$bound]
    '
  fi
}


check_file_fetch() {
  node=$1
  fhash=$2
  fname=$3

  test_expect_success "can fetch file" '
    ipfsi $node cat $fhash --offline > fetch_out
  '

  test_expect_success "file looks good" '
    ipfsi $node cat $fhash --offline > /dev/null 2> fetch_error
    test_should_not_contain "could not find $fhash" fetch_error
  '
}

check_no_file_fetch() {
  node=$1
  fhash=$2

  test_expect_success "node cannot fetch file" '
    ipfsi $node cat $fhash --offline 2> fetch_error
    test_should_contain "could not find $fhash" fetch_error
  '

}

check_has_cid_root() {
  node=$1
  cid=$2

  test_expect_success "node $node can get cid root $cid" '
    ipfsi $node get $cid --offline >/dev/null 2> get_error
    test_should_not_contain "block was not found locally" get_error
  '
}

check_not_has_cid_root() {
  node=$1
  cid=$2

  test_expect_success "node $node can get cid root $cid" '
    ipfsi $node get $cid --offline >/dev/null 2> get_error
    test_should_contain "block was not found locally" get_error
  '
}

run_pull_test() {
  startup_cluster_disconnected 2 "$@"

  test_expect_success "clean repo before test" '
    ipfsi 0 repo gc > /dev/null &&
    ipfsi 1 repo gc > /dev/null
  '

  configure_cm_port 2

  test_expect_success "import test CAR file on node 0" '
    ipfsi 0 dag import ../t0000-car-mirror-data/car-mirror.car
  '

  check_has_cid_root 0 QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W
  check_not_has_cid_root 1 QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W

  # pull CIDs from node 0 to node 1

  # Just confirming car mirror is serving over the right port
  test_expect_success "curl" '
    curl -v --data-binary @../t0000-car-mirror-data/pull.cbor "http://localhost:2504/dag/pull" --output blah.car 2> curl_out &&
    test_should_not_contain "Internal Server Error" curl_out
  '

  # test_expect_success "car mirror pull works" '
  #   cmi 1 pull --from $(cm_command_addr 0)
  # '

  # check_has_cid_root 1 QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W

  test_expect_success "shut down nodes" '
    iptb stop && iptb_wait_stop
  '
}

test_expect_success "set up testbed" '
  iptb testbed create -type localipfs -count 2 -force -init
'

test_expect_success "configure the plugin" '
  configure_cm_ports 2
'

run_pull_test

test_done
