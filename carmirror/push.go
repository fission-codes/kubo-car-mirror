package carmirror

import (
	"context"

	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
)

type Push struct {
	stream bool
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

	// At the very least we probably need session ids for checking progress for long running requests

	// Future session stuff maybe
	// err = push.remote.NewPushSession(push.cid)
	// if err != nil {
	// 	log.Debugf("error creating push session: %s", err)
	// 	return err
	// }
	// log.Debugf("push has receive session: %s", push.sid)

	return push.remote.Push(ctx, push.cids)
}
