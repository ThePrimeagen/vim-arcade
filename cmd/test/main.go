package main

import (
	"fmt"
)

func main() {
    fmt.Printf("%016x\n", 10)
    fmt.Printf("%016x\n", 100)
    fmt.Printf("%016x\n", 1000)
    fmt.Printf("%016x\n", 10000)
    fmt.Printf("%016x\n", 100000)

    bytes := []byte(fmt.Sprintf("%016x", 100000))
    fmt.Printf("Bytes(%d): %+v\n", len(bytes), bytes)
}

