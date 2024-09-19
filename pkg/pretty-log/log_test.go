package prettylog_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
)

func waitForParams(params prettylog.PrettyLoggerParams) {
    time.Sleep(params.ThrottleTime + time.Millisecond)
}

func getLogger() (prettylog.PrettyLoggerParams, *slog.Logger, *bytes.Buffer) {
    b := make([]byte, 0, 8192)
    buf := bytes.NewBuffer(b)
    params := prettylog.NewParams(buf)
    params.ThrottleTime = time.Millisecond * 10
	prettyHandler := prettylog.NewHandler(&slog.HandlerOptions{
		Level:       slog.LevelDebug,
		AddSource:   false,
		ReplaceAttr: nil,
	}, params, prettylog.WithoutColor())

	logger := slog.New(prettyHandler).With("process", "t").With("area", "t")
    return params, logger, buf
}

func filterEmpty(args []string) []string {
    out := []string{}
    for _, v := range args {
        if v != "" {
            out = append(out, v)
        }
    }
    return out
}

func TestPrettyLoggerSimpleDedupe(t *testing.T) {
    params, logger, buf := getLogger()

    logger.Info("no dedupe")
    logger.Info("dedupe")
    logger.Info("dedupe")
    logger.Info("dedupe")
    logger.Info("no dedupe")
    waitForParams(params)

    parts := filterEmpty(strings.Split(string(buf.Bytes()), "\n"))
    require.Equal(t, len(parts), 3)
    require.Equal(t, parts[0], "t:t INFO: no dedupe")
    require.Equal(t, parts[1], "t:t 3 INFO: dedupe")
    require.Equal(t, parts[2], "t:t INFO: no dedupe")
}

func TestPrettyLoggerWithArgs(t *testing.T) {
    params, logger, buf := getLogger()

    logger.Info("no dedupe")
    logger.Info("dedupe", "arg", 1)
    logger.Info("dedupe", "arg", 2)
    logger.Info("dedupe", "arg", 2)
    logger.Info("no dedupe")
    waitForParams(params)

    parts := filterEmpty(strings.Split(string(buf.Bytes()), "\n"))
    require.Equal(t, len(parts), 4)
    require.Equal(t, parts[0], "t:t INFO: no dedupe")
    require.Equal(t, parts[1], "t:t INFO: dedupe arg=1")
    require.Equal(t, parts[2], "t:t 2 INFO: dedupe arg=2")
    require.Equal(t, parts[3], "t:t INFO: no dedupe")
}
