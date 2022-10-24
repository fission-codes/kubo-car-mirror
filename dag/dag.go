package dag

import (
	"context"
	"fmt"

	gocid "github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
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
		if cidsSet.Visit(cid) {
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
	}

	return
}
