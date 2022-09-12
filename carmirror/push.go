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

const (
	defaultNumBlocksPerPush = 5
)

type Push struct {
	stream bool
	// sid    string // session ID for this push
	diff   string
	cids   []gocid.Cid
	lng    ipld.NodeGetter // local NodeGetter (Block Getter)
	remote CarMirrorable   // place we're sending to
}

// NewPush initiates a push to a remote.
func NewPush(lng ipld.NodeGetter, cids []gocid.Cid, diff string, remote CarMirrorable, stream bool) *Push {
	push := &Push{
		stream: stream,
		lng:    lng,
		remote: remote,
		cids:   cids,
		diff:   diff,
	}
	return push
}

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
	numBlocksPerPush          int64
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

func NewPusher(ctx context.Context, lng ipld.NodeGetter, capi coreiface.CoreAPI, cids []gocid.Cid, diff string, stream bool, remote CarMirrorable) *Pusher {
	pusher := &Pusher{
		ctx:                       ctx,
		lng:                       lng,
		capi:                      capi,
		remainingRoots:            cids,
		diff:                      diff,
		stream:                    stream,
		remote:                    remote,
		currentRound:              0,
		numBlocksPerPush:          defaultNumBlocksPerPush,
		providerGraphConfirmation: nil,
	}
	return pusher
}

// func (p *Pusher) Estimate() (err error) {
// 	// The sending Requestor begins with a local phase estimating what the Provider has in its store. This may be from stateful information (e.g. Bloom filters and CIDs) learned in a previous round, or by using a heuristic such as knowing that the provider has a previous copy associated with an IPNS record or DNSLink. If no information is available, the estimate is the empty set.

// 	return nil
// }

func (p *Pusher) Next() bool {
	return len(p.remainingRoots) != 0
}

// NextCids returns the next grouping of CIDs to push, remaining CIDs that aren't getting pushed, and remaining CID roots that encompass all remaining CIDs to push.
// These lists of CIDs are created based on the value of numBlocksPerPush.
func (p *Pusher) NextCids() (pushCids []gocid.Cid, remainingCids []gocid.Cid, remainingRoots []gocid.Cid, err error) {

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
				if len(pushCids) < int(p.numBlocksPerPush) {
					// TODO: if p.providerGraphConfirmation exists, don't add cids to push list if provider already has them in this confirmation bloom
					// TODO: Also exclude dupes from past sends, since SkipDuplicates in this traverse doesn't span past pushes.
					pushCids = append(pushCids, current.Node.Cid())
					lastDepth = current.Depth
				} else {
					if current.Depth == lastDepth {
						remainingRoots = append(remainingRoots, current.Node.Cid())
					}
					remainingCids = append(remainingCids, current.Node.Cid())
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

	pushCids, remainingCids, remainingRoots, err := p.NextCids()
	if err != nil {
		return errors.Wrapf(err, "unable to get next CIDs")
	}

	var providerGraphEstimate *bloom.Filter
	if p.currentRound == 0 {
		// Cold start.  Create bloom of remainingCids for payload
		n := uint64(len(remainingCids))
		providerGraphEstimate = bloom.NewFilterWithEstimates(n, 0.0001)
		for _, cid := range remainingCids {
			providerGraphEstimate.Add(cid.Bytes())
		}
		log.Debugf("round 0, building providerGraphEstimate from all CIDs")
	} else if p.providerGraphConfirmation != nil {
		// TODO: update this
		providerGraphEstimate = p.providerGraphConfirmation
		log.Debugf("round > 0 and providerGraphConfirmation not nil, setting providerGraphEstimate to providerGraphConfirmation")
	} else {
		log.Debugf("p.providerGraphConfirmation is nil, so not setting providerGraphEstimate")
	}

	// Send payload to provider, with provider returning providerGraphConfirmation bloom and SR
	// TODO: Probably need status of response too
	providerGraphConfirmation, subgraphRoots, err := p.remote.Push(p.ctx, pushCids, providerGraphEstimate, p.diff)
	if err != nil {
		return err
	}
	log.Debugf("providerGraphConfirmation=%v, subgraphRoots=%v", providerGraphConfirmation, subgraphRoots)
	p.providerGraphConfirmation = providerGraphConfirmation

	// Set p.remainingCids to remainingRoots + SR
	// Maybe set SR to something in state, because if empty and still have remainingCids, we need to send nil bloom.
	p.remainingRoots = append(remainingRoots, subgraphRoots...)
	log.Debugf("remainingRoots = %v", p.remainingRoots)

	// Return nil if no error

	p.currentRound += 1

	return nil
}
