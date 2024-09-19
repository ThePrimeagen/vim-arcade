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

type DummyClient struct {
	logger *slog.Logger
	host   string
	port   uint16
	conn   net.Conn
	done   chan struct{}
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
	}
}

func NewDummyClient(host string, port uint16) DummyClient {
	return DummyClient{
		host:   host,
		port:   uint16(port),
		logger: getDummyClientLogger(),
		done:   make(chan struct{}, 1),
	}
}

func (d *DummyClient) Write(data []byte) error {
	assert.NotNil(d.conn, "expected the connection to be not nil")
	// TODO maybe consider ensure we write all...
	_, err := d.conn.Write(data)
	return err
}

func (d *DummyClient) Connect(ctx context.Context) error {
	d.logger.Info("connecting")
	conn, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", d.host, d.port))
	assert.NoError(err, "could not connect to server", "error", err)

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

func (d *DummyClient) Wait() {
	<-d.done
}

func (d *DummyClient) Disconnect() {
	if d.conn != nil {
		err := d.conn.Close()
		if err != nil {
			d.logger.Error("error on close during disconnect", "err", err)
		}
	}
}
