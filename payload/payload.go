package payload

import (
	cbor "github.com/fxamacker/cbor/v2"
)

type PullRequestor struct {
	RS []string `cbor:"rs,omitempty"` // Requested CIDs
	BK uint     `cbor:"bk,omitempty"` // Bloom filter hash count
	BB []byte   `cbor:"bb,omitempty"` // Bloom filter binary
}

type PullProvider []byte // Pull Provider payload is just a CARv1

type PushRequestor struct {
	BK uint   `cbor:"bk,omitempty"` // Bloom filter hash count
	BB []byte `cbor:"bb,omitempty"` // Bloom filter binary
	PL []byte `cbor:"pl,omitempty"` // Data payload, CARv1
}

type PushProvider struct {
	SR []string `cbor:"sr,omitempty"` // Incomplete subgraph roots
	BK uint     `cbor:"bk,omitempty"` // Bloom filter hash count
	BB []byte   `cbor:"bb,omitempty"` // Bloom filter binary
}

// Encode the payload
func Encode(pl interface{}) ([]byte, error) {
	if m, err := cbor.Marshal(pl); err != nil {
		return nil, err
	} else {
		return m, nil
	}
}

// Decode the payload
func Decode(b []byte, v interface{}) error {
	if err := cbor.Unmarshal(b, v); err != nil {
		return err
	}
	return nil
}
