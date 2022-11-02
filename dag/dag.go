package dag

import (
	"context"
	"fmt"

	gocid "github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	golog "github.com/ipfs/go-log"
	mdag "github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-merkledag/traverse"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

var log = golog.Logger("dag")

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

func SubgraphRoots(ctx context.Context, lng ipld.NodeGetter, cids []gocid.Cid) (subgraphRoots []gocid.Cid) {
	subgraphRootsSet := gocid.NewSet()
	cidsSet := gocid.NewSet()
	for _, cid := range cids {
		cidsSet.Add(cid)
	}

	// iterate through cids
	// if they have links and any links are not in cids, add those links to subgraphRoots, ignoring dupes
	// var node format.Node
	for _, cid := range cids {
		node, err := lng.Get(ctx, cid)
		if err != nil {
			// If the CID isn't found locally, treat it as a subgraph root, though perhaps we should error
			subgraphRootsSet.Add(cid)
			continue
		}
		for _, link := range node.Links() {
			if !cidsSet.Has(link.Cid) {
				subgraphRootsSet.Add(link.Cid)
			}
		}
	}
	// put subgraph roots in a slice
	subgraphRoots = make([]gocid.Cid, subgraphRootsSet.Len())
	i := 0
	subgraphRootsSet.ForEach(func(cid gocid.Cid) error {
		subgraphRoots[i] = cid
		i++
		return nil
	})

	return
}

// Maybe change this to include roots too.
// This traverses.  Maybe we want an option that doesn't traverse too.
func NextCids(ctx context.Context, cids []gocid.Cid, lng ipld.NodeGetter, capi coreiface.CoreAPI, maxBlocks uint64) (nextCids []gocid.Cid, remainingCids []gocid.Cid, notFoundCids []gocid.Cid) {
	// for each cid
	//  if didn't visit cid and nextCids is less than maxBlocks
	//    add to nextCids
	//    traverse
	//      if didn't visit cid and nextCids is less than maxBlocks
	//        add to nextCids
	//    ... anytime nextCids is not less than maxBlocks, add remaining cids to remainingCids

	cidsSet := gocid.NewSet()

	for _, cid := range cids {
		var rp path.Resolved
		var nd ipld.Node

		rp, err := capi.ResolvePath(ctx, path.New(cid.String()))
		if err != nil {
			notFoundCids = append(notFoundCids, cid)
			log.Debugf("unable to resolve path for root cid %s.  Adding to notFoundCids.  err=%v", cid.String(), err)
			continue
		}

		nodeGetter := mdag.NewSession(ctx, lng)
		nd, err = nodeGetter.Get(ctx, rp.Cid())
		if err != nil {
			notFoundCids = append(notFoundCids, cid)
			log.Debugf("unable to get nodes for root cid %s.  Adding to notFoundCids.  err=%v", cid.String(), err)
			continue
		}

		err = traverse.Traverse(nd, traverse.Options{
			DAG:   nodeGetter,
			Order: traverse.BFS, // Breadth first
			Func: func(current traverse.State) error {
				if cidsSet.Visit(current.Node.Cid()) {
					if len(nextCids) < int(maxBlocks) {
						nextCids = append(nextCids, current.Node.Cid())
					} else {
						remainingCids = append(remainingCids, current.Node.Cid())
					}
				}
				return nil
			},

			ErrFunc:        nil,
			SkipDuplicates: true,
		})
		if err != nil {
			notFoundCids = append(notFoundCids, cid)
			log.Debugf("error traversing DAG.  Adding to notFoundCids.  err=%v", err)
			continue
		}
	}

	return
}
