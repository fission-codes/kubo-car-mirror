package carmirror

import (
	"net/http"
)

const (
	httpCarMirrorProtocolIDHeader = "car-mirror-version"
)

// HTTPRemoteHandler exposes a CarMirror remote over HTTP by exposing a HTTP handler
// that interlocks with methods exposed by HTTPClient
func HTTPRemoteHandler(ds *CarMirror) http.HandlerFunc {
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
