package matchmaking

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"vim-arcade.theprimeagen.com/pkg/assert"
)

type MatchMakingServer struct {
    params MatchMakingServerParams
}

type GameServer interface {
    GetBestServer() (string, error)
}

type MatchMakingServerParams struct {
    Port int
    GameServer GameServer
}

// TODO(v1) create no garbage ([]byte...)
func (m *MatchMakingServer) handleNewConnection(conn net.Conn) {
    // TODO(v1) authenticate
    // TODO(v1) command (connect to game, connect twitch acconut, merge account, etc etc)

    gs, err := m.params.GameServer.GetBestServer()
    defer conn.Close()
    if err != nil {
        slog.Error("getting best server error", "error", err)
        return
    }

    // TODO(v1) develop a protocol
    conn.Write([]byte(gs))
}

func innerListenForConnections(listener net.Listener) <-chan net.Conn {
    ch := make(chan net.Conn, 10)
    go func() {
        for {
            c, err := listener.Accept()
            assert.NoError(err, "tcp listener has failed to accept a connection", "err", err)
            ch <- c
        }
    }()
    return ch
}

func (m *MatchMakingServer) listenForConnections(ctx context.Context, listener net.Listener) {
    conns := innerListenForConnections(listener)
    outer:
    for {
        select {
        case <-ctx.Done():
            break outer
        case c := <-conns:
            m.handleNewConnection(c)
        }
    }
    err := listener.Close()
    if err != nil {
        slog.Error("closing tcp server: error", "error", err)
    }
}

func NewMatchMakingServer(params MatchMakingServerParams) *MatchMakingServer {
    return &MatchMakingServer{
        params: params,
    }
}

func (m *MatchMakingServer) Run(ctx context.Context) error {
    portStr := fmt.Sprintf(":%d", m.params.Port)
    l, err := net.Listen("tcp4", portStr)
    if err != nil {
        return err
    }
    m.listenForConnections(ctx, l)
    return nil
}

