package sim

import (
	"fmt"
	"strings"
	"time"

	assert "vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	"vim-arcade.theprimeagen.com/pkg/matchmaking"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)

type ServerState struct {
	Sqlite      *gameserverstats.Sqlite
	Server      *servermanagement.LocalServers
	MatchMaking *matchmaking.MatchMakingServer
	Port        int
	Factory     *TestingClientFactory
	Conns       ConnMap
}

func (s *ServerState) Close() {
	s.MatchMaking.Close()
	s.Server.Close()

	err := s.Sqlite.Close()
	assert.NoError(err, "sqlite errored on close")
}

func (s *ServerState) String() string {
	configs, err := s.Sqlite.GetAllGameServerConfigs()
	configsStr := strings.Builder{}
	if err != nil {
		_, err = configsStr.WriteString(fmt.Sprintf("unable to get server configs: %s", err))
		assert.NoError(err, "never should happen (famous last words)")
	} else {
		for i, c := range configs {
			if i > 0 {
				configsStr.WriteString("\n")
			}
			configsStr.WriteString(c.String())
		}
	}

	connections := s.Sqlite.GetTotalConnectionCount()
	return fmt.Sprintf(`ServerState:
Connections: %s
Servers
%s
`, connections.String(), configsStr.String())
}

type ServerStateWaiter struct {
    Stats gameserverstats.GSSRetriever
    startConfigs []gameserverstats.GameServerConfig
    conns gameserverstats.GameServecConfigConnectionStats
    startTime time.Time
}

func NewStateWaiter(stats gameserverstats.GSSRetriever) *ServerStateWaiter {
    return &ServerStateWaiter{
        Stats: stats,
        startConfigs: []gameserverstats.GameServerConfig{},
    }
}

func (s *ServerStateWaiter) StartRound() {
	startConfigs, err := s.Stats.GetAllGameServerConfigs()
	assert.NoError(err, "StartRound: unable to get all server configs")
    s.startConfigs = startConfigs
    s.conns = s.Stats.GetTotalConnectionCount()
    s.startTime = time.Now()
}

func (s *ServerStateWaiter) WaitForRound(added, removed int, t time.Duration) {
    expected := s.conns.Connections + added - removed
	start := time.Now()
	for time.Now().Sub(start).Milliseconds() < t.Milliseconds() {
		conns := s.Stats.GetTotalConnectionCount()

		if conns.Connections == expected &&
			conns.ConnectionsAdded == added &&
			conns.ConnectionsRemoved == removed {
			break
		}
		<-time.NewTimer(time.Millisecond * 10).C
	}
}

func (s *ServerStateWaiter) AssertRound(adds, removes []*dummy.DummyClient) time.Duration {
	endConfig, err := s.Stats.GetAllGameServerConfigs()
    assert.NoError(err, "AssertRound: unable to get configs")
    AssertServerState(s.startConfigs, endConfig, adds, removes)

    return time.Now().Sub(s.startTime)
}
