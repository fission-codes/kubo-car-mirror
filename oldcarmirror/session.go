package oldcarmirror

import (
	"context"
	"math/rand"

	ipld "github.com/ipfs/go-ipld-format"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
)

type session struct {
	ctx  context.Context
	lng  ipld.NodeGetter
	bapi coreiface.BlockAPI

	id string
}

func newSession(ctx context.Context, lng ipld.NodeGetter, bapi coreiface.BlockAPI) (s *session, err error) {

	s = &session{
		id:   randStringBytesMask(10),
		ctx:  ctx,
		lng:  lng,
		bapi: bapi,
	}

	log.Debugf("created session: %s", s.id)
	return s, nil
}

// the best stack overflow answer evaarrr: https://stackoverflow.com/a/22892986/9416066
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func randStringBytesMask(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
