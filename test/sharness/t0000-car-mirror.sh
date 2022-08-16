#!/usr/bin/env bash

test_description="Test CAR Mirror"

. lib/test-lib.sh

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
    test_cmp $fname fetch_out
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

check_dir_fetch() {
  node=$1
  ref=$2

  test_expect_success "node can fetch all refs for dir" '
    ipfsi $node refs -r $ref --offline > /dev/null
  '
}

check_no_dir_fetch() {
  node=$1
  ref=$2

  test_expect_success "node cannot fetch all refs for dir" '
    ipfsi $node refs -r $ref --offline > /dev/null 2> fetch_error
    test_should_contain "could not find $ref" fetch_error
  '

}

run_single_file_test() {
  test_expect_success "add a file on node1" '
    random 1000000 > filea &&
    FILEA_HASH=$(ipfsi 0 add -q filea)
  '

  check_file_fetch 0 $FILEA_HASH filea
  check_no_file_fetch 1 $FILEA_HASH
}

run_random_dir_test() {
  test_expect_success "create a bunch of random files" '
    random-files -depth=3 -dirs=4 -files=5 -seed=5 foobar > /dev/null
  '

  test_expect_success "add those on node 0" '
    DIR_HASH=$(ipfsi 0 add -r -Q foobar)
  '

  check_dir_fetch 0 $DIR_HASH
  check_no_dir_fetch 1 $DIR_HASH
}

run_advanced_test() {
  startup_cluster_disconnected 2 "$@"

  test_expect_success "clean repo before test" '
    ipfsi 0 repo gc > /dev/null &&
    ipfsi 1 repo gc > /dev/null
  '

  run_single_file_test
  run_random_dir_test

  test_expect_success "shut down nodes" '
    iptb stop && iptb_wait_stop
  '
}

test_expect_success "set up testbed" '
  iptb testbed create -type localipfs -count 2 -force -init
'

run_advanced_test

test_done
