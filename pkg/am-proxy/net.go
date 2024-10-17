package amproxy

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/packet"
)

type AMTCPConnection struct {
	conn net.Conn

	connStr string
}

func CreateTCPConnectionFrom(connString string) (AMConnection, error) {
	conn, err := net.Dial("tcp", connString)
	if err != nil {
		return nil, err
	}

	return &AMTCPConnection{
		conn:    conn,
		connStr: connString,
	}, nil
}

func (a *AMTCPConnection) Read(b []byte) (int, error) {
	return a.conn.Read(b)
}

func (a *AMTCPConnection) Write(b []byte) (int, error) {
	return a.conn.Write(b)
}

func (a *AMTCPConnection) Close() error {
	return a.conn.Close()
}

func (a *AMTCPConnection) String() string {
	return a.connStr
}

func (a *AMTCPConnection) Id() string {
	return "AMTCPConnection DOES NOT HAVE ID YET...."
}

func (a *AMTCPConnection) Addr() string {
	return a.connStr
}

type AMTCPProxy struct {
	port     uint16
	proxy    *AMProxy
	logger   *slog.Logger
	listener net.Listener
	ready    chan struct{}
}

func NewTCPProxy(proxy *AMProxy, port uint16) AMTCPProxy {
	ll := slog.Default().With("area", "AMTCPProxy")
	return AMTCPProxy{
		port:   port,
		logger: ll,
		proxy:  proxy,
		ready:  make(chan struct{}, 1),
	}
}

func NewConnection(conn net.Conn) AMConnection {
	return &AMTCPConnection{
		conn:    conn,
		connStr: conn.LocalAddr().String(),
	}
}

func listen(listener net.Listener, ch chan net.Conn) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		ch <- conn
	}
}

func (a *AMTCPProxy) WaitForReady(ctx context.Context) {
    select {
    case <-a.ready:
        a.logger.Info("ready")
    case <-ctx.Done():
    }
}

func (a *AMTCPProxy) Run(ctx context.Context) {
	// TODO validate that 0.0.0.0 works with docker
	portStr := fmt.Sprintf("0.0.0.0:%d", a.port)

	a.logger.Info("server starting", "host:port", portStr)
	l, err := net.Listen("tcp4", portStr)
	assert.NoError(err, "unable to create proxy connection")
	a.logger.Info("server started", "host:port", portStr)

	a.listener = l

	ch := make(chan net.Conn, 10)
	go listen(l, ch)

	a.logger.Info("about to ready", "host:port", portStr)
	a.ready <- struct{}{}
	a.logger.Info("ready sent", "host:port", portStr)

outer:
	for {
		select {
		case conn := <-ch:
			go func() {
				err := a.proxy.Add(NewConnection(conn))
                if err != nil {
                    pkt := packet.CreateErrorPacket(err)
                    _, err = pkt.Into(conn)
                    if err != nil {
                        a.logger.Error("unable to write error packet into connection", "err", err)
                    }
                }
			}()
		case <-ctx.Done():
			break outer
		}
	}
}

func (a *AMTCPProxy) Close() {
	if a.listener != nil {
		a.listener.Close()
	}

	a.proxy.Close()
}
