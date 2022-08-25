package carmirror

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/fission-codes/go-car-mirror/dag"
	"github.com/fission-codes/go-car-mirror/payload"
	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	carv1 "github.com/ipld/go-car"
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

func (rem *HTTPClient) Push(ctx context.Context, cids []cid.Cid) error {
	log.Debugf("HTTPClient.Push")

	var b bytes.Buffer
	w := bufio.NewWriter(&b)

	if err := carv1.WriteCar(ctx, rem.NodeGetter, cids, w); err != nil {
		log.Debugf("error while writing car file: err=%v", err.Error())
		return err
	}

	// We must flush the buffer or we could get unexpected EOF errors on the other end
	w.Flush()
	pl := payload.PushRequestor{BB: nil, BK: 0, PL: b.Bytes()}
	plBytes, err := payload.CborEncode(pl)
	if err != nil {
		log.Debugf("error while encoding payload in cbor: err=%v", err.Error())
		return err
	}
	plReader := bytes.NewReader(plBytes)

	url := fmt.Sprintf("%s%s", rem.URL, "/dag/push")
	req, err := http.NewRequest("POST", url, plReader)
	req.Header.Set("Content-Type", cborMIMEType)
	req.Header.Set("Accept", cborMIMEType)

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

	log.Debugf("before reading all body, err=%v", err)
	resBytes, err := ioutil.ReadAll(res.Body)
	if resBytes == nil {
		return err
	}

	// TODO: we expect a CBOR response and should parse it for now.  Longer term it will be used for multiple rounds.
	resMsg := string(resBytes)
	log.Debugf("expected response to be nil, got %v", resMsg)

	return nil
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
	_, err = AddAllFromCarReader(ctx, rem.BlockAPI, bytes.NewReader(resBytes), nil)
	if err != nil {
		// getting unexpected EOF as err here
		log.Debugf("error in AddAllFromCarReader: err=%v", err.Error())
		return err
	}

	return nil
}

func HTTPRemotePushHandler(cm *CarMirror) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		_, err = AddAllFromCarReader(r.Context(), cm.bapi, bytes.NewReader(pushRequest.PL), nil)
		if err != nil {
			// getting unexpected EOF as err here
			log.Debugf("error in AddAllFromCarReader: err=%v", err.Error())
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// On success, return the PushProviderPayload, for now with nothing of interest
		pushProvider := payload.PushProvider{SR: []string{}, BK: 0, BB: nil}
		pushProviderBytes, err := payload.CborEncode(pushProvider)
		if err != nil {
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(pushProviderBytes)
	}
}

func HTTPRemotePullHandler(cm *CarMirror) http.HandlerFunc {
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

		if err := carv1.WriteCar(r.Context(), cm.lng, cids, bw); err != nil {
			log.Debugf("error while writing car file: err=%v", err.Error())
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		bw.Flush()

		// return the car file
		w.Write(b.Bytes())
	}
}
