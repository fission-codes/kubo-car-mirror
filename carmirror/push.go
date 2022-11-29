package carmirror

import (
	"context"

	"github.com/fission-codes/go-bloom"
	gocid "github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	mdag "github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-merkledag/traverse"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/pkg/errors"
	"github.com/zeebo/xxh3"
)

type Pusher struct {
	ctx                       context.Context
	lng                       ipld.NodeGetter
	capi                      coreiface.CoreAPI
	remainingRoots            []gocid.Cid
	cleanupCids               []gocid.Cid
	subgraphRoots             []gocid.Cid
	diff                      string
	stream                    bool
	remote                    CarMirrorable
	maxBlocksPerRound         int64
	maxBlocksPerColdCall      int64
	currentRound              int64
	providerGraphConfirmation *bloom.Filter[[]byte, bloom.HashFunction[[]byte]]
	// cidsSeen hashmap
	// remoteBloom
}

func NewPusher(ctx context.Context, cfg *Config, lng ipld.NodeGetter, capi coreiface.CoreAPI, cids []gocid.Cid, diff string, stream bool, remote CarMirrorable) *Pusher {
	pusher := &Pusher{
		ctx:                       ctx,
		lng:                       lng,
		capi:                      capi,
		remainingRoots:            cids,
		diff:                      diff,
		stream:                    stream,
		remote:                    remote,
		currentRound:              0,
		maxBlocksPerRound:         cfg.MaxBlocksPerRound,
		maxBlocksPerColdCall:      cfg.MaxBlocksPerColdCall,
		providerGraphConfirmation: nil,
	}
	return pusher
}

// Next returns true if there are any CIDs left to push.
func (p *Pusher) Next() bool {
	return len(p.remainingRoots)+len(p.cleanupCids) > 0
}

// Split the list of root CIDs into CIDs we will push next, remaining CIDs for adding to bloom filters, and remaining root CIDs for future pushes, and lastly the CIDs that aren't found locally.
// The list of CIDs to push next will always include root CIDs ignoring bloom filters, and then CIDs under those roots CIDs based on bloom filter tests.
// This ensures that we always have root CIDs that encompass all roots we intend to push, so that the returned subgraph roots will always cover all remaining roots.
func (p *Pusher) NextCids(rootCids []gocid.Cid) (pushCids []gocid.Cid, remainingCids []gocid.Cid, remainingRoots []gocid.Cid, notFoundPushCids []gocid.Cid) {
	rootCidsSet := gocid.NewSet()
	for _, cid := range rootCids {
		rootCidsSet.Add(cid)
	}
	pushCidsSet := gocid.NewSet()
	remainingCidsSet := gocid.NewSet()
	remainingRootsSet := gocid.NewSet()

	var maxBlocks uint64
	if p.currentRound == 0 {
		maxBlocks = uint64(p.maxBlocksPerColdCall)
	} else {
		maxBlocks = uint64(p.maxBlocksPerRound)
	}

	for _, cid := range rootCids {
		var rp path.Resolved
		var nd ipld.Node
		var lastDepth int
		lastDepth = 0

		rp, err := p.capi.ResolvePath(p.ctx, path.New(cid.String()))
		if err != nil {
			notFoundPushCids = append(notFoundPushCids, cid)
			log.Debugf("unable to resolve path for root cid %s.  Adding to notFoundPushCids.  err=%v", cid.String(), err)
			continue
		}

		nodeGetter := mdag.NewSession(p.ctx, p.lng)
		nd, err = nodeGetter.Get(p.ctx, rp.Cid())
		if err != nil {
			notFoundPushCids = append(notFoundPushCids, cid)
			log.Debugf("unable to get nodes for root cid %s.  Adding to notFoundPushCids.  err=%v", cid.String(), err)
			continue
		}

		err = traverse.Traverse(nd, traverse.Options{
			DAG:   nodeGetter,
			Order: traverse.BFS, // Breadth first
			Func: func(current traverse.State) error {
				// Always push root CIDs, or all CIDs if no bloom was provided.  Otherwise only push if CID isn't in bloom.
				if rootCidsSet.Has(cid) || len(p.providerGraphConfirmation.Bytes()) == 0 || !p.providerGraphConfirmation.Test(current.Node.Cid().Bytes()) {
					if len(pushCids) < int(maxBlocks) {
						if pushCidsSet.Visit(current.Node.Cid()) {
							pushCids = append(pushCids, current.Node.Cid())
						}
						lastDepth = current.Depth
					} else {
						if current.Depth == lastDepth {
							if remainingRootsSet.Visit(current.Node.Cid()) {
								remainingRoots = append(remainingRoots, current.Node.Cid())
							}
						}
						if remainingCidsSet.Visit(current.Node.Cid()) {
							remainingCids = append(remainingCids, current.Node.Cid())
						}
					}
				}

				return nil
			},
			ErrFunc:        nil,
			SkipDuplicates: true,
		})
		if err != nil {
			notFoundPushCids = append(notFoundPushCids, cid)
			log.Debugf("error traversing DAG.  Adding to notFoundPushCids.  err=%v", err)
			continue
		}
	}

	// TODO: If we still don't have our limit of blocks, should we include some cleanup CIDs as well, if present?

	return
}

// ShouldCleanup returns true if there are straggler CIDs to cleanup and no more root CIDs to push.
func (p *Pusher) ShouldCleanup() bool {
	return len(p.remainingRoots) == 0 && len(p.cleanupCids) > 0
}

// Cleanup pushes the collected cleanupCids without the use of bloom filters.
func (p *Pusher) Cleanup() (err error) {
	log.Debugf("Pusher.Cleanup")
	pushCids, err := p.DoPush(p.cleanupCids, false)
	if err != nil {
		return err
	}

	// Remove pushed cids from cleanupCids
	var newCleanupCids []gocid.Cid
	cleanupCidsSet := gocid.NewSet()
	for _, cid := range p.cleanupCids {
		cleanupCidsSet.Add(cid)
	}
	for _, cid := range pushCids {
		if !cleanupCidsSet.Has(cid) {
			newCleanupCids = append(newCleanupCids, cid)
		}
	}
	p.cleanupCids = newCleanupCids

	return nil
}

func (p *Pusher) Push() (err error) {
	_, err = p.DoPush(p.remainingRoots, true)
	if err != nil {
		return err
	}
	return nil
}

// Push executes the push, blocking until complete
func (p *Pusher) DoPush(remainingRoots []gocid.Cid, includeBloom bool) (pushCids []gocid.Cid, err error) {
	// On a partial cold call, the Provider Graph Estimate MUST contain the entire graph minus CIDs in the initial payload. The Provider MUST respond with a Bloom filter of all CIDs that match the Provider Graph Estimate, which is called the Provider Graph Confirmation. On subsequent rounds, the Provider Graph Estimate continues to be refined until is is empty or the entire graph has been synchronized.
	log.Debugf("Pusher.Push")

	pushCids, remainingCids, remainingRoots, _ := p.NextCids(remainingRoots)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get next CIDs")
	}
	log.Debugf("pushCids=%v", pushCids)
	log.Debugf("remainingCids=%v", remainingCids)
	log.Debugf("remainingRoots=%v", remainingRoots)

	var providerGraphEstimate *bloom.Filter[[]byte, bloom.HashFunction[[]byte]]
	if includeBloom {
		if p.currentRound == 0 {
			// Cold start.  Create bloom of remainingCids for payload
			// TODO: If we have all cids locally underneath the root and if we don't have a diff param, no bloom is needed.
			n := uint64(len(remainingCids))
			var function bloom.HashFunction[[]byte] = xxh3.HashSeed

			providerGraphEstimate, err = bloom.NewFilterWithEstimates(n, 0.0001, function)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to create new filter")
			}
			for _, cid := range remainingCids {
				providerGraphEstimate.Add(cid.Bytes())
			}
			log.Debugf("Cold start.  Building providerGraphEstimate from all CIDs.")
		} else if p.providerGraphConfirmation != nil {
			// TODO: If we don't update it elsewhere, make sure the providerGraphEstimate is a bloom that combines all returned confirmations into a new bloom with all of their adds.
			log.Debugf("round > 0 and providerGraphConfirmation not nil, setting providerGraphEstimate to providerGraphConfirmation")
			providerGraphEstimate = p.providerGraphConfirmation
		} else {
			log.Debugf("p.providerGraphConfirmation is nil, so not setting providerGraphEstimate")
		}
	}

	// Send payload to provider, with provider returning providerGraphConfirmation bloom and SR
	// TODO: Probably need status of response too
	providerGraphConfirmation, subgraphRoots, err := p.remote.Push(p.ctx, pushCids, providerGraphEstimate, p.diff)
	if err != nil {
		return nil, err
	}
	// TODO: Combine previous bloom with new bloom, resizing if necessary.
	p.providerGraphConfirmation = providerGraphConfirmation

	// Set p.remainingCids to remainingRoots + SR
	// Maybe set SR to something in state, because if empty and still have remainingCids, we need to send nil bloom.
	remainingRootsSet := gocid.NewSet()
	for _, cid := range remainingRoots {
		remainingRootsSet.Add(cid)
	}
	for _, cid := range subgraphRoots {
		if remainingRootsSet.Visit(cid) {
			remainingRoots = append(remainingRoots, cid)
		}
	}
	p.remainingRoots = remainingRoots

	p.currentRound += 1

	return pushCids, nil
}
