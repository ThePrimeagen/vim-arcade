package e2etests

import (
	"log/slog"
	"time"

	"vim-arcade.theprimeagen.com/pkg/dummy"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	assert "vim-arcade.theprimeagen.com/pkg/assert"
)

func AssertClients(state *ServerState, clients []*dummy.DummyClient) {
    for _, client := range clients {
        AssertClient(state, client)
    }
}

func AssertClient(state *ServerState, client *dummy.DummyClient) {
    slog.Info("assertClient", "client", client.String())
    config, err := state.Sqlite.GetConfigByHostAndPort(client.GameServerHost, client.GameServerPort)
    assert.NoError(err, "unable to get config by host and port", "client", client)
    assert.NotNil(config, "expected a config to be present", "client", client)
}

func AssertConnectionCount(state *ServerState, counts gameserverstats.GameServecConfigConnectionStats, dur time.Duration) {
    slog.Info("assertConnectionCount", "count", counts.String())

    start := time.Now()
    for time.Now().Sub(start) < dur {
        conns := state.Sqlite.GetTotalConnectionCount()
        if conns.Equal(&counts) {
            break
        }
    }

    conns := state.Sqlite.GetTotalConnectionCount()
    assert.Assert(conns.Connections == counts.Connections, "expceted the same number of connections")
    assert.Assert(conns.ConnectionsAdded == counts.ConnectionsAdded, "expceted the same number of connections added")
    assert.Assert(conns.ConnectionsRemoved == counts.ConnectionsRemoved, "expceted the same number of connections removed")
}

