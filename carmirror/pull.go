package carmirror

import (
	"context"

	"github.com/fission-codes/go-bloom"
	"github.com/fission-codes/kubo-car-mirror/dag"
	gocid "github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	mdag "github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-merkledag/traverse"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/pkg/errors"
)

type Puller struct {
	ctx            context.Context
	lng            ipld.NodeGetter
	capi           coreiface.CoreAPI
	pullRoots      []gocid.Cid
	remainingRoots []gocid.Cid
	// CID roots of extra shared structure, to create initial bloom from.
	// This is analogous to the diff param for Push, but is not specified in the spec, left up to implementors.
	sharedRoots       []gocid.Cid
	bloom             *bloom.Filter
	remote            CarMirrorable
	currentRound      int64
	cleanupRound      bool
	maxBlocksPerRound int64
}

func NewPuller(ctx context.Context, cfg *Config, lng ipld.NodeGetter, capi coreiface.CoreAPI, pullRoots []gocid.Cid, sharedRoots []gocid.Cid, remote CarMirrorable) *Puller {
	puller := &Puller{
		ctx:               ctx,
		lng:               lng,
		capi:              capi,
		pullRoots:         pullRoots,
		remainingRoots:    pullRoots,
		sharedRoots:       sharedRoots,
		remote:            remote,
		currentRound:      0,
		maxBlocksPerRound: cfg.MaxBlocksPerRound,
		cleanupRound:      false,
	}
	return puller
}

// Next returns true if there are more CIDs to transmit.
func (p *Puller) Next() bool {
	return len(p.remainingRoots) > 0
}

// ShouldCleanup returns true if the time has come for a cleanup round.
// TODO: Should we cleanup as soon as a request returns only the requested roots?  If so, this method needs to change.
func (p *Puller) ShouldCleanup() bool {
	return p.cleanupRound
}

// Pull performs a pull on the remaining roots using the requestor bloom filter.
func (p *Puller) Pull() (err error) {
	log.Debugf("Puller.Pull")
	_, err = p.DoPull(p.remainingRoots, true)
	if err != nil {
		return err
	}
	return nil
}

// Cleanup performs a pull on the cleanup CIDs using no bloom filter.
func (p *Puller) Cleanup() (err error) {
	log.Debugf("Puller.Cleanup")
	_, err = p.DoPull(p.remainingRoots, false)
	if err != nil {
		return err
	}
	p.cleanupRound = false

	return nil
}

func (p *Puller) DoPull(roots []gocid.Cid, includeBloom bool) (pullCids []gocid.Cid, err error) {
	log.Debugf("Puller.DoPull")

	// TODO: Use remainingRoots for efficiency, vs recomputing remainingRoots every time.
	pullRoots, remainingRoots := p.NextCids(roots)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get next CIDs")
	}
	log.Debugf("pullRoots=%v", pullRoots)
	log.Debugf("remainingRoots=%v", remainingRoots)

	// Need to decide when to recreate due to saturation
	if includeBloom {
		// TODO: If blockstore is small enough, just add all blocks to the bloom
		if p.currentRound == 0 {
			bloomCids := p.UniqueLocalCids(append(p.pullRoots, p.sharedRoots...))
			n := uint64(len(bloomCids))
			p.bloom = bloom.NewFilterWithEstimates(n, bloom.EstimateFPP(n))
			// p.bloom = bloom.NewFilterWithEstimates(n, 0.001)
			for _, cid := range bloomCids {
				p.bloom.Add(cid.Bytes())
			}
			log.Debugf("Cold start.  Building bloom from all known local CIDs.")
		}
	}

	// TODO: for pull, subgraph roots are recalculated on the requestor side, not returned from pull

	// This method submits the request for roots, receives a car file, adds to local blockstore, returns list of those new CIDs
	pulledCids, err := p.remote.Pull(p.ctx, pullRoots, p.bloom)
	if err != nil {
		return nil, err
	}

	// TODO: add cids to caches as well, somewhere, maybe

	// TODO: Add new pulled CIDs to bloom so ready for next round
	for _, cid := range pulledCids {
		// TODO: resize if saturated
		p.bloom.Add(cid.Bytes())
	}

	// If only the requested roots were returned, we need a cleanup round
	p.cleanupRound = sameCids(pulledCids, pullRoots)

	// Update remaining roots
	localCidsAfterPull := p.UniqueLocalCids(pullRoots)
	pulledRemainingRoots, err := p.RemainingRoots(localCidsAfterPull)
	log.Debugf("localCidsAfterPull = %v", localCidsAfterPull)
	log.Debugf("pulledRemainingRoots = %v", pulledRemainingRoots)
	if err != nil {
		return
	}
	p.remainingRoots = append(pulledRemainingRoots, remainingRoots...)

	// This won't be needed once I clean up where blooms are set
	p.currentRound += 1

	return pullCids, nil
}

func sameCids(a []gocid.Cid, b []gocid.Cid) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equals(b[i]) {
			return false
		}
	}
	return true
}

// CleanupCids returns the list of CIDs from pullRoots that are not in pulledCids.
// All pullRoots are supposed to be transmitted regardless of bloom filter, so their
// absence indicates they may need to be cleaned up.
func CleanupCids(pullRoots []gocid.Cid, pulledCids []gocid.Cid) (cleanupCids []gocid.Cid) {
	pulledCidsSet := gocid.NewSet()
	for _, cid := range pulledCids {
		pulledCidsSet.Add(cid)
	}
	cleanupCidsSet := gocid.NewSet()

	for _, cid := range pullRoots {
		if !pulledCidsSet.Has(cid) {
			if cleanupCidsSet.Visit(cid) {
				cleanupCids = append(cleanupCids, cid)
			}
		}
	}

	return
}

// RemainingRoots returns the list of subgraph roots underneath pullRoots that are not cleanup CIDs.
func (p *Puller) RemainingRoots(pullRoots []gocid.Cid) (remainingRoots []gocid.Cid, err error) {
	subgraphRoots := dag.SubgraphRoots(p.ctx, p.lng, pullRoots)
	if err != nil {
		log.Debugf("error getting subgraph roots. err = %v, subgraphRoots = %v", err, subgraphRoots)
		return nil, err
	}

	return subgraphRoots, nil
}

func (p *Puller) NextCids(roots []gocid.Cid) (pullRoots []gocid.Cid, remainingRoots []gocid.Cid) {
	rootsSet := gocid.NewSet()

	for _, cid := range roots {
		if rootsSet.Visit(cid) {
			if len(pullRoots) < int(p.maxBlocksPerRound) {
				pullRoots = append(pullRoots, cid)
			} else {
				remainingRoots = append(remainingRoots, cid)
			}
		}
	}

	return
}

func (p *Puller) UniqueLocalCids(rootCids []gocid.Cid) (cids []gocid.Cid) {
	cidSet := gocid.NewSet()
	nodeGetter := mdag.NewSession(p.ctx, p.lng)

	for _, cid := range rootCids {
		// cidSet.Add(cid)
		// cids = append(cids, cid)

		var rp path.Resolved
		var nd ipld.Node

		// TODO: return error
		rp, err := p.capi.ResolvePath(p.ctx, path.New(cid.String()))
		if err != nil {
			log.Debugf("Failed to resolve path for cid %v.  Ignoring.  err = %v", cid.String(), err)
			continue
		}

		nd, err = nodeGetter.Get(p.ctx, rp.Cid())
		if err != nil {
			log.Debugf("Failed to get node for cid %v.  Ignoring.  err = %v", cid.String(), err)
			continue
		}

		err = traverse.Traverse(nd, traverse.Options{
			DAG:   nodeGetter,
			Order: traverse.BFS, // Does order matter?
			Func: func(current traverse.State) error {
				if cidSet.Visit(current.Node.Cid()) {
					cids = append(cids, current.Node.Cid())
				}

				return nil
			},
			ErrFunc:        nil,
			SkipDuplicates: true,
		})
		if err != nil {
			log.Debugf("error traversing DAG for root CID %v.  err=%v", err)
			continue
		}
	}

	return
}
