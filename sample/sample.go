package main

import (
	"fmt"

	"github.com/fission-codes/go-bloom"
	"github.com/zeebo/xxh3"
)

func main() {
	// uncomment the indices printing and then run this

	var function bloom.HashFunction[[]byte] = xxh3.HashSeed
	f1, _ := bloom.NewFilter(uint64(1000), uint64(4), function)
	fmt.Printf("indices for 'one':\n")
	f1.Add([]byte("one"))
	fmt.Printf("indices for 'two':\n")
	f1.Add([]byte("two"))
	fmt.Printf("indices for 'three':\n")
	f1.Add([]byte("three"))
	fmt.Printf("f1: %X\n", f1.Bytes())

	f2, _ := bloom.NewFilter(10, 3, function)
	fmt.Printf("indices for 'ducks':\n")
	f2.Add([]byte("ducks"))
	fmt.Printf("indices for 'chickens':\n")
	f2.Add([]byte("chickens"))
	fmt.Printf("indices for 'goats':\n")
	f2.Add([]byte("goats"))
	fmt.Printf("f2: %X\n", f2.Bytes())
}
