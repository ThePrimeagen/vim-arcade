package amproxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"vim-arcade.theprimeagen.com/pkg/assert"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)

type MatchMakingServer struct {
	servers  GameServer
	logger   *slog.Logger
	listener net.Listener
	ready    bool

	waitingForServer bool

	mutex             sync.Mutex
	wait              sync.WaitGroup
	lastCreatedGameId string
}

func (m *MatchMakingServer) startWaiting() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.waitingForServer {
		m.waitingForServer = true
		m.wait = sync.WaitGroup{}
		m.wait.Add(1)
		return true
	}

	return false
}

func (m *MatchMakingServer) stopWaiting() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.waitingForServer {
		return
	}

	m.wait.Done()
	m.waitingForServer = false
}

func (m *MatchMakingServer) createAndWait(ctx context.Context) string {
	m.logger.Info("going to create and wait for new game server")
	if !m.startWaiting() {
		m.logger.Info("already waiting on server")
		m.wait.Wait()
		m.logger.Info("waited for server to be created", "id", m.lastCreatedGameId)
		return m.lastCreatedGameId
	}

	// TODO messaging goes way better...
	// TODO horizontal scaling can be quite difficult for the current method
	gameId, err := m.servers.CreateNewServer(ctx)

	// seee i hate this method.. it feels very prone to failure...
	m.lastCreatedGameId = gameId

	if err != nil {
		// TODO If there are no more servers available to create (max server count)
		// then the queue needs to begin
		assert.Never("unimplemented")

        // i am thinking that i am going to have to share the error across the
        // connections
	}

	m.logger.Info("waiting for server", "id", gameId)
	err = m.servers.WaitForReady(ctx, gameId)
	m.logger.Info("server created", "id", gameId)
	assert.NoError(err, "i need to be able to handle the issue of failing to create server or the server cannot ready")

	m.stopWaiting()
	return gameId
}

// TODO(v1) create no garbage ([]byte...)
func (m *MatchMakingServer) matchmake(ctx context.Context, conn AMConnection) (string, error) {
    connId := conn.Id()

	gameId, err := m.servers.GetBestServer()
	m.logger.Info("getting best server", "gameId", gameId, "error", err, "id", connId)
	if errors.Is(err, servermanagement.NoBestServer) {
		gameId = m.createAndWait(ctx)
	} else if err != nil {
		m.logger.Error("getting best server error", "error", err, "id", connId)
		return "", err
	}

	gs, err := m.servers.GetConnectionString(gameId)
	assert.NoError(err, "game server id somehow wasn't found", "id", connId)
	assert.Assert(gs != "", "game server gameString did not produce a host:port pair", "id", gameId, "id", connId)

	// TODO probably better to just get a full server information
	m.logger.Info("game server selected", "host:port", gs, "id", connId)

    return gs, nil
}

func (m *MatchMakingServer) Close() {
	m.logger.Warn("closing down")
    //... hmm
}

func NewMatchMakingServer(servers GameServer) *MatchMakingServer {
	return &MatchMakingServer{
        servers: servers,
		logger:           slog.Default().With("area", "MatchMakingServer"),
		waitingForServer: false,
		ready:            false,
		mutex:            sync.Mutex{},
	}
}

func (m *MatchMakingServer) String() string {
	return fmt.Sprintf(`-------- MatchMaking --------
connected: %v
%s
`, m.listener != nil, m.servers.String())
}
