package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
    wait := sync.WaitGroup{}
    outerWait := sync.WaitGroup{}

    wait.Add(1)
    outerWait.Add(2)

    go func() {
        wait.Wait()
        fmt.Printf("wait group 1\n")
        outerWait.Done()
    }()

    go func() {
        wait.Wait()
        fmt.Printf("wait group 2\n")
        outerWait.Done()
    }()

    <-time.NewTimer(time.Millisecond * 100).C

    // this should trigger wait to go.. do we get two print statements?
    wait.Done()
    outerWait.Wait()
}

