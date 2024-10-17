package sim

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	amproxy "vim-arcade.theprimeagen.com/pkg/am-proxy"
	assert "vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)

type ServerState struct {
	Sqlite  *gameserverstats.Sqlite
	Server  *servermanagement.LocalServers
	Proxy   *amproxy.AMTCPProxy
	Port    int
	Factory *TestingClientFactory
	Conns   ConnMap
}

func (s *ServerState) Close() {
	s.Proxy.Close()
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
	Stats        gameserverstats.GSSRetriever
	startConfigs []gameserverstats.GameServerConfig
	conns        gameserverstats.GameServecConfigConnectionStats
	startTime    time.Time
	logger       *slog.Logger
}

func NewStateWaiter(stats gameserverstats.GSSRetriever) *ServerStateWaiter {
	return &ServerStateWaiter{
		Stats:        stats,
		startConfigs: []gameserverstats.GameServerConfig{},
		logger:       slog.Default().With("area", "StateWaiter"),
	}
}

func (s *ServerStateWaiter) StartRound() gameserverstats.GameServecConfigConnectionStats {
	startConfigs, err := s.Stats.GetAllGameServerConfigs()
	assert.NoError(err, "StartRound: unable to get all server configs")
	s.startConfigs = startConfigs
	s.conns = s.Stats.GetTotalConnectionCount()
	s.startTime = time.Now()

	return s.conns
}

func (s *ServerStateWaiter) WaitForRound(added, removed int, t time.Duration) {
	s.conns.Connections += added - removed
	s.conns.ConnectionsRemoved += removed
	s.conns.ConnectionsAdded += added

	start := time.Now()
	for time.Now().Sub(start).Milliseconds() < t.Milliseconds() {
		conns := s.Stats.GetTotalConnectionCount()

		if conns.Equal(&s.conns) {
			break
		}
		<-time.NewTimer(time.Millisecond * 250).C
	}
}

func (s *ServerStateWaiter) AssertRound(adds, removes []*dummy.DummyClient) time.Duration {
	endConfig, err := s.Stats.GetAllGameServerConfigs()
	assert.NoError(err, "AssertRound: unable to get configs")
	AssertServerState(s.startConfigs, endConfig, adds, removes)

	return time.Now().Sub(s.startTime)
}
