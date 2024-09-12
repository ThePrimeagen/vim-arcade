package dummy

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"

	"vim-arcade.theprimeagen.com/pkg/assert"
)

type hostAndPort struct {
	host string
	port uint16
}

type DummyClient struct {
	logger *slog.Logger
	host   string
	port   uint16
	conn   net.Conn
	done   chan struct{}
	ready  chan struct{}
}

func getDummyClientLogger() *slog.Logger {
	return slog.Default().With("area", "DummyClient")
}

func NewDummyClientFromConnString(hostAndPort string) DummyClient {
	parts := strings.SplitN(hostAndPort, ":", 2)
	port, err := strconv.Atoi(parts[1])
	assert.NoError(err, "dummy client was provided a bad string", "hostAndPortString", hostAndPort)
	return DummyClient{
		host:   parts[0],
		port:   uint16(port),
		logger: getDummyClientLogger(),
		done:   make(chan struct{}, 1),
		ready:   make(chan struct{}, 1),
	}
}

func NewDummyClient(host string, port uint16) DummyClient {
	return DummyClient{
		host:   host,
		port:   uint16(port),
		logger: getDummyClientLogger(),
		done:   make(chan struct{}, 1),
		ready:   make(chan struct{}, 1),
	}
}

func (d *DummyClient) Write(data []byte) error {
	assert.NotNil(d.conn, "expected the connection to be not nil")
	// TODO maybe consider ensure we write all...
	_, err := d.conn.Write(data)
	return err
}

func (d *DummyClient) connectToMatchMaking(ctx context.Context) hostAndPort {
	d.logger.Info("connect to matchmaking")
	conn, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", d.host, d.port))
	assert.NoError(err, "could not connect to server", "error", err)

	data := make([]byte, 1000, 1000)
	n, err := conn.Read(data)
	assert.NoError(err, "client could not read from match making server", "err", err)
	data = data[0:n]

	parts := strings.Split(string(data), ":")
	assert.Assert(len(parts) == 2, "malformed string from server", "fromServer", string(data))

	port, err := strconv.Atoi(parts[1])
	assert.NoError(err, "port was not a number", "err", err)

	return hostAndPort{
		port: uint16(port),
		host: parts[0],
	}
}

func (d *DummyClient) Connect(ctx context.Context) error {
	d.logger.Info("client connecting to match making")
	hap := d.connectToMatchMaking(ctx)
	d.logger.Info("client connecting to game server")
	conn, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", hap.host, hap.port))
	assert.NoError(err, "client could not connect to the game server", "err", err)
    d.ready<-struct{}{}

	go func() {
		data := make([]byte, 1000, 1000)
		for {
			// TODO do i need to make this any better for dummy test clients?
			// probably not?
			n, err := conn.Read(data)
			if err != nil {
				d.logger.Error("connection read error", "err", err)
				break
			}

			d.logger.Info("data received", "data", string(data[0:n]))
		}

		d.done <- struct{}{}
	}()

	return nil
}

func (d *DummyClient) WaitForDone() {
	<-d.done
}

func (d *DummyClient) WaitForReady() {
	<-d.ready
}

func (d *DummyClient) Disconnect() {
	if d.conn != nil {
		err := d.conn.Close()
		if err != nil {
			d.logger.Error("error on close during disconnect", "err", err)
		}
	}
}
