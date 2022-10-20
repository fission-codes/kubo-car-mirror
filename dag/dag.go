package dag

import (
	"context"
	"fmt"

	gocid "github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	ipld "github.com/ipfs/go-ipld-format"
)

// Converts a list of CID strings into a list of `cid.Cid`s.
// Returns `nil, error` if any CID strings fail to parse.
func ParseCids(cids []string) ([]gocid.Cid, error) {
	rootCids := make([]gocid.Cid, len(cids))
	rootCidsSet := gocid.NewSet()

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

func ParseCid(cidStr string) (*gocid.Cid, error) {
	parsedCid, err := gocid.Decode(cidStr)
	if err != nil {
		return nil, fmt.Errorf("CID %q cannot be parsed: %v", cidStr, err)
	}

	return &parsedCid, nil
}

func SubgraphRoots(ctx context.Context, lng ipld.NodeGetter, cids []gocid.Cid) (subgraphRoots []gocid.Cid, err error) {
	subgraphRootsMap := make(map[gocid.Cid]bool)
	// convert cids to hashmap efficient membership checking
	cidsMap := make(map[gocid.Cid]bool)
	for _, c := range cids {
		cidsMap[c] = true
	}

	// iterate through cids
	// if they have links and any links are not in cids, add to subgraphRoots, ignoring dupes
	var node format.Node
	for _, c := range cids {
		node, err = lng.Get(ctx, c)
		if err != nil {
			return
		}
		for _, link := range node.Links() {
			if _, ok := cidsMap[link.Cid]; !ok {
				subgraphRootsMap[link.Cid] = true
			}
		}
	}
	// convert subgraph roots map back to slice
	subgraphRoots = make([]gocid.Cid, len(subgraphRootsMap))
	i := 0
	for k := range subgraphRootsMap {
		subgraphRoots[i] = k
		i++
	}

	return
}
