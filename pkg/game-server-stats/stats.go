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

type GameServerConfig struct {
    State State

	Id string

	Connections int

	// TODO possible?
	Load float32

    Host string

    Port int
}

func stateToString(state State) string {
    switch (state) {
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
    return fmt.Sprintf("Server(%s): Conns=%d Load=%f State=%s", g.Id, g.Connections, g.Load, stateToString(g.State))
}

func (g *GameServerConfig) Addr() string {
    return fmt.Sprintf("%s:%d", g.Host, g.Port)
}

// TODO I don't know what to call this thing...
type GSSRetriever interface {
    GetById(string) *GameServerConfig
    Iter() func(yield func(i int, s GameServerConfig) bool)
    Run(ctx context.Context)
    Update(stats GameServerConfig) error
    GetServerCount() int
    GetConnectionCount() int
}
