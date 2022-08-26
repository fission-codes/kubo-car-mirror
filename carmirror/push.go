package carmirror

import (
	"context"

	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
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
func NewPush(lng ipld.NodeGetter, cids []cid.Cid, remote CarMirrorable, stream bool) *Push {
	push := &Push{
		stream: stream,
		lng:    lng,
		remote: remote,
		cids:   cids,
	}
	return push
}

// Do executes the push, blocking until complete
func (push *Push) Do(ctx context.Context) (err error) {
	log.Debugf("initiating push: stream=%v, remote=%v, cids=%v", push.stream, push.remote, push.cids)

	// Create new push session, save session id on the push, then call push.remote.Push
	// push.sid, push.diff, err = push.remote.NewReceiveSession(push.info, push.pinOnComplete, push.meta)

	// dsync creates a bunch of senders that have sid and send data.  They receive blocks and track with sid.
	// we need to track blocks / cids by sid for ttl purposes
	// we need to update ttl with last block received for sid
	// we need to use caches tied to sessions as well
	// or do caches need to be unique to a session?  maybe some sid specific, some global?

	return push.remote.Push(ctx, push.cids)
}
