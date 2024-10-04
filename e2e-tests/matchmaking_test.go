package e2etests

import (
	"context"
	"log/slog"
	"testing"
	"time"

	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)

func TestMatchMakingCreateServer(t *testing.T) {
    logger := prettylog.SetProgramLevelPrettyLogger(prettylog.NewParams(prettylog.CreateLoggerSink()))
    logger = logger.With("area", "TestMatchMakingCreateServer").With("process", "test")
    slog.SetDefault(logger)

    ctx, cancel := context.WithCancel(context.Background())
    path := getDBPath("no_server")
    state := createEnvironment(ctx, path, servermanagement.ServerParams{
        MaxLoad: 0.9,
    })

    logger.Info("Created environment", "state", state.String())
    client := state.Factory.New()
    logger.Info("Created Client", "state", state.String())

    t.Cleanup(func() {cancel()})

    assertClient(&state, client);
    assertConnectionCount(&state, gameserverstats.GameServecConfigConnectionStats{
        Connections: 1,
        ConnectionsAdded: 1,
        ConnectionsRemoved: 0,
    }, time.Second * 5)
}

