package oldcarmirror

import (
	"fmt"

	cid "github.com/ipfs/go-cid"
)

// Converts a list of CID strings into a list of `cid.Cid`s.
// Returns `nil, error` if any CID strings fail to parse.
func ParseCids(cids []string) ([]cid.Cid, error) {
	rootCids := make([]cid.Cid, len(cids))
	rootCidsSet := cid.NewSet()

	// Create a CAR file containing all CIDs
	i := 0
	for _, rootCid := range cids {
		parsedRootCid, err := ParseCid(rootCid)
		if err != nil {
			return nil, err
		}
		if rootCidsSet.Visit(*parsedRootCid) {
			rootCids[i] = *parsedRootCid
			i += 1
		}
	}

	return rootCids, nil
}

func ParseCid(cidStr string) (*cid.Cid, error) {
	parsedCid, err := cid.Decode(cidStr)
	if err != nil {
		return nil, fmt.Errorf("CID %q cannot be parsed: %v", cidStr, err)
	}

	return &parsedCid, nil
}
