package main

import (
	"fmt"
	"time"
)

func panicme(i int) {
    if i > 3 {
        panic("OH NO DONKEYKONG")
    }
    fmt.Printf("panicme finished\n")
}

func main() {
    fmt.Printf("hello world\n")
    go panicme(0)
    go panicme(1)
    go panicme(2)
    go panicme(3)
    go panicme(4)

    time.Sleep(time.Second)
    fmt.Printf("goodbye world\n")
}

