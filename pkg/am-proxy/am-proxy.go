package amproxy

import (
	"log/slog"
	"net"
)

type AMProxyParams struct {
	Port   int
	logger *slog.Logger
}

func (m *AMProxyParams) Close() {
	m.logger.Warn("closing down")
	if m.listener != nil {
		err := m.listener.Close()
		if err != nil {
			m.logger.Error("closing tcp server: error", "error", err)
		}
	}
}

// Note for later..
// ok... this is a bit nutty
// i don't know what to do with this other than i need to create a server
// but i don't want to create two at a time..
//
// i am going to return a bool because i need to know if i should be the one
// that creates the server or not... weird approach.. i know..
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
	gameId, err := m.Params.GameServer.CreateNewServer(ctx)

	// seee i hate this method.. it feels very prone to failure...
	m.lastCreatedGameId = gameId

	if err != nil {
		// TODO If there are no more servers available to create (max server count)
		// then the queue needs to begin
		assert.Never("unimplemented")
	}

	m.logger.Info("waiting for server", "id", gameId)
	err = m.Params.GameServer.WaitForReady(ctx, gameId)
	m.logger.Info("server created", "id", gameId)
	assert.NoError(err, "i need to be able to handle the issue of failing to create server or the server cannot ready")

	m.stopWaiting()
	return gameId
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

	// TODO clean this up and use the packet parser to parse out the packets
	// TODO Also do not die if the client doesn't send correct data, instead
	// kill the client
	b := make([]byte, 100, 100)
	n, err := conn.Read(b)
	assert.NoError(err, "unable to read from connection")

	connId, err := packet.ParseClientId(string(b[0:n]))
	assert.NoError(err, "unable to parse out id of client", "data", string(b[0:n]))

	gameId, err := m.Params.GameServer.GetBestServer()
	m.logger.Info("getting best server", "gameId", gameId, "error", err, "id", connId)
	if errors.Is(err, servermanagement.NoBestServer) {
		gameId = m.createAndWait(ctx)
	} else if err != nil {
		m.logger.Error("getting best server error", "error", err, "id", connId)
		return
	}

	gs, err := m.Params.GameServer.GetConnectionString(gameId)
	assert.NoError(err, "game server id somehow wasn't found", "id", connId)
	assert.Assert(gs != "", "game server gameString did not produce a host:port pair", "id", gameId, "id", connId)

	// TODO probably better to just get a full server information
	m.logger.Info("game server selected", "host:port", gs, "id", connId)

	// TODO(v1) develop a protocol
	conn.Write([]byte(gs))
}

func innerListenForConnections(listener net.Listener, ctx context.Context) <-chan net.Conn {
	ch := make(chan net.Conn, 10)
	go func() {
	outer:
		for {
			c, err := listener.Accept()
			select {
			case <-ctx.Done():
				if !errors.Is(err, net.ErrClosed) {
					assert.NoError(err, "matchmaking was unable to accept connection (with context done)")
				}
				break outer
			default:
				assert.NoError(err, "matchmaking was unable to accept connection")
			}
			ch <- c
		}
	}()
	return ch
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
			m.logger.Info("new mm connection")

			// TODO when there is no more room for servers
			// we need to queue the connection
			m.handleNewConnection(ctx, c)
		}
	}
}

func NewMatchMakingServer(params MatchMakingServerParams) *MatchMakingServer {
	return &MatchMakingServer{
		Params:           params,
		logger:           slog.Default().With("area", "MatchMakingServer"),
		waitingForServer: false,
		ready:            false,
		mutex:            sync.Mutex{},
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
