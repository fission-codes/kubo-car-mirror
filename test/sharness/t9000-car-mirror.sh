#!/usr/bin/env bash

test_description="Test CAR Mirror"

. lib/test-lib.sh

test_init_ipfs
test_launch_ipfs_daemon

## ============================================================================
## Test _redirects file support
## ============================================================================

# Import test case
# Run `ipfs cat /ipfs/$REDIRECTS_DIR_CID/_redirects` to see sample _redirects file
test_expect_success "Add the CAR Mirror file test directory" '
  ipfs dag import ../t9000-car-mirror-data/car-mirror.car
'

test_kill_ipfs_daemon

test_done
