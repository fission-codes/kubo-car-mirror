package server

import (
	"fmt"
	"log"
	"net/http"
)

func Root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "/")
}

func DagPush(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "/dag/push")
}

func DagPull(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "/dag/pull")
}

func Serve() error {
	server := http.Server{
		Addr: "127.0.0.1:8080",
	}

	http.HandleFunc("/api/v0/dag/push", DagPush)
	http.HandleFunc("/api/v0/dag/pull", DagPull)
	http.HandleFunc("/api/v0/", Root)
	http.HandleFunc("/", Root)

	log.Print("Serving API on http://localhost:8080")
	return server.ListenAndServe()
}
