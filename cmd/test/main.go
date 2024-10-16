package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

var id = 0
func getNextId() int {
    id++
    return id
}

type Suicedek struct {
    Id int
    Ctx context.Context
    Cancel context.CancelFunc
}

func NewSuicedek(ctx context.Context, cancel context.CancelFunc) *Suicedek {
    return &Suicedek{Id: getNextId(), Ctx: ctx, Cancel: cancel}
}

func (s *Suicedek) Run(wait *sync.WaitGroup) {
    <-s.Ctx.Done()
    fmt.Printf("S is done %d\n", s.Id)

    wait.Done()
}

func main() {
    parent, cancel := context.WithCancel(context.Background())

    s := []*Suicedek{}
    for range 5 {
        c, can := context.WithCancel(parent)
        s = append(s, NewSuicedek(c, can))
    }

    wait := sync.WaitGroup{}
    parentSu := s[0]
    for range 5 {
        c, can := context.WithCancel(parentSu.Ctx)
        s = append(s, NewSuicedek(c, can))
    }

    wait.Add(len(s))
    for _, su := range s {
        go su.Run(&wait)
    }

    s[0].Cancel()

    time.Sleep(time.Second * 10)

    cancel()

    wait.Wait()
}

