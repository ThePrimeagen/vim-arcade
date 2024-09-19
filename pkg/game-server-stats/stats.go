package gameserverstats

import (
	"context"
	"fmt"
)

type State int

const (
	GSStateInitializing State = iota
	GSStateReady
	GSStateClosed
)

type GameServecConfigConnectionStats struct {
	Connections        int `db:"connections"`
	ConnectionsAdded   int `db:"connections_added"`
	ConnectionsRemoved int `db:"connections_removed"`
}

type GameServerConfig struct {
	State State `db:"state"`

	Id string `db:"id"`

	Connections        int `db:"connections"`
	ConnectionsAdded   int `db:"connections_added"`
	ConnectionsRemoved int `db:"connections_removed"`

	// TODO possible?
	Load float32 `db:"load"`

	Host string `db:"host"`

	Port int `db:"port"`
}

func stateToString(state State) string {
	switch state {
	case GSStateInitializing:
		return "init"
	case GSStateReady:
		return "ready"
	case GSStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

func (g *GameServerConfig) String() string {
	return fmt.Sprintf("Server(%s): Addr=%s Conns=%d Load=%f State=%s", g.Id, g.Addr(), g.Connections, g.Load, stateToString(g.State))
}

func (g *GameServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", g.Host, g.Port)
}

// TODO I don't know what to call this thing...
type GSSRetriever interface {
	GetById(string) *GameServerConfig
	GetAllGameServerConfigs() ([]GameServerConfig, error)
	Run(ctx context.Context)
	GetServersByUtilization(maxLoad float64) []GameServerConfig
	Update(stats GameServerConfig) error
	GetServerCount() int
    GetTotalConnectionCount() GameServecConfigConnectionStats
}
