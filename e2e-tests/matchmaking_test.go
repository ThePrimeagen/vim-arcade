package e2etests

import (
	"context"
	"testing"
	"time"

	"vim-arcade.theprimeagen.com/e2e-tests/sim"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)

func TestMatchMakingCreateServer(t *testing.T) {
    logger := sim.CreateLogger("TestMatchMakingCreateServer")
    ctx, cancel := context.WithCancel(context.Background())
    sim.KillContext(cancel)

    path := sim.GetDBPath("no_server")
    state := sim.CreateEnvironment(ctx, path, servermanagement.ServerParams{
        MaxLoad: 0.9,
    })

    logger.Info("Created environment", "state", state.String())
    client := state.Factory.New()
    logger.Info("Created Client", "state", state.String())

    t.Cleanup(func() {cancel()})

    sim.AssertClient(&state, client);
    sim.AssertConnectionCount(&state, gameserverstats.GameServecConfigConnectionStats{
        Connections: 1,
        ConnectionsAdded: 1,
        ConnectionsRemoved: 0,
    }, time.Second * 5)
}

func TestMakingServerWithBatchRequest(t *testing.T) {
    sim.CreateLogger("TestMakingServerWithBatchRequest")
    ctx, cancel := context.WithCancel(context.Background())
    sim.KillContext(cancel)

    path := sim.GetDBPath("no_server")
    state := sim.CreateEnvironment(ctx, path, servermanagement.ServerParams{
        MaxLoad: 0.9,
    })

    clients := state.Factory.CreateBatchedConnections(15)
    t.Cleanup(func() {cancel()})

    sim.AssertClients(&state, clients);
    sim.AssertAllClientsSameServer(&state, clients);
    sim.AssertConnectionCount(&state, gameserverstats.GameServecConfigConnectionStats{
        Connections: 15,
        ConnectionsAdded: 15,
        ConnectionsRemoved: 0,
    }, time.Second * 5)
}
