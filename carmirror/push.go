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

	return push.remote.Push(ctx, push.cids)
}
