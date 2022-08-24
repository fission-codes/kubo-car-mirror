package carmirror

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/fission-codes/go-car-mirror/payload"
	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	carv1 "github.com/ipld/go-car"
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

	resMsg := string(resBytes)
	log.Debugf("expected response to be nil, got %v", resMsg)

	return nil
}

// func (rem *HTTPClient) NewPushSession() (PushSession, error) {
// 	return &httpPushSession{
// 		rem: rem,
// 	}, nil
// }

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
		// log.Debugf("decoded payload = %v", pushRequest)

		// w.WriteHeader(http.StatusOK)
	}
}

func HTTPRemotePullHandler(cm *CarMirror) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("In HTTPRemotePullHandler")
		w.Header().Set(httpCarMirrorProtocolIDHeader, string(CarMirrorProtocolID))
		w.WriteHeader(http.StatusOK)
	}
}

// HTTPRemoteHandler exposes a CarMirror remote over HTTP by exposing a HTTP handler
// that interlocks with methods exposed by HTTPClient
func HTTPRemoteHandler(ds *CarMirror) http.HandlerFunc {
	// TODO: Add handler for /push and /pull
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpCarMirrorProtocolIDHeader, string(CarMirrorProtocolID))

		// switch r.Method {
		// case http.MethodPost:
		// 	createDsyncSession(ds, w, r)
		// case http.MethodPut:
		// 	if r.Header.Get("Content-Type") == carMIMEType {
		// 		if err := ds.ReceiveBlocks(r.Context(), r.FormValue("sid"), r.Body); err != nil {
		// 			w.WriteHeader(http.StatusBadRequest)
		// 			w.Write([]byte(err.Error()))
		// 			return
		// 		}
		// 		w.WriteHeader(http.StatusOK)
		// 		return
		// 	}

		// 	receiveBlockHTTP(ds, w, r)
		// case http.MethodGet:
		// 	mfstID := r.FormValue("manifest")
		// 	blockID := r.FormValue("block")
		// 	if mfstID == "" && blockID == "" {
		// 		w.WriteHeader(http.StatusBadRequest)
		// 		w.Write([]byte("either manifest or block query params are required"))
		// 	} else if mfstID != "" {

		// 		meta := map[string]string{}
		// 		for key := range r.URL.Query() {
		// 			if key != "manifest" {
		// 				meta[key] = r.URL.Query().Get(key)
		// 			}
		// 		}

		// 		mfst, err := ds.GetDagInfo(r.Context(), mfstID, meta)
		// 		if err != nil {
		// 			w.WriteHeader(http.StatusInternalServerError)
		// 			w.Write([]byte(err.Error()))
		// 			return
		// 		}

		// 		data, err := json.Marshal(mfst)
		// 		if err != nil {
		// 			w.WriteHeader(http.StatusInternalServerError)
		// 			w.Write([]byte(err.Error()))
		// 			return
		// 		}

		// 		w.Header().Set("Content-Type", jsonMIMEType)
		// 		w.Write(data)
		// 	} else {
		// 		data, err := ds.GetBlock(r.Context(), blockID)
		// 		if err != nil {
		// 			w.WriteHeader(http.StatusInternalServerError)
		// 			w.Write([]byte(err.Error()))
		// 			return
		// 		}
		// 		w.Header().Set("Content-Type", binaryMIMEType)
		// 		w.Write(data)
		// 	}
		// case http.MethodPatch:
		// 	meta := map[string]string{}
		// 	for key := range r.URL.Query() {
		// 		meta[key] = r.URL.Query().Get(key)
		// 	}

		// 	info, err := decodeDAGInfoBody(r)
		// 	if err != nil {
		// 		w.WriteHeader(http.StatusBadRequest)
		// 		w.Write([]byte(err.Error()))
		// 		return
		// 	}
		// 	r, err := ds.OpenBlockStream(r.Context(), info, meta)
		// 	if err != nil {
		// 		w.WriteHeader(http.StatusBadRequest)
		// 		w.Write([]byte(err.Error()))
		// 		return
		// 	}

		// 	w.Header().Set("Content-Type", carMIMEType)
		// 	w.WriteHeader(http.StatusOK)
		// 	defer r.Close()
		// 	io.Copy(w, r)
		// 	return

		// case http.MethodDelete:
		// 	cid := r.FormValue("cid")
		// 	meta := map[string]string{}
		// 	for key := range r.URL.Query() {
		// 		if key != "cid" {
		// 			meta[key] = r.URL.Query().Get(key)
		// 		}
		// 	}

		// 	if err := ds.RemoveCID(r.Context(), cid, meta); err != nil {
		// 		w.WriteHeader(http.StatusInternalServerError)
		// 		w.Write([]byte(err.Error()))
		// 		return
		// 	}

		// 	w.WriteHeader(http.StatusOK)
		// }
	}
}
