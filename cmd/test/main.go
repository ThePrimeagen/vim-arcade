package main

import (
	"fmt"
	"sync"
)

func main() {
    wait := sync.WaitGroup{}

    wait.Add(2)
    wait.Done()
    wait.Done()
    wait.Done()
    wait.Done()

    wait.Wait()

    fmt.Println("DONE")

}

