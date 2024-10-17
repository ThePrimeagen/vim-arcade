package sim

import (
	"context"
	"log/slog"
	"time"

	assert "vim-arcade.theprimeagen.com/pkg/assert"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
)

var created = false
var ctx context.Context
var cancel context.CancelFunc

func CancelTopLevelContext() {
    TopLevelContext()
    cancel()
}

func TopLevelContext() context.Context {
    if created == false {
        ctx, cancel = context.WithCancel(context.Background())
        created = true
    }

    return ctx
}

func KillContext(dur time.Duration) {
    TopLevelContext()
    go func() {
        time.Sleep(dur)
        cancel()
        assert.Never("context should never be killed with KillContext")
    }()
}

func CreateLogger(name string) *slog.Logger {
    logger := prettylog.CreateLoggerFromEnv(nil)
    logger = logger.With("area", name).With("process", "sim")
    slog.SetDefault(logger)

    logger.Error("Test Logger Created")

    return logger
}


