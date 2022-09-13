package dag

import (
	"fmt"

	cid "github.com/ipfs/go-cid"
)

// Converts a list of CID strings into a list of `cid.Cid`s.
// Returns `nil, error` if any CID strings fail to parse.
func ParseCids(cids []string) ([]cid.Cid, error) {
	var rootCids []cid.Cid
	rootCidsSet := cid.NewSet()

	// Create a CAR file containing all CIDs
	for _, rootCid := range cids {
		parsedRootCid, err := ParseCid(rootCid)
		if err != nil {
			return nil, err
		}
		if rootCidsSet.Visit(*parsedRootCid) {
			rootCids = append(rootCids, *parsedRootCid)
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
