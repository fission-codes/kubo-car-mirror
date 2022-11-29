package carmirror

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/fission-codes/go-bloom"
	"github.com/fission-codes/kubo-car-mirror/dag"
	"github.com/fission-codes/kubo-car-mirror/payload"
	"github.com/ipfs/go-cid"
	gocid "github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/zeebo/xxh3"
)

var (
	_ CarMirrorable = (*HTTPClient)(nil)
)

// HTTPClient is the request side of doing dsync over HTTP
type HTTPClient struct {
	URL        string
	NodeGetter format.NodeGetter
	BlockAPI   coreiface.BlockAPI
	// remProtocolID protocol.ID
}

const (
	httpCarMirrorProtocolIDHeader = "car-mirror-version"
	carMIMEType                   = "archive/car"
	cborMIMEType                  = "application/cbor"
)

func (rem *HTTPClient) Push(ctx context.Context, cids []cid.Cid, providerGraphEstimate *bloom.Filter[[]byte, bloom.HashFunction[[]byte]], diff string) (providerGraphConfirmation *bloom.Filter[[]byte, bloom.HashFunction[[]byte]], subgraphRoots []gocid.Cid, err error) {
	log.Debugf("HTTPClient.Push")

	var b bytes.Buffer
	w := bufio.NewWriter(&b)

	if err = WriteCar(ctx, rem.NodeGetter, cids, w); err != nil {
		log.Debugf("error while writing car file: err=%v", err.Error())
		return
	}

	// We must flush the buffer or we could get unexpected EOF errors on the other end
	w.Flush()

	// TODO: conditional providerGraphEstimate logic, might be nil
	var pl payload.PushRequestor
	if providerGraphEstimate != nil {
		pl = payload.PushRequestor{BB: providerGraphEstimate.Bytes(), BK: uint(providerGraphEstimate.HashCount()), PL: b.Bytes()}
	} else {
		pl = payload.PushRequestor{BB: nil, BK: 0, PL: b.Bytes()}
	}
	plBytes, err := payload.CborEncode(pl)
	if err != nil {
		log.Debugf("error while encoding payload in cbor: err=%v", err.Error())
		return
	}
	plReader := bytes.NewReader(plBytes)

	var endpoint string
	if diff != "" {
		endpoint = fmt.Sprintf("%s/dag/push?diff=%s", rem.URL, diff)
	} else {
		endpoint = fmt.Sprintf("%s/dag/push", rem.URL)
	}
	req, err := http.NewRequest("POST", endpoint, plReader)
	req.Header.Set("Content-Type", cborMIMEType)
	req.Header.Set("Accept", cborMIMEType)

	if err != nil {
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	resBytes, err := ioutil.ReadAll(res.Body)
	if resBytes == nil {
		return
	}

	var pushProvider payload.PushProvider
	if err = payload.CborDecode(resBytes, &pushProvider); err != nil {
		return
	}

	subgraphRoots, err = dag.ParseCids(pushProvider.SR)
	if err != nil {
		return
	}

	providerGraphConfirmation = bloom.NewFilterFromBloomBytes[[]byte, bloom.HashFunction[[]byte]](uint64(len(pushProvider.BB)*8), uint64(pushProvider.BK), pushProvider.BB, xxh3.HashSeed)
	if err != nil {
		return
	}

	return
}

func (rem *HTTPClient) Pull(ctx context.Context, cids []cid.Cid) error {
	// create payload
	cidStrs := make([]string, len(cids))
	for i, c := range cids {
		cidStrs[i] = c.String()
	}
	pullRequest := payload.PullRequestor{RS: cidStrs, BK: 0, BB: nil}
	plBytes, err := payload.CborEncode(pullRequest)
	if err != nil {
		return err
	}
	plReader := bytes.NewReader(plBytes)

	// request the pull
	url := fmt.Sprintf("%s%s", rem.URL, "/dag/pull")
	req, err := http.NewRequest("POST", url, plReader)
	req.Header.Set("Content-Type", cborMIMEType)
	req.Header.Set("Accept", carMIMEType)

	log.Debugf("req = %v", req)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	log.Debugf("res = %v", res)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// receive the car payload
	log.Debugf("before reading all body, err=%v", err)
	resBytes, err := ioutil.ReadAll(res.Body)
	if resBytes == nil {
		return err
	}

	// add car to local blockstore
	_, _, err = AddAllFromCarReader(ctx, rem.BlockAPI, bytes.NewReader(resBytes), nil)
	if err != nil {
		// getting unexpected EOF as err here
		log.Debugf("error in AddAllFromCarReader: err=%v", err.Error())
		return err
	}

	return nil
}

func (cm *CarMirror) HTTPRemotePushHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: parse diff param
		// diff := r.FormValue("diff")
		log.Debugf("In HTTPRemotePushHandler")
		w.Header().Set(httpCarMirrorProtocolIDHeader, string(CarMirrorProtocolID))

		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Debugf("could not read body: err=%v", err)
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var pushRequest payload.PushRequestor
		if err := payload.CborDecode(data, &pushRequest); err != nil {
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// TODO: save root CIDs from CAR so we can walk them and construct bloom filter
		_, cids, err := AddAllFromCarReader(r.Context(), cm.bapi, bytes.NewReader(pushRequest.PL), nil)
		if err != nil {
			// getting unexpected EOF as err here
			log.Debugf("error in AddAllFromCarReader: err=%v", err.Error())
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		subgraphRoots, err := cm.SubgraphRoots(r.Context(), cids)
		if err != nil {
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		subgraphRootsStr := make([]string, len(subgraphRoots))
		for i, c := range subgraphRoots {
			subgraphRootsStr[i] = c.String()
		}

		// TODO: Use diff to generate a bloom filter to return in the pushProvider payload
		// Resolve diff
		// Locally traverse DAG underneath diff and get list of CIDs, adding them to bloom
		// Also collect list of subgraph roots to return in SR in the payload
		// (relative to the CIDs in the push? Or in the diff?)  For all CIDs pushed for the entire session or the request?

		// Start with subgraphRoots
		var providerGraphConfirmation *bloom.Filter[[]byte, bloom.HashFunction[[]byte]]
		bloomCids, err := cm.GetLocalCids(r.Context(), subgraphRoots)
		if err != nil {
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		n := uint64(len(bloomCids) * 8)
		providerGraphConfirmation, err = bloom.NewFilterWithEstimates[[]byte, bloom.HashFunction[[]byte]](n, 0.0001, xxh3.HashSeed)
		if err != nil {
			log.Debugf("error in bloom.NewFilterWithEstimates: err=%v", err.Error())
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, cid := range bloomCids {
			providerGraphConfirmation.Add(cid.Bytes())
		}

		// On success, return the PushProviderPayload, for now with nothing of interest
		pushProvider := payload.PushProvider{SR: subgraphRootsStr, BK: uint(providerGraphConfirmation.HashCount()), BB: providerGraphConfirmation.Bytes()}
		log.Debugf("pushProvider=%v", pushProvider)
		pushProviderBytes, err := payload.CborEncode(pushProvider)
		if err != nil {
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(pushProviderBytes)

		// Complete is 200.  Success is 202.
	}
}

func (cm *CarMirror) SubgraphRoots(ctx context.Context, cids []cid.Cid) (subgraphRoots []cid.Cid, err error) {
	subgraphRootsMap := make(map[cid.Cid]bool)
	// convert cids to hashmap efficient membership checking
	cidsMap := make(map[cid.Cid]bool)
	for _, c := range cids {
		cidsMap[c] = true
	}

	// iterate through cids
	// if they have links and any links are not in cids, add to subgraphRoots, ignoring dupes
	var node format.Node
	for _, c := range cids {
		node, err = cm.lng.Get(ctx, c)
		if err != nil {
			return
		}
		for _, link := range node.Links() {
			if _, ok := cidsMap[link.Cid]; !ok {
				subgraphRootsMap[link.Cid] = true
			}
		}
	}
	// convert subgraph roots map back to slice
	subgraphRoots = make([]cid.Cid, len(subgraphRootsMap))
	i := 0
	for k := range subgraphRootsMap {
		subgraphRoots[i] = k
		i++
	}

	return
}

func (cm *CarMirror) HTTPRemotePullHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("In HTTPRemotePullHandler")
		w.Header().Set(httpCarMirrorProtocolIDHeader, string(CarMirrorProtocolID))

		// decode the cbor request
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Debugf("could not read body: err=%v", err)
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var pullRequest payload.PullRequestor
		if err := payload.CborDecode(data, &pullRequest); err != nil {
			log.Debugf("could not decode cbor")
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		cidStrs := pullRequest.RS
		cids, err := dag.ParseCids(cidStrs)
		if err != nil {
			log.Debugf("could not parse cids: cidStrs=%v", cidStrs)
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Create a car file from the requested cids
		var b bytes.Buffer
		bw := bufio.NewWriter(&b)

		if err := WriteCar(r.Context(), cm.lng, cids, bw); err != nil {
			log.Debugf("error while writing car file: err=%v", err.Error())
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		bw.Flush()

		// return the car file
		w.Write(b.Bytes())

		// TODO: Return 404 if unable to find any new CID roots.  Otherwise 200.
	}
}
