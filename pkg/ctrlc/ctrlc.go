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
    logger := slog.Default().With("area", "ctclc")
    go func() {
        <-c
        logger.Info("ctrl-c", "area", "ctrlc")
        cancel()
        time.Sleep(time.Millisecond * 250)
        // Run Cleanup
        os.Exit(1)
    }()
}


