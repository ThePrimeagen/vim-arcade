package sim

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	assert "vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
)

func AssertClients(state *ServerState, clients []*dummy.DummyClient) {
    for _, client := range clients {
        AssertClient(state, client)
    }
}

func AssertAllClientsSameServer(state *ServerState, clients []*dummy.DummyClient) {
    slog.Info("AssertAllClientsSameServer", "client", len(clients))
    if len(clients) == 0 {
        return
    }

    ip := clients[0].GameServerAddr()
    for _, c := range clients {
        assert.Assert(c.GameServerAddr() == ip, "client ip isn't the same", "expected", ip, "received", c.GameServerAddr())
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

func AssertServerStateCreation(server *ServerState, configs []ServerCreationConfig) {
}

type ConnectionValidator map[string]int

func sumConfigConns(configs []gameserverstats.GameServerConfig) ConnectionValidator {
	out := make(map[string]int)
	for _, c := range configs {
		out[c.Addr()] = c.Connections
	}
	return out
}

func (c *ConnectionValidator) Add(conns []*dummy.DummyClient) {
	for _, conn := range conns {
		fmt.Fprintf(os.Stderr, "ConnectionValidator#Add: %s\n", conn.GameServerAddr())
		(*c)[conn.GameServerAddr()] += 1
	}
}

func (c *ConnectionValidator) Remove(conns []*dummy.DummyClient) {
	for _, conn := range conns {
		fmt.Fprintf(os.Stderr, "ConnectionValidator#Remove: %s\n", conn.GameServerAddr())
		(*c)[conn.GameServerAddr()] -= 1
	}
}

func (c *ConnectionValidator) String() string {
	out := make([]string, 0, len(*c))
	for k, v := range *c {
		out = append(out, fmt.Sprintf("%s = %d", k, v))
	}
	return strings.Join(out, "\n")
}

func AssertServerState(before []gameserverstats.GameServerConfig, after []gameserverstats.GameServerConfig, adds []*dummy.DummyClient, removes []*dummy.DummyClient) {
    beforeValidator := sumConfigConns(before)
    afterValidator := sumConfigConns(after)

    beforeValidator.Add(adds)
    beforeValidator.Remove(removes)

    beforeKeysIter := maps.Keys(beforeValidator)
    afterKeysIter := maps.Keys(afterValidator)

    beforeKeys := slices.SortedFunc(beforeKeysIter, func(a, b string) int {
        return strings.Compare(a, b)
    })
    afterKeys := slices.SortedFunc(afterKeysIter, func(a, b string) int {
        return strings.Compare(a, b)
    })

    assert.Assert(len(beforeKeys) == len(afterKeys), "before and after keys have different lengths", "before", beforeKeys, "after", afterKeys)
    for i, v := range beforeKeys {
        assert.Assert(afterKeys[i] == v, "before and after key order doesn't match", "i", i, "before", v, "after", afterKeys[i], "beforeKeys", beforeKeys, "afterKeys", afterKeys)
        if beforeValidator[v] != afterValidator[v] {
            slog.Error("--------------- Validation Failed ---------------")

            b := sumConfigConns(before)
            slog.Error("server state before", "before", b.String(), "after", afterValidator.String())
            slog.Error("Adds")
            for i, c := range adds {
                slog.Error("    client", "i", i, "addr", c.GameServerAddr())
            }

            slog.Error("Removes")
            for i, c := range removes {
                slog.Error("    client", "i", i, "addr", c.GameServerAddr())
            }
            assert.Never("expected vs received connection count mismatch", "failedOn", v, "expected", afterValidator, "received", beforeValidator)
        }
    }
}

