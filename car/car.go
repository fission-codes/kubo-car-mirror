package car

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	blocks "github.com/ipfs/go-block-format"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-merkledag"
	carv2 "github.com/ipld/go-car/v2"
	"github.com/ipld/go-car/v2/blockstore"
)

// Create CARv1 pull requestor payload
func CreatePullRequestorPayload(cids []cid.Cid, bk uint, bb []byte) ([]byte, error) {
	return nil, nil
}

// func CreatePushRequestorPayload(cids []cid.Cid, k uint, bb []byte)

// Do I need an io.Writer?
func CreateCar(cids []cid.Cid) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// thisRawNode := merkledag.NewRawNode([]byte("fish"))
	thisBlock := merkledag.NewRawNode([]byte("fish")).Block
	thatBlock := merkledag.NewRawNode([]byte("lobster")).Block
	andTheOtherBlock := merkledag.NewRawNode([]byte("barreleye")).Block

	tdir, err := ioutil.TempDir(os.TempDir(), "example-*")
	if err != nil {
		panic(err)
	}
	dst := filepath.Join(tdir, "sample-rw-bs-v2.car")
	roots := []cid.Cid{thisBlock.Cid(), thatBlock.Cid(), andTheOtherBlock.Cid()}

	rwbs, err := blockstore.OpenReadWrite(dst, roots, carv2.UseDataPadding(1413), carv2.UseIndexPadding(42))
	if err != nil {
		panic(err)
	}

	// Put all blocks onto the blockstore.
	blocks := []blocks.Block{thisBlock, thatBlock}
	if err := rwbs.PutMany(ctx, blocks); err != nil {
		panic(err)
	}

	// This writes the car file to disk
	rwbs.Finalize()

	// Any blocks put can be read back using the same blockstore instance.
	// block, err := rwbs.Get(ctx, thatBlock.Cid())
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("Read back block just put with raw value of `%v`.\n", string(block.RawData()))
}
