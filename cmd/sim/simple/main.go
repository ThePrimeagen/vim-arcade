package main

import (
	"context"
	"os"
    // i am sure there is a better way to do this
	"path"
	"time"

	"vim-arcade.theprimeagen.com/e2e-tests/sim"
	"vim-arcade.theprimeagen.com/pkg/assert"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)


func main() {
    logger := sim.CreateLogger("simple-sim")

    ctx, cancel := context.WithCancel(context.Background())
    sim.KillContext(cancel)

    cwd, err := os.Getwd()
    assert.NoError(err, "unable to get cwd")
    p := path.Join(cwd, "e2e-tests/data/no_server")

    state := sim.CreateEnvironment(ctx, p, servermanagement.ServerParams{
        MaxLoad: 0.9,
    })

    logger.Info("Created environment", "state", state.String())
    client := state.Factory.New()
    logger.Info("Created Client", "state", state.String())

    defer cancel()
    sim.AssertClient(&state, client);
    sim.AssertConnectionCount(&state, gameserverstats.GameServecConfigConnectionStats{
        Connections: 1,
        ConnectionsAdded: 1,
        ConnectionsRemoved: 0,
    }, time.Second * 5)

    client.Disconnect()

    stats := gameserverstats.GameServerConfig{
        Id: client.ServerId,
        Connections: 0,
        ConnectionsAdded: 1,
        ConnectionsRemoved: 1,
    }

    // ok i want to assert things about match making now...
    // the proxy itself
    sim.AssertServerStats(&state, stats, time.Second * 5)
}

