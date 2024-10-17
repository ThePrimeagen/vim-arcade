package main

import (
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

    ctx := sim.TopLevelContext()
    sim.KillContext(time.Second * 7)

    cwd, err := os.Getwd()
    assert.NoError(err, "unable to get cwd")
    p := path.Join(cwd, "e2e-tests/data/no_server")

    state := sim.CreateEnvironment(ctx, p, servermanagement.ServerParams{
        MaxLoad: 0.9,
    })

    CLIENT_COUNT := 15
    logger.Info("Created environment", "state", state.String())
    clients := state.Factory.CreateBatchedConnections(CLIENT_COUNT)
    logger.Info("Created Client", "state", state.String())

    defer sim.CancelTopLevelContext()
    for _, c := range clients {
        sim.AssertClient(&state, c);
    }

    sim.AssertConnectionCount(&state, gameserverstats.GameServecConfigConnectionStats{
        Connections: 15,
        ConnectionsAdded: 15,
        ConnectionsRemoved: 0,
    }, time.Second * 5)

    for i := range CLIENT_COUNT {
        c := clients[i]
        c.Disconnect()
        stats := gameserverstats.GameServerConfig{
            Id: c.ServerId,
            Connections: 15 - (i + 1),
            ConnectionsAdded: 15,
            ConnectionsRemoved: i + 1,
        }
        sim.AssertServerStats(&state, stats, time.Second * 5)
    }
}

