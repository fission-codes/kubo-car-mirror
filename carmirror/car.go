package carmirror

import (
	"bytes"
	"context"
	"io"

	"github.com/ipfs/go-cid"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipld/go-car"
)

func AddAllFromCarReader(ctx context.Context, bapi coreiface.BlockAPI, r io.Reader, progCh chan cid.Cid) (int, []cid.Cid, error) {
	cids := []cid.Cid{}
	rdr, err := car.NewCarReader(r)
	if err != nil {
		return 0, cids, err
	}

	added := 0
	buf := &bytes.Buffer{}
	for {
		blk, err := rdr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return added, cids, err
		}

		if _, err := buf.Write(blk.RawData()); err != nil {
			return added, cids, err
		}
		if _, err = bapi.Put(ctx, buf); err != nil {
			return added, cids, err
		}

		buf.Reset()
		added++
		cids = append(cids, blk.Cid())

		log.Debugf("wrote block %s", blk.Cid())
		if progCh != nil {
			go func() { progCh <- blk.Cid() }()
		}
	}

	return added, cids, nil
}
