package amproxy

import (
	"context"
	"fmt"
	"log/slog"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/packet"
)

var AMProxyDisallowed = fmt.Errorf("unable to connnect, please try again later")

type AMConnectionWrapper struct {
	cConn AMConnection
	gConn AMConnection

	ctx    context.Context
	cancel context.CancelFunc

	cFramer packet.PacketFramer
	gFramer packet.PacketFramer

	// hell yeah brother
	gsId string
}

func (a *AMConnectionWrapper) Close() error {
	a.cancel()
	a.cConn.Close()
	if a.gConn != nil {
		a.gConn.Close()
	}
	return nil
}

type AMProxyStats struct {
	ActiveConnections int
	TotalConnections  int
	Errors            int
}

type AMProxy struct {
	servers GameServer
	match   *MatchMakingServer
	factory ConnectionFactory

	logger *slog.Logger
	ctx    context.Context
	cancel context.CancelFunc
	closed bool
	stats  AMProxyStats
}

func NewAMProxy(outer context.Context, servers GameServer, factory ConnectionFactory) AMProxy {
	ctx, cancel := context.WithCancel(outer)
	return AMProxy{
		servers: servers,
		match:   NewMatchMakingServer(servers),
		factory: factory,

		logger: slog.Default().With("area", "AMProxy"),
		ctx:    ctx,
		cancel: cancel,
		closed: false,
		stats:  AMProxyStats{},
	}
}

func (m *AMProxy) allowedToConnect(AMConnection) error {
	return nil
}

func (m *AMProxy) Add(conn AMConnection) error {
	assert.Assert(m.closed == false, "adding connections when the proxy has been closed")

	m.stats.ActiveConnections += 1

	if err := m.allowedToConnect(conn); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(m.ctx)
	wrapper := &AMConnectionWrapper{
		cConn:  conn,
		ctx:    ctx,
		cancel: cancel,
	}

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

	// there is only one place to execute this...
	gameConnInfo, err := m.match.matchmake(m.ctx, w.cConn)
	if err != nil {
		m.removeConnection(w, err)
		return
	}

	gameConn, err := m.factory(gameConnInfo.Addr)
	if err != nil {
		m.removeConnection(w, err)
		return
	}

	w.gConn = gameConn
	go packet.FrameWithReader(&w.gFramer, w.gConn)

	// wait.. what is the id???
	resp := packet.CreateServerAuthResponse(true, gameConnInfo.Id)
	_, err = resp.Into(w.cConn)
	if err != nil {
		m.removeConnection(w, err)
		return
	}

	go m.handleConnectionLifecycles(w)
}

func (m *AMProxy) handleConnectionLifecycles(w *AMConnectionWrapper) {
	for {
		select {
		case pkt := <-w.gFramer.C:
			switch pkt.Type() {
			case packet.PacketCloseConnection:
				_, err := pkt.Into(w.cConn)
				m.removeConnection(w, err)
			default:
				_, err := pkt.Into(w.cConn)
				if err != nil {
					m.removeConnection(w, err)
				}
			}
		case pkt := <-w.cFramer.C:
			switch pkt.Type() {
			case packet.PacketCloseConnection:
				_, err := pkt.Into(w.gConn)
				m.removeConnection(w, err)
			default:
				_, err := pkt.Into(w.gConn)
				if err != nil {
					m.removeConnection(w, err)
				}
			}
		case <-w.ctx.Done():
			m.logger.Info("connection finished", "server-id", w.gsId)
            return
		}
	}
}

func (m *AMProxy) Close() {
	m.logger.Warn("closing down")
	m.closed = true
	m.cancel()
}

func (p *AMProxy) Run() {
	<-p.ctx.Done()
	p.Close()
}
