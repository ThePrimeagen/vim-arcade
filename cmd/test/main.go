package main

import (
	"log/slog"
	"time"
)

func main() {
    timer := time.NewTimer(time.Second)

    go func() {
        slog.Info("starting timer")
        _, ok := <-timer.C
        slog.Info("stopping timer", "ok", ok)
    }()

    timer.Stop()
    time.Sleep(time.Second * 2)
}

