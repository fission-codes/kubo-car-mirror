package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/fission-codes/go-car-mirror/payload"
	"github.com/julienschmidt/httprouter"
)

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "")
}

func DagPush(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	stream, err := strconv.ParseBool(r.URL.Query().Get("stream"))
	if err != nil {
		stream = false
	}

	diff := r.URL.Query().Get("diff")

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		errStr := fmt.Sprintf("Failed to read request body. Error=%v", err.Error())
		http.Error(w, errStr, 500)
		return
	}

	var pushRequest payload.PushRequestor
	if err := payload.CborDecode(body, &pushRequest); err != nil {
		errStr := fmt.Sprintf("Failed to decode CBOR. Error=%v", err.Error())
		http.Error(w, errStr, 500)
		return
	}

	fmt.Fprintf(w, "/dag/push, stream=%v, diff=%v, request=%v\n", stream, diff, pushRequest)
}

func DagPull(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	stream, err := strconv.ParseBool(r.URL.Query().Get("stream"))
	if err != nil {
		stream = false
	}

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		errStr := fmt.Sprintf("Failed to read request body. Error=%v", err.Error())
		http.Error(w, errStr, 500)
		return
	}

	var pullRequest payload.PullRequestor
	if err := payload.CborDecode(body, &pullRequest); err != nil {
		errStr := fmt.Sprintf("Failed to decode CBOR. Error=%v", err.Error())
		http.Error(w, errStr, 500)
		return
	}

	fmt.Fprintf(w, "/dag/pull, stream=%v, request=%v\n", stream, pullRequest)
}

// func Router()  {
// 	return router := httprouter.New()
// 	router.GET("/", Index)
// 	router.POST("/dag/push", DagPush)
// 	router.POST("/dag/pull", DagPull)

// 	// return router.ServerHTTP
// }

func ServeHTTP() error {
	router := httprouter.New()
	router.GET("/", Index)
	router.POST("/dag/push", DagPush)
	router.POST("/dag/pull", DagPull)

	log.Print("Serving API on http://localhost:8080")
	return http.ListenAndServe(":8080", router)
}
