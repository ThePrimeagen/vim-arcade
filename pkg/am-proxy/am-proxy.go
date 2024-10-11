package amproxy

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/packet"
)

var AMProxyDisallowed = fmt.Errorf("unable to connnect, please try again later")

type AMConnectionWrapper struct {
	conn   AMConnection
	ctx    context.Context
	cancel context.CancelFunc
	idx    int
}

type AMProxy struct {
	servers GameServer

	logger      *slog.Logger
	connections []*AMConnectionWrapper
	open        []int
	ctx         context.Context
	mutex       sync.Mutex
	closed    bool
}

func NewAMProxy(ctx context.Context, servers GameServer) AMProxy {
	return AMProxy{
		servers: servers,

		logger:      slog.Default().With("area", "AMProxy"),
		connections: []*AMConnectionWrapper{},
		open:        []int{},
		ctx:         ctx,
		mutex:       sync.Mutex{},
        closed:  false,
	}
}

// do my basic connection stuff
func (m *AMProxy) allowedToConnect(conn AMConnection) error {
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

func (m *AMProxy) authenticate(pkt *packet.Packet) error {

	return nil
}

func (m *AMProxy) removeConnection(w *AMConnectionWrapper, report error) {

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.connections = append(m.connections[:w.idx], m.connections[w.idx+1:]...)
}

func (m *AMProxy) handleConnection(w *AMConnectionWrapper) {
	authPacket, err := packet.ReadOne(w.conn)
	if err != nil {
		m.removeConnection(w, err)
	}

	// TODO i probably want to have a "user" object that i can
	// serialize/deserialize
	if err = m.authenticate(authPacket); err != nil {
		m.removeConnection(w, err)
	}

	// START_HERE: bring over the server finding logic
	// this is the reverse proxy we have been talking about!
}

func (m *AMProxy) Close() {
	m.logger.Warn("closing down")
	m.mutex.Lock()
	defer m.mutex.Unlock()

    m.closed = true
    for _, c := range m.connections {
        c.cancel()
        c.conn.Close()
    }

    m.connections = []*AMConnectionWrapper{}
}

func (p *AMProxy) Run() {
    p.ctx.Done()
    p.Close()
}
