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

func createLogger() *slog.Logger {
    logger := prettylog.SetProgramLevelPrettyLogger(prettylog.NewParams(prettylog.CreateLoggerSink()))
    logger = logger.With("area", "TestMatchMakingCreateServer").With("process", "test")
    slog.SetDefault(logger)
    return logger
}

func TestMatchMakingCreateServer(t *testing.T) {
    logger := createLogger()

    ctx, cancel := context.WithCancel(context.Background())
    path := getDBPath("no_server")
    state := createEnvironment(ctx, path, servermanagement.ServerParams{
        MaxLoad: 0.9,
    })

    logger.Info("Created environment", "state", state.String())
    client := state.Factory.New()
    logger.Info("Created Client", "state", state.String())

    t.Cleanup(func() {cancel()})

    AssertClient(&state, client);
    AssertConnectionCount(&state, gameserverstats.GameServecConfigConnectionStats{
        Connections: 1,
        ConnectionsAdded: 1,
        ConnectionsRemoved: 0,
    }, time.Second * 5)
}

func TestMakingServerWithBatchRequest(t *testing.T) {
    logger := createLogger()

    ctx, cancel := context.WithCancel(context.Background())
    path := getDBPath("no_server")
    state := createEnvironment(ctx, path, servermanagement.ServerParams{
        MaxLoad: 0.9,
    })

    logger.Info("Created environment", "state", state.String())
    clients := CreateBatchedConnections(15, state.Factory, logger)
    logger.Info("Created Client", "state", state.String())

    t.Cleanup(func() {cancel()})

    AssertClients(&state, clients);
    AssertConnectionCount(&state, gameserverstats.GameServecConfigConnectionStats{
        Connections: 15,
        ConnectionsAdded: 15,
        ConnectionsRemoved: 0,
    }, time.Second * 5)
}
