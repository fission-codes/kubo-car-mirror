package carmirror

import (
	"context"

	"github.com/fission-codes/go-car-mirror/bloom"
	gocid "github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	mdag "github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-merkledag/traverse"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/pkg/errors"
)

// Do executes the push, blocking until complete
// func (push *Push) Do(ctx context.Context) (err error) {
// 	log.Debugf("initiating push: stream=%v, remote=%v, cids=%v, diff=%v", push.stream, push.remote, push.cids, push.diff)

// 	// Create new push session, save session id on the push, then call push.remote.Push
// 	// push.sid, push.diff, err = push.remote.NewReceiveSession(push.info, push.pinOnComplete, push.meta)

// 	// dsync creates a bunch of senders that have sid and send data.  They receive blocks and track with sid.
// 	// we need to track blocks / cids by sid for ttl purposes
// 	// we need to update ttl with last block received for sid
// 	// we need to use caches tied to sessions as well
// 	// or do caches need to be unique to a session?  maybe some sid specific, some global?

// 	return push.remote.Push(ctx, push.cids, push.diff)
// }

type Pusher struct {
	ctx                       context.Context
	lng                       ipld.NodeGetter
	capi                      coreiface.CoreAPI
	remainingRoots            []gocid.Cid
	diff                      string
	stream                    bool
	remote                    CarMirrorable
	maxBlocksPerRound         int64
	maxBlocksPerColdCall      int64
	currentRound              int64
	providerGraphConfirmation *bloom.Filter
	// cidsSeen hashmap
	// remoteBloom
}

// type PusherOption func(p *Pusher)
// func NewPusher(opts ...PusherOption) *Pusher {
//    pusher :=  &Pusher{}
//    for _, opt := range opts {
//       opt(pusher)
//    }
//    return pusher
// }

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

func (p *Pusher) Next() bool {
	log.Debugf("p.remainingRoots=%v", p.remainingRoots)
	log.Debugf("p.providerGraphConfirmation=%v", p.providerGraphConfirmation)

	for _, cid := range p.remainingRoots {
		if p.providerGraphConfirmation == nil {
			log.Debugf("p.providerGraphConfirmation is nil")
			log.Debugf("Next returning true")
			return true
		}
		if len(p.providerGraphConfirmation.Bytes()) == 0 {
			log.Debugf("p.providerGraphConfirmation is size 0")
			log.Debugf("Next returning true")
			return true
		}
		if !p.providerGraphConfirmation.Test(cid.Bytes()) {
			log.Debugf("CID %v not found in p.providerGraphConfirmation", cid.String())
			log.Debugf("Next returning true")
			return true
		} else {
			log.Debugf("CID %v found in p.providerGraphConfirmation", cid.String())
		}
	}

	log.Debugf("Next returning false")
	return false
}

// NextCids returns the next grouping of CIDs to push, remaining CIDs that aren't getting pushed, and remaining CID roots that encompass all remaining CIDs to push.
// These lists of CIDs are created based on the value of maxBlocksPerRound.
func (p *Pusher) NextCids() (pushCids []gocid.Cid, remainingCids []gocid.Cid, remainingRoots []gocid.Cid, err error) {
	pushCidsSet := gocid.NewSet()
	remainingCidsSet := gocid.NewSet()
	remainingRootsSet := gocid.NewSet()
	var maxBlocks uint64
	if p.currentRound == 0 {
		maxBlocks = uint64(p.maxBlocksPerColdCall)
	} else {
		maxBlocks = uint64(p.maxBlocksPerRound)
	}

	for _, cid := range p.remainingRoots {
		var rp path.Resolved
		var nd ipld.Node
		var lastDepth int
		lastDepth = 0

		rp, err = p.capi.ResolvePath(p.ctx, path.New(cid.String()))
		if err != nil {
			err = errors.Wrapf(err, "unable to resolve path for root cid %s", cid.String())
			return
		}

		nodeGetter := mdag.NewSession(p.ctx, p.lng)
		nd, err = nodeGetter.Get(p.ctx, rp.Cid())
		if err != nil {
			err = errors.Wrapf(err, "unable to get nodes for root cid %s", cid.String())
			return
		}

		err = traverse.Traverse(nd, traverse.Options{
			DAG:   nodeGetter,
			Order: traverse.BFS, // Breadth first
			Func: func(current traverse.State) error {
				// Never nil in current implementation.  Just empty list of bytes.
				if p.providerGraphConfirmation == nil || len(p.providerGraphConfirmation.Bytes()) == 0 || !p.providerGraphConfirmation.Test(current.Node.Cid().Bytes()) {
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
			err = errors.Wrapf(err, "error traversing DAG: %v", err)
			return
		}
	}

	return
}

// Do executes the push, blocking until complete
func (p *Pusher) Push() (err error) {
	// On a partial cold call, the Provider Graph Estimate MUST contain the entire graph minus CIDs in the initial payload. The Provider MUST respond with a Bloom filter of all CIDs that match the Provider Graph Estimate, which is called the Provider Graph Confirmation. On subsequent rounds, the Provider Graph Estimate continues to be refined until is is empty or the entire graph has been synchronized.
	log.Debugf("Pusher.Push")

	pushCids, remainingCids, remainingRoots, err := p.NextCids()
	if err != nil {
		return errors.Wrapf(err, "unable to get next CIDs")
	}
	log.Debugf("pushCids=%v", pushCids)
	log.Debugf("remainingCids=%v", remainingCids)
	log.Debugf("remainingRoots=%v", remainingRoots)

	var providerGraphEstimate *bloom.Filter
	if p.currentRound == 0 {
		// Cold start.  Create bloom of remainingCids for payload
		n := uint64(len(remainingCids))
		providerGraphEstimate = bloom.NewFilterWithEstimates(n, 0.0001)
		for _, cid := range remainingCids {
			providerGraphEstimate.Add(cid.Bytes())
		}
		log.Debugf("Cold start.  Building providerGraphEstimate from all CIDs.")
	} else if p.providerGraphConfirmation != nil {
		// TODO: update this
		log.Debugf("round > 0 and providerGraphConfirmation not nil, setting providerGraphEstimate to providerGraphConfirmation")
		providerGraphEstimate = p.providerGraphConfirmation
	} else {
		log.Debugf("p.providerGraphConfirmation is nil, so not setting providerGraphEstimate")
	}

	// Send payload to provider, with provider returning providerGraphConfirmation bloom and SR
	// TODO: Probably need status of response too
	providerGraphConfirmation, subgraphRoots, err := p.remote.Push(p.ctx, pushCids, providerGraphEstimate, p.diff)
	if err != nil {
		return err
	}
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

	// Return nil if no error

	p.currentRound += 1

	return nil
}
