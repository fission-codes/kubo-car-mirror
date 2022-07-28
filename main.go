package main

import (
	"log"

	"github.com/fission-suite/car-mirror/server"
)

func main() {
	log.Fatal(server.ServeHTTP())
}
