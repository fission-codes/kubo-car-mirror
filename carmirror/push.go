package carmirror

import (
	"context"

	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
)

const (
	defaultNumBlocksPerPush = 5
)

type Push struct {
	stream bool
	sid    string // session ID for this push
	diff   string
	cids   []cid.Cid
	lng    ipld.NodeGetter // local NodeGetter (Block Getter)
	remote CarMirrorable   // place we're sending to
}

// NewPush initiates a push to a remote.
func NewPush(lng ipld.NodeGetter, cids []cid.Cid, diff string, remote CarMirrorable, stream bool) *Push {
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
func (push *Push) Do(ctx context.Context) (err error) {
	log.Debugf("initiating push: stream=%v, remote=%v, cids=%v, diff=%v", push.stream, push.remote, push.cids, push.diff)

	// Create new push session, save session id on the push, then call push.remote.Push
	// push.sid, push.diff, err = push.remote.NewReceiveSession(push.info, push.pinOnComplete, push.meta)

	// dsync creates a bunch of senders that have sid and send data.  They receive blocks and track with sid.
	// we need to track blocks / cids by sid for ttl purposes
	// we need to update ttl with last block received for sid
	// we need to use caches tied to sessions as well
	// or do caches need to be unique to a session?  maybe some sid specific, some global?

	return push.remote.Push(ctx, push.cids, push.diff)
}

type Pusher struct {
	lng              ipld.NodeGetter
	cids             []cid.Cid
	diff             string
	stream           bool
	remote           CarMirrorable
	numBlocksPerPush int64
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

func NewPusher(lng ipld.NodeGetter, cids []cid.Cid, diff string, stream bool, remote CarMirrorable) *Pusher {
	pusher := &Pusher{
		lng:    lng,
		cids:   cids,
		diff:   diff,
		stream: stream,
		remote: remote,
		// TODO: Make this configurable
		numBlocksPerPush: defaultNumBlocksPerPush,
	}
	return pusher
}

func (p *Pusher) Next() bool {
	return len(p.cids) == 0
}

func (p *Pusher) Value() {
	// Return a Push, which can then be Do`d
}
