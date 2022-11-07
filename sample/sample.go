package main

import (
	"fmt"

	"github.com/fission-codes/kubo-car-mirror/bloom"
)

func main() {
	// uncomment the indices printing and then run this

	f1 := bloom.NewFilter(1000, 4)
	fmt.Printf("indices for 'one':\n")
	f1.Add([]byte("one"))
	fmt.Printf("indices for 'two':\n")
	f1.Add([]byte("two"))
	fmt.Printf("indices for 'three':\n")
	f1.Add([]byte("three"))
	fmt.Printf("f1: %X\n", f1.Bytes())

	f2 := bloom.NewFilter(10, 3)
	fmt.Printf("indices for 'ducks':\n")
	f2.Add([]byte("ducks"))
	fmt.Printf("indices for 'chickens':\n")
	f2.Add([]byte("chickens"))
	fmt.Printf("indices for 'goats':\n")
	f2.Add([]byte("goats"))
	fmt.Printf("f2: %X\n", f2.Bytes())
}
