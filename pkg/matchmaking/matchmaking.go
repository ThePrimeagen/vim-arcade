package matchmaking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"vim-arcade.theprimeagen.com/pkg/assert"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)

type MatchMakingServer struct {
	params   MatchMakingServerParams
	logger   *slog.Logger
	listener net.Listener
}

type GameServer interface {
	GetBestServer() (string, error)
	CreateNewServer(ctx context.Context) (string, error)
	WaitForReady(ctx context.Context, id string) error
	GetConnectionString(id string) (string, error)
}

type MatchMakingServerParams struct {
	Port       int
	GameServer GameServer
}

// TODO(v1) create no garbage ([]byte...)
func (m *MatchMakingServer) handleNewConnection(ctx context.Context, conn net.Conn) {
	// TODO(v1) authenticate
	// TODO(v1) command (connect to game, connect twitch acconut, merge account, etc etc)

    defer func() {
        m.logger.Warn("client connection closed")
        err := conn.Close()
        if err != nil {
            m.logger.Error("unable to close client connection", "error", err)
        }
    }()

    gameString, err := m.params.GameServer.GetBestServer()
    if errors.Is(err, servermanagement.NoBestServer) {
        // TODO mutex lock
        // - great if i scale vertically the match making server
        // TODO messaging goes way better
        // - if i ever scale horizontally the match making..
        // TODO When there is a server in init mode, do not create a new one
        gameString, err = m.params.GameServer.CreateNewServer(ctx)
        if err != nil {
            // TODO If there are no more servers available to create (max server count)
            // then the queue needs to begin
            assert.Never("unimplemented")
        }

        err = m.params.GameServer.WaitForReady(ctx, gameString)
        assert.NoError(err, "unsure how this happened", "err", err)
    } else if err != nil {
        m.logger.Error("getting best server error", "error", err)
        return
    }

    gs, err := m.params.GameServer.GetConnectionString(gameString)
    assert.NoError(err, "game server id somehow wasn't found", "err", err)

    // TODO probably better to just get a full server information
    m.logger.Info("game server selected", "host:port", gs)

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
			m.logger.Warn("context done")
			break outer
		case c := <-conns:
			m.logger.Info("new connection")

            // TODO when there is no more room for servers
            // we need to queue the connection
			m.handleNewConnection(ctx, c)
		}
	}
}

func (m *MatchMakingServer) Close() {
	m.logger.Warn("closing down")
	err := m.listener.Close()
	if err != nil {
		m.logger.Error("closing tcp server: error", "error", err)
	}
}

func NewMatchMakingServer(params MatchMakingServerParams) *MatchMakingServer {
	return &MatchMakingServer{
		params: params,
		logger: slog.Default().With("area", "MatchMakingServer"),
	}
}

func (m *MatchMakingServer) Run(ctx context.Context) error {
	portStr := fmt.Sprintf(":%d", m.params.Port)
	m.logger.Info("starting server", "host:port", portStr)
	l, err := net.Listen("tcp4", portStr)
	if err != nil {
		return err
	}
	m.listenForConnections(ctx, l)
	return nil
}
