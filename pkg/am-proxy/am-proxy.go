package amproxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/packet"
)

var AMProxyDisallowed = fmt.Errorf("unable to connnect, please try again later")

type AMConnectionWrapper struct {
	conn           AMConnection
	gameServerConn AMConnection

	ctx    context.Context
	cancel context.CancelFunc
	idx    int
}

func (a *AMConnectionWrapper) Close() error {
    a.cancel()
    a.conn.Close()
    if a.gameServerConn != nil {
        a.gameServerConn.Close()
    }
    return nil
}

type AMProxy struct {
	servers GameServer
	match   *MatchMakingServer
    factory ConnectionFactory

	logger      *slog.Logger
	connections []*AMConnectionWrapper
	open        []int
	ctx         context.Context
	mutex       sync.Mutex
	closed      bool
}

func NewAMProxy(ctx context.Context, servers GameServer, factory ConnectionFactory) AMProxy {
	return AMProxy{
		servers: servers,
		match:   NewMatchMakingServer(servers),
        factory: factory,

		logger:      slog.Default().With("area", "AMProxy"),
		connections: []*AMConnectionWrapper{},
		open:        []int{},
		ctx:         ctx,
		mutex:       sync.Mutex{},
		closed:      false,
	}
}

// do my basic connection stuff
func (m *AMProxy) allowedToConnect(AMConnection) error {
	return nil
}

func (m *AMProxy) Add(conn AMConnection) error {
	assert.Assert(m.closed == false, "adding connections when the proxy has been closed")

	if err := m.allowedToConnect(conn); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(m.ctx)
	wrapper := &AMConnectionWrapper{
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	var idx int
	if len(m.open) > 0 {
		idx = m.open[len(m.open)-1]
		m.open = m.open[:len(m.open)-1]
		m.connections[idx] = wrapper
	} else {
		idx = len(m.connections)
		m.connections = append(m.connections, wrapper)
	}
	wrapper.idx = idx

	go m.handleConnection(wrapper)

	return nil
}

func (m *AMProxy) authenticate(*packet.Packet) error {
	return nil
}

func (m *AMProxy) removeConnection(w *AMConnectionWrapper, report error) {

	if report != nil {
		pkt := packet.CreateErrorPacket(report)
		_, err := pkt.Into(w.conn)
		if err != nil {
			m.logger.Error("could not write error message into connection", "error", err)
		}
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.connections = append(m.connections[:w.idx], m.connections[w.idx+1:]...)
}

func (m *AMProxy) handleConnection(w *AMConnectionWrapper) {
	authPacket, err := packet.ReadOne(w.conn)
	if err != nil {
		m.removeConnection(w, err)
		return
	}

	// TODO i probably want to have a "user" object that i can
	// serialize/deserialize
	if err = m.authenticate(authPacket); err != nil {
		m.removeConnection(w, err)
		return
	}

	gameConnStr, err := m.match.matchmake(m.ctx, w.conn)
	if err != nil {
		m.removeConnection(w, err)
		return
	}

    gameConn, err := m.factory(gameConnStr)
	if err != nil {
		m.removeConnection(w, err)
		return
	}

    w.gameServerConn = gameConn

    // TODO i probably need to handle a whole set of conditions here...
    // i mean i don't even forward out errors from one side to the other
    // there is also going to be a close
    // holy cow... so much stuff
    go io.Copy(w.gameServerConn, w.conn)
    go io.Copy(w.conn, w.gameServerConn)
}

func (m *AMProxy) Close() {
	m.logger.Warn("closing down")
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.closed = true
	for _, c := range m.connections {
        c.Close()
	}

	m.connections = []*AMConnectionWrapper{}
}

func (p *AMProxy) Run() {
	p.ctx.Done()
	p.Close()
}
