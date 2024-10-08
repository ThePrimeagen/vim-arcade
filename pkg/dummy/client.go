package dummy

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/utils"
)

type ClientState int

const (
	CSInitialized ClientState = iota
	CSConnecting
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

type DummyClient struct {
	logger         *slog.Logger
	MMHost         string
	MMPort         uint16
	conn           net.Conn
	done           chan struct{}
	ready          chan struct{}
	mutex          sync.Mutex
	GameServerHost string
	GameServerPort uint16
	State          ClientState
	ConnId         int
}

func (c *DummyClient) String() string {
	return fmt.Sprintf("Host=%s Port=%d", c.MMHost, c.MMPort)
}

var clientId = 0

func getDummyClientLogger() *slog.Logger {
	return slog.Default().With("area", "DummyClient")
}

func NewDummyClientFromConnString(hostAndPort string) DummyClient {
	parts := strings.SplitN(hostAndPort, ":", 2)
	port, err := strconv.Atoi(parts[1])
	assert.NoError(err, "dummy client was provided a bad string", "hostAndPortString", hostAndPort)
	logger := getDummyClientLogger()

	clientId++
	return DummyClient{
		State:  CSInitialized,
		MMHost: parts[0],
		MMPort: uint16(port),
		mutex:  sync.Mutex{},
		logger: logger,
		ConnId: clientId,
		done:   make(chan struct{}, 1),
		ready:  make(chan struct{}, 1),
	}
}

func NewDummyClient(host string, port uint16) DummyClient {
	clientId++
	return DummyClient{
		State:  CSInitialized,
		MMHost: host,
		MMPort: uint16(port),
		logger: getDummyClientLogger(),
		done:   make(chan struct{}, 1),
		ready:  make(chan struct{}, 1),
		ConnId: clientId,
	}
}

func (d *DummyClient) HostAndPort() (string, uint16) {
	return d.MMHost, d.MMPort
}

func (d *DummyClient) GameServerAddr() string {
	return fmt.Sprintf("%s:%d", d.GameServerHost, d.GameServerPort)
}

func (d *DummyClient) Write(data []byte) error {
	assert.NotNil(d.conn, "expected the connection to be not nil")
	// TODO maybe consider ensure we write all...
	_, err := d.conn.Write(data)
	return err
}

// TODO probably do something with context, maybe utils is context done
func (d *DummyClient) connectToMatchMaking(_ context.Context) hostAndPort {
	connStr := fmt.Sprintf("%s:%d", d.MMHost, d.MMPort)
	d.logger.Info("connect to matchmaking", "conn", connStr)
	conn, err := net.Dial("tcp4", connStr)
	assert.NoError(err, "could not connect to server")
	d.logger.Info("connected to the match making server", "conn", connStr)

	data := make([]byte, 1000, 1000)
	n, err := conn.Read(data)
	assert.NoError(err, "client could not read from match making server")
	data = data[0:n]

	parts := strings.Split(string(data), ":")
	assert.Assert(len(parts) == 2, "malformed string from server", "fromServer", string(data))

	port, err := strconv.Atoi(parts[1])
	assert.NoError(err, "port was not a number")

	return hostAndPort{
		port: uint16(port),
		host: parts[0],
	}
}

func (d *DummyClient) Connect(ctx context.Context) error {
	d.State = CSConnecting
	d.logger.Info("client connecting to match making")
	hap := d.connectToMatchMaking(ctx)
	d.GameServerHost = hap.host
	d.GameServerPort = hap.port
	d.logger.Info("client connecting to game server", "host", hap.host, "port", hap.port)
	conn, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", hap.host, hap.port))
	assert.NoError(err, "client could not connect to the game server")
	d.State = CSConnected
	d.conn = conn
	d.logger.Info("client connected to game server", "host", d.MMHost, "port", d.MMPort)
	d.ready <- struct{}{}

	ctxReader := utils.NewContextReader(ctx)
	go ctxReader.Read(conn)

	go func() {
		for bytes := range ctxReader.Out {
			d.logger.Error("message received", "data", string(bytes))
		}

		if err, ok := <-ctxReader.Err; ok {
			d.logger.Error("error with client", "error", err)
		}
		d.State = CSDisconnected
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
