package carmirror

import (
	"context"
	"testing"

	cm "github.com/fission-codes/go-car-mirror/core"
	cmipld "github.com/fission-codes/go-car-mirror/ipld"
	"golang.org/x/exp/slices"

	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ipfs/boxo/filestore"
	keystore "github.com/ipfs/boxo/keystore"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/bootstrap"
	"github.com/ipfs/kubo/core/coreapi"
	mock "github.com/ipfs/kubo/core/mock"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/repo"

	coreiface "github.com/ipfs/boxo/coreiface"
	"github.com/ipfs/go-datastore"
	syncds "github.com/ipfs/go-datastore/sync"
	"github.com/ipfs/kubo/config"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
)

const testPeerID = "QmTFauExutTsy4XP6JbMFcw2Wa9645HJt2bTqL6qYDCKfe"

// Copied from kubo/core/coreapi/test/api_test.go
func MakeAPISwarm(ctx context.Context, fullIdentity bool, n int) ([]coreiface.CoreAPI, error) {
	mn := mocknet.New()

	nodes := make([]*core.IpfsNode, n)
	apis := make([]coreiface.CoreAPI, n)

	for i := 0; i < n; i++ {
		var ident config.Identity
		if fullIdentity {
			sk, pk, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
			if err != nil {
				return nil, err
			}

			id, err := peer.IDFromPublicKey(pk)
			if err != nil {
				return nil, err
			}

			kbytes, err := crypto.MarshalPrivateKey(sk)
			if err != nil {
				return nil, err
			}

			ident = config.Identity{
				PeerID:  id.Pretty(),
				PrivKey: base64.StdEncoding.EncodeToString(kbytes),
			}
		} else {
			ident = config.Identity{
				PeerID: testPeerID,
			}
		}

		c := config.Config{}
		c.Addresses.Swarm = []string{fmt.Sprintf("/ip4/18.0.%d.1/tcp/4001", i)}
		c.Identity = ident
		c.Experimental.FilestoreEnabled = true

		ds := syncds.MutexWrap(datastore.NewMapDatastore())
		r := &repo.Mock{
			C: c,
			D: ds,
			K: keystore.NewMemKeystore(),
			F: filestore.NewFileManager(ds, filepath.Dir(os.TempDir())),
		}

		node, err := core.NewNode(ctx, &core.BuildCfg{
			Routing: libp2p.DHTServerOption,
			Repo:    r,
			Host:    mock.MockHostOption(mn),
			Online:  fullIdentity,
			ExtraOpts: map[string]bool{
				"pubsub": true,
			},
		})
		if err != nil {
			return nil, err
		}
		nodes[i] = node
		apis[i], err = coreapi.NewCoreAPI(node)
		if err != nil {
			return nil, err
		}
	}

	err := mn.LinkAll()
	if err != nil {
		return nil, err
	}

	bsinf := bootstrap.BootstrapConfigWithPeers(
		[]peer.AddrInfo{
			nodes[0].Peerstore.PeerInfo(nodes[0].Identity),
		},
	)

	for _, n := range nodes[1:] {
		if err := n.Bootstrap(bsinf); err != nil {
			return nil, err
		}
	}

	return apis, nil
}

func TestStore(t *testing.T) {
	if nodes, err := MakeAPISwarm(context.Background(), false, 1); err != nil {
		t.Errorf("error instantiating test API, %v", err)
	} else {
		node := nodes[0]
		var store cm.BlockStore[cmipld.Cid] = NewKuboStore(node)
		block, err := cmipld.TryBlockFromCBOR("blockityblockblock")
		if err != nil {
			t.Errorf("Error creating block %v", err)
		}
		block2, err := store.Add(context.Background(), block)
		if err != nil {
			t.Errorf("Error writing block %v", err)
		}
		if !slices.Equal(block.RawData(), block2.RawData()) {
			t.Errorf("Written block not equal to original")
		}
		block3, err := store.Get(context.Background(), block.Id())
		if err != nil {
			t.Errorf("Error retrieving block %v", err)
		}
		if !slices.Equal(block.RawData(), block3.RawData()) {
			t.Errorf("Retrieved block not equal to original")
		}
	}
}
