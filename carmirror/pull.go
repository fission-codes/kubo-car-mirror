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

// NewPush initiates a pull to a remote.
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

	// For now, just send a CAR file.  We'll get fancier later.

	// pull.sid, pull.diff, err = pull.remote.NewReceiveSession(pull.info, pull.pinOnComplete, pull.meta)
	// if err != nil {
	// 	log.Debugf("error creating receive session: %s", err)
	// 	return err
	// }
	// log.Debugf("pull has receive session: %s", pull.sid)
	// return pull.do(ctx)

	return pull.remote.Pull(ctx, pull.cids)
	// return nil
}
