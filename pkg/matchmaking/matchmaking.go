package matchmaking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"vim-arcade.theprimeagen.com/pkg/assert"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)

type MatchMakingServer struct {
	Params   MatchMakingServerParams
	logger   *slog.Logger
	listener net.Listener
	ready    bool
}

// TODO consider all of these operations with game type
// there will possibly be a day where i have more than one game type
type GameServer interface {
	GetBestServer() (string, error)
	CreateNewServer(ctx context.Context) (string, error)
	WaitForReady(ctx context.Context, id string) error
	GetConnectionString(id string) (string, error)
    //ListServers() []gameserverstats.GameServerConfig
	String() string
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

	gameId, err := m.Params.GameServer.GetBestServer()
	if errors.Is(err, servermanagement.NoBestServer) {
		// TODO mutex lock
		// - great if i scale vertically the match making server
		// TODO messaging goes way better
		// - if i ever scale horizontally the match making..
		// TODO When there is a server in init mode, do not create a new one
		gameId, err = m.Params.GameServer.CreateNewServer(ctx)
		if err != nil {
			// TODO If there are no more servers available to create (max server count)
			// then the queue needs to begin
			assert.Never("unimplemented")
		}

		m.logger.Info("waiting for server", "id", gameId)
		err = m.Params.GameServer.WaitForReady(ctx, gameId)
		m.logger.Info("server created", "id", gameId)
		assert.NoError(err, "unsure how this happened")
	} else if err != nil {
		m.logger.Error("getting best server error", "error", err)
		return
	}

	gs, err := m.Params.GameServer.GetConnectionString(gameId)
	assert.NoError(err, "game server id somehow wasn't found")
    assert.Assert(gs != "", "game server gameString did not produce a host:port pair", "id", gameId)

	// TODO probably better to just get a full server information
	m.logger.Info("game server selected", "host:port", gs)

	// TODO(v1) develop a protocol
	conn.Write([]byte(gs))
}

func innerListenForConnections(listener net.Listener, ctx context.Context) <-chan net.Conn {
	ch := make(chan net.Conn, 10)
	go func() {
		for {
			c, err := listener.Accept()
            select {
            case <-ctx.Done():
                if nerr, ok := err.(*net.OpError); ok && nerr.Err.Error() != "use of closed network connection" {
                    assert.NoError(err, "matchmaking was unable to accept connection")
                }
            default:
                assert.NoError(err, "matchmaking was unable to accept connection")
            }
			ch <- c
		}
	}()
	return ch
}

func (m *MatchMakingServer) checkForDeadInstances() {
    //m.Params.GameServer.ListServers()
}

func (m *MatchMakingServer) listenForConnections(ctx context.Context, listener net.Listener) {
	conns := innerListenForConnections(listener, ctx)

    // TODO Configurable
    timer := time.NewTicker(time.Second * 3)

    m.logger.Warn("listening for connections")

    outer:
	for {
		select {
        case <-timer.C:
            m.checkForDeadInstances()
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
    if m.listener != nil {
        err := m.listener.Close()
        if err != nil {
            m.logger.Error("closing tcp server: error", "error", err)
        }
    }
}

func NewMatchMakingServer(params MatchMakingServerParams) *MatchMakingServer {
	return &MatchMakingServer{
		Params: params,
		logger: slog.Default().With("area", "MatchMakingServer"),
	}
}

func (m *MatchMakingServer) Run(ctx context.Context) error {
	portStr := fmt.Sprintf(":%d", m.Params.Port)
	m.logger.Info("starting server", "host:port", portStr)
	l, err := net.Listen("tcp4", portStr)
    m.listener = l
	if err != nil {
		return err
	}

    m.ready = true
	m.listenForConnections(ctx, l)
	return nil
}

func (m *MatchMakingServer) WaitForReady(ctx context.Context) {
    outer:
    for {
        select {
        // TODO wtf was i thinking here?
        // PORQUE CHANNEL
        case <-time.NewTimer(time.Millisecond * 50).C:
            if m.ready {
                break outer
            }
        case <-ctx.Done():
            break outer
        }
    }
}

func (m *MatchMakingServer) String() string {
    return fmt.Sprintf(`-------- MatchMaking --------
connected: %v
%s
`, m.listener != nil, m.Params.GameServer.String())
}
