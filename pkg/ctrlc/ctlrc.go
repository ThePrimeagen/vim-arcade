package ctrlc

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"
)


func HandleCtrlC(cancel context.CancelFunc) {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)
    go func() {
        <-c
        cancel()
        slog.Info("ctrl-c", "area", "ctrlc")
        time.Sleep(time.Millisecond * 250)
        // Run Cleanup
        os.Exit(1)
    }()
}


