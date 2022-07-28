package main

import (
	"log"

	"github.com/fission-codes/go-car-mirror/server"
)

func main() {
	log.Fatal(server.ServeHTTP())
}
