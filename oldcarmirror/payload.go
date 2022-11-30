package oldcarmirror

import (
	"fmt"

	cbor "github.com/fxamacker/cbor/v2"
)

type PullRequestor struct {
	RS []string `cbor:"rs,omitempty"` // Requested CIDs
	BK uint     `cbor:"bk,omitempty"` // Bloom filter hash count
	BB []byte   `cbor:"bb,omitempty"` // Bloom filter binary
}

func (p PullRequestor) String() string {
	return fmt.Sprintf("payload.PullRequestor: rs=%v, bk=%v, bb=%X", p.RS, p.BK, p.BB)
}

type PullProvider []byte // Pull Provider payload is just a CARv1

type PushRequestor struct {
	BK uint   `cbor:"bk,omitempty"` // Bloom filter hash count
	BB []byte `cbor:"bb,omitempty"` // Bloom filter binary
	PL []byte `cbor:"pl,omitempty"` // Data payload, CARv1
}

func (p PushRequestor) String() string {
	return fmt.Sprintf("payload.PushRequestor: bk=%v, bb=%X, pl=%v", p.BK, p.BB, p.PL)
}

type PushProvider struct {
	SR []string `cbor:"sr,omitempty"` // Incomplete subgraph roots
	BK uint     `cbor:"bk,omitempty"` // Bloom filter hash count
	BB []byte   `cbor:"bb,omitempty"` // Bloom filter binary
}

func (p PushProvider) String() string {
	return fmt.Sprintf("payload.PushProvider: sr=%v, bk=%v, bb=%X", p.SR, p.BK, p.BB)
}

// CborEncode encodes the payload in CBOR.
func CborEncode(pl interface{}) ([]byte, error) {
	if m, err := cbor.Marshal(pl); err != nil {
		return nil, err
	} else {
		return m, nil
	}
}

// CborDecode decodes the payload from CBOR.
func CborDecode(b []byte, v interface{}) error {
	if err := cbor.Unmarshal(b, v); err != nil {
		return err
	}
	return nil
}
