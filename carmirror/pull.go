package carmirror

import (
	"context"

	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
)

type Pull struct {
	stream bool
	cids   []cid.Cid
	lng    ipld.NodeGetter // local NodeGetter (Block Getter)
	remote CarMirrorable   // place we're sending to
}

// NewPull initiates a pull to a remote.
func NewPull(lng ipld.NodeGetter, cids []cid.Cid, remote CarMirrorable, stream bool) *Pull {
	pull := &Pull{
		stream: stream,
		lng:    lng,
		remote: remote,
		cids:   cids,
	}
	return pull
}

// Do executes the pull, blocking until complete
func (pull *Pull) Do(ctx context.Context) (err error) {
	log.Debugf("initialiating pull")

	return pull.remote.Pull(ctx, pull.cids)
}
