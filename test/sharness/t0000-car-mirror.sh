#!/usr/bin/env bash

test_description="Test CAR Mirror"

. lib/test-lib.sh

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

configure_cm_ports() {
  num_nodes=$1
  for ((node=0; node<$num_nodes; node++)); do
    test_expect_success "configure car mirror port for node $node" "
      ipfsi $node config --json Plugins.Plugins.car-mirror.Config.HTTPCommandsAddr '\""$(cm_commands_addr $node)"\"' &&
      ipfsi $node config --json Plugins.Plugins.car-mirror.Config.HTTPRemoteAddr '\""$(cm_remote_addr $node)"\"' &&
      ipfsi $node config Plugins.Plugins.car-mirror.Config.HTTPCommandsAddr > node_config &&
      test_should_contain \""$(cm_commands_addr $node)"\" node_config &&
      ipfsi $node config Plugins.Plugins.car-mirror.Config.HTTPRemoteAddr > node_config &&
      test_should_contain \""$(cm_remote_addr $node)"\" node_config
    "
  done
}

# carmirror equivalent of ipfsi, allowing us to call the carmirror cli for a given node
carmirrori() {
  node="$1"
  shift

  carmirror --commands-address "$(cm_commands_addr $node)" "$@"
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

  # configure_cm_ports 2

  test_expect_success "import test CAR file on node 0" '
    ipfsi 0 dag import ../t0000-car-mirror-data/car-mirror.car
  '

  check_has_cid_root 0 QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W
  check_not_has_cid_root 1 QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W

  # pull CIDs from node 0 to node 1

  # Just confirming car mirror is serving over the right port
  test_expect_success "curl" '
    curl -v --data-binary @../t0000-car-mirror-data/pull.cbor "http://localhost:2502/dag/pull/new" --output blah.car 2> curl_out &&
    test_should_not_contain "Internal Server Error" curl_out
  '

  test_expect_success "car mirror push works" '
    carmirrori 0 push --commands-address $(cm_commands_addr 1) QmWXCR7ZwcQpvzJA5fjkQMJTe2rwJgYUtoSxBXFZ3uBY1W $(cm_remote_addr )
  '

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
