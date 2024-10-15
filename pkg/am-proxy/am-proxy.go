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
	cConn AMConnection
	gConn AMConnection

	ctx    context.Context
	cancel context.CancelFunc
	idx    int

	cFramer packet.PacketFramer
	gFramer packet.PacketFramer
}

func (a *AMConnectionWrapper) Close() error {
	a.cancel()
	a.cConn.Close()
	if a.gConn != nil {
		a.gConn.Close()
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
		cConn:  conn,
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
		_, err := pkt.Into(w.cConn)
		if err != nil {
			m.logger.Error("could not write error message into connection", "error", err)
		}
	}

    w.Close()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	//m.connections = append(m.connections[:w.idx], m.connections[w.idx+1:]...)
	m.connections[w.idx] = nil
	m.open = append(m.open, w.idx)
}

func (m *AMProxy) handleConnection(w *AMConnectionWrapper) {
	w.cFramer = packet.NewPacketFramer()
    w.gFramer = packet.NewPacketFramer()
    go packet.FrameWithReader(&w.cFramer, w.cConn)

    // TODO(v1) this could hang forever and dumb brown hat hackers could hurt
    // my delicious ports and memory :(
	authPacket, ok := <-w.cFramer.C

	if !ok || authPacket == nil {
		m.removeConnection(w, nil)
		return
	}

	// TODO i probably want to have a "user" object that i can
	// serialize/deserialize
    if err := m.authenticate(authPacket); err != nil {
		m.removeConnection(w, err)
		return
	}

	gameConnStr, err := m.match.matchmake(m.ctx, w.cConn)
	if err != nil {
		m.removeConnection(w, err)
		return
	}

	gameConn, err := m.factory(gameConnStr)
	if err != nil {
		m.removeConnection(w, err)
		return
	}

	w.gConn = gameConn
    go packet.FrameWithReader(&w.gFramer, w.gConn)

	resp := packet.CreateServerAuthResponse(true)
	_, err = resp.Into(w.cConn)
	if err != nil {
		m.removeConnection(w, err)
		return
	}

	go m.handleConnection(w)
}

func (m *AMProxy) handleConnectionLifecycles(w AMConnectionWrapper) {
	// TODO handle the lifecycles
	// turn client and server into framers and handle the closing edge cases
	// and context shutting off
    // already have the framers by this point, so just intercept the packets
    // and check for any control plane packets
	go func() {
		io.Copy(w.gConn, w.cConn)
	}()

	go func() {
		io.Copy(w.cConn, w.gConn)
	}()
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
