package sim

import (
	"context"
	"log/slog"
	"time"

	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
)

func KillContext(cancel context.CancelFunc) {
    go func() {
        time.Sleep(time.Second * 5)
        cancel()
    }()
}

func CreateLogger(name string) *slog.Logger {
    logger := prettylog.SetProgramLevelPrettyLogger(prettylog.NewParams(prettylog.CreateLoggerSink()))
    logger = logger.With("area", name).With("process", "sim")
    slog.SetDefault(logger)

    logger.Error("Test Logger Created")

    return logger
}


