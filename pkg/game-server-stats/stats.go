package gameserverstats

import (
	"context"
	"fmt"
)

type State int

const (
	GSStateInitializing State = iota
	GSStateReady
	GSStateIdle
	GSStateClosed
)

type GameServecConfigConnectionStats struct {
	Connections        int `db:"connections"`
	ConnectionsAdded   int `db:"connections_added"`
	ConnectionsRemoved int `db:"connections_removed"`
}

func (g *GameServecConfigConnectionStats) String() string {
    return fmt.Sprintf("Conns=%d Added=%d Removed=%d", g.Connections, g.ConnectionsAdded, g.ConnectionsRemoved)
}

func (g *GameServecConfigConnectionStats) Equal(other *GameServecConfigConnectionStats) bool {
    return g.Connections == other.Connections &&
        g.ConnectionsRemoved == other.ConnectionsRemoved &&
        g.ConnectionsAdded == other.ConnectionsAdded
}

type GameServerConfig struct {
	State State `db:"state"`

	Id string `db:"id"`

	Connections        int `db:"connections"`
	ConnectionsAdded   int `db:"connections_added"`
	ConnectionsRemoved int `db:"connections_removed"`

	LastUpdateMS int64 `db:"last_updated"`

	// TODO possible?
	Load float32 `db:"load"`

	Host string `db:"host"`

	Port int `db:"port"`
}

func (g *GameServerConfig) Equal(other *GameServerConfig) bool {
    return g.Id == other.Id &&
        g.Connections == other.Connections &&
        g.ConnectionsAdded == other.ConnectionsAdded &&
        g.ConnectionsRemoved == other.ConnectionsRemoved
}

func stateToString(state State) string {
	switch state {
	case GSStateInitializing:
		return "init"
	case GSStateReady:
		return "ready"
	case GSStateIdle:
		return "idle"
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
