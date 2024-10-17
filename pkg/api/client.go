package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/packet"
	"vim-arcade.theprimeagen.com/pkg/utils"
)

type ClientState int

const (
	CSInitialized ClientState = iota
	CSConnecting
	CSAuthenticating
	CSConnected
	CSDisconnected
)

func ClientStateToString(state ClientState) string {
	switch state {
	case CSInitialized:
		return "initialized"
	case CSConnecting:
		return "connecting"
	case CSConnected:
		return "connected"
	case CSDisconnected:
		return "disconnected"
	}

	assert.Never("unknown client state", "state", state)
	return ""
}

type hostAndPort struct {
	host string
	port uint16
}

type Client struct {
	logger *slog.Logger
	Host   string
	Port   uint16
	conn   net.Conn
	closed bool
	done   chan struct{}
	ready  chan struct{}
	mutex  sync.Mutex
	State  ClientState
	id     [16]byte
}

func (c *Client) String() string {
	return fmt.Sprintf("Host=%s Port=%d", c.Host, c.Port)
}

func getClientLogger(id []byte) *slog.Logger {
	return slog.Default().With("area", "Client").With("id", hex.EncodeToString(id))
}

func NewClientFromConnString(hostAndPort string, id [16]byte) Client {
	parts := strings.SplitN(hostAndPort, ":", 2)
	port, err := strconv.Atoi(parts[1])
	assert.NoError(err, "client was provided a bad string", "hostAndPortString", hostAndPort)
    logger := getClientLogger(id[:])

	return Client{
		State:  CSInitialized,
		Host:   parts[0],
		Port:   uint16(port),
		mutex:  sync.Mutex{},
		logger: logger,
		id: id,
		done:   make(chan struct{}, 1),
		ready:  make(chan struct{}, 1),
		closed: false,
	}
}

func NewClient(host string, port uint16, id [16]byte) Client {
	return Client{
		State:  CSInitialized,
		Host:   host,
		Port:   uint16(port),
		mutex:  sync.Mutex{},
        logger: getClientLogger(id[:]),
		done:   make(chan struct{}, 1),
		ready:  make(chan struct{}, 1),
		id: id,
		closed: false,
	}
}

func (d *Client) Id() string {
    return hex.EncodeToString(d.id[:])
}

func (d *Client) Addr() string {
	return fmt.Sprintf("%s:%d", d.Host, d.Port)
}

func (d *Client) Write(data []byte) error {
	assert.NotNil(d.conn, "expected the connection to be not nil")
	// TODO maybe consider ensure we write all...
	_, err := d.conn.Write(data)
	return err
}

func (d *Client) Connect(ctx context.Context) error {
	d.State = CSConnecting
	d.logger.Info("client connecting to match making")
	connStr := fmt.Sprintf("%s:%d", d.Host, d.Port)
	d.logger.Info("connect to matchmaking", "conn", connStr)
	conn, err := net.Dial("tcp4", connStr)
	assert.NoError(err, "could not connect to server")
	d.logger.Info("connected to the match making server", "conn", connStr)

    // TODO emit event?
	d.State = CSAuthenticating

    pkt := packet.CreateClientAuth(d.id[:])

    // TODO this is where i left off, i hope you haven't forgotten what you
    // are doing after a weekend of kids soccer...
    packet.RequestResponse(&pkt)

    _, err = conn.Write(d.id[:])
    if err != nil {
        return err
    }

	d.ready <- struct{}{}

	ctxReader := utils.NewContextReader(ctx)
	go ctxReader.Read(conn)

	go func() {
		for bytes := range ctxReader.Out {
			d.logger.Error("message received", "data", string(bytes))
		}

		if err, ok := <-ctxReader.Err; ok && !d.closed {
			d.logger.Error("error with client", "error", err)
		}

		d.State = CSDisconnected
		d.done <- struct{}{}
	}()

	return nil
}

func (d *Client) WaitForDone() {
	<-d.done
}

func (d *Client) WaitForReady() {
	<-d.ready
}

func (d *Client) authenticate() error {
    return d.Write(d.id[:])
}

func (d *Client) Disconnect() {
	d.closed = true
	assert.NotNil(d.conn, "attempting to disconnect a non connected client")

	n, err := d.conn.Write(packet.LegacyClientClose())
	if err != nil {
		d.logger.Error("unable to write ClientClose to source", "n", n, "err", err)
	}

	err = d.conn.Close()
	if err != nil {
		d.logger.Error("error on close during disconnect", "err", err)
	}
}
