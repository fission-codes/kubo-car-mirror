package dag

import (
	"testing"

	cid "github.com/ipfs/go-cid"
	"gotest.tools/assert"
)

func TestParseCids(t *testing.T) {
	// One CID
	cidStrings := []string{"bafybeigs3wowz6pug7ckfgtwrsrltjjx5disx5pztnucgt4ygryv5w6qy4"}
	cids, err := ParseCids(cidStrings)
	assert.Equal(t, err, nil, "No errors should be returned parsing CIDs")

	c, err := cid.Decode("bafybeigs3wowz6pug7ckfgtwrsrltjjx5disx5pztnucgt4ygryv5w6qy4")
	assert.Equal(t, err, nil, "Should not get error decoding CID")
	assert.Equal(t, c, cids[0], "CIDs should match")
	assert.Equal(t, len(cids), 1, "One CID should be in list")

	// Many CIDs

	// One invalid CID

	// No CIDs
}
