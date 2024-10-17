package amproxy

import (
	"context"
	"io"
)

// TODO consider all of these operations with game type
// there will possibly be a day where i have more than one game type
//go:generate mockery --name GameServer
type GameServer interface {
	GetBestServer() (string, error)
	CreateNewServer(ctx context.Context) (string, error)
	WaitForReady(ctx context.Context, id string) error
	GetConnectionString(id string) (string, error)
	//ListServers() []gameserverstats.GameServerConfig
	String() string
}

type AMConnection interface {
	io.Reader
	io.Writer
	io.Closer
	Addr() string
    Id() string
}

type ConnectionFactory func(string) (AMConnection, error)
