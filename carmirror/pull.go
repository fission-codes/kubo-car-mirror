package carmirror

import (
	"context"

	"github.com/fission-codes/go-car-mirror/bloom"
	"github.com/fission-codes/go-car-mirror/dag"
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
	cleanupCids       []gocid.Cid
	bloom             *bloom.Filter
	remote            CarMirrorable
	currentRound      int64
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
	}
	return puller
}

func (p *Puller) Next() bool {
	return len(p.remainingRoots)+len(p.cleanupCids) > 0
}

func (p *Puller) ShouldCleanup() bool {
	return len(p.remainingRoots) == 0 && len(p.cleanupCids) > 0
}

func (p *Puller) Pull() (err error) {
	_, err = p.DoPull(p.remainingRoots, true)
	if err != nil {
		return err
	}
	return nil
}

func (p *Puller) Cleanup() (err error) {
	log.Debugf("Puller.Cleanup")
	pullCids, err := p.DoPull(p.cleanupCids, false)
	if err != nil {
		return err
	}

	// Remove pulled cids from cleanupCids
	var newCleanupCids []gocid.Cid
	cleanupCidsSet := gocid.NewSet()
	for _, cid := range p.cleanupCids {
		cleanupCidsSet.Add(cid)
	}
	for _, cid := range pullCids {
		if !cleanupCidsSet.Has(cid) {
			newCleanupCids = append(newCleanupCids, cid)
		}
	}
	p.cleanupCids = newCleanupCids

	return nil
}

func (p *Puller) DoPull(roots []gocid.Cid, includeBloom bool) (pullCids []gocid.Cid, err error) {
	log.Debugf("Puller.Pull")

	pullRoots, remainingRoots := p.NextCids(roots)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get next CIDs")
	}
	log.Debugf("pullRoots=%v", pullRoots)
	log.Debugf("remainingRoots=%v", remainingRoots)

	// Need to decide when to recreate due to saturation
	if includeBloom {
		if p.currentRound == 0 {
			bloomCids := p.UniqueLocalCids(append(p.pullRoots, p.sharedRoots...))
			n := uint64(len(bloomCids))
			p.bloom = bloom.NewFilterWithEstimates(n, 0.0001)
			for _, cid := range bloomCids {
				p.bloom.Add(cid.Bytes())
			}
			log.Debugf("Cold start.  Building bloom from all known local CIDs.")
		} else if p.bloom != nil {
			// TODO: If we don't update it elsewhere, make sure the blm is a bloom that combines all returned confirmations into a new bloom with all of their adds.
			log.Debugf("round > 0 and bloom not nil, setting blm to bloom")
		} else {
			log.Debugf("p.bloom is nil, so not setting blm")
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

	// Compute subgraph roots from those we pulled, and apend to remaining roots
	subgraphRoots, err := dag.SubgraphRoots(p.ctx, p.lng, pulledCids)
	if err != nil {
		return
	}
	p.remainingRoots = append(remainingRoots, subgraphRoots...)

	// This won't be needed once I clean up where blooms are set
	p.currentRound += 1

	return pullCids, nil
}

func (p *Puller) NextCids(roots []gocid.Cid) (pullRoots []gocid.Cid, remainingRoots []gocid.Cid) {
	rootsSet := gocid.NewSet()

	for _, cid := range roots {
		log.Debugf("roots: cid = %v", cid)
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
		if cidSet.Visit(cid) {
			cids = append(cids, cid)

			var rp path.Resolved
			var nd ipld.Node

			rp, err := p.capi.ResolvePath(p.ctx, path.New(cid.String()))
			if err != nil {
				continue
			}

			nd, err = nodeGetter.Get(p.ctx, rp.Cid())
			if err != nil {
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
	}

	return
}
