package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"

	"vim-arcade.theprimeagen.com/pkg/assert"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
)

type Proxied struct {
	to     net.Conn
	from   net.Conn
	cancel context.CancelFunc
	ctx    context.Context
}

func ProxiedFromConn(from net.Conn, addr string) *Proxied {
    to, err := net.Dial("tcp4", addr)
    assert.NoError(err, "unable to connect to origin server")

    return &Proxied{
        from: from,
        to: to,
    }
}

func (p *Proxied) Run(outer context.Context) {
    ctx, cancel := context.WithCancel(outer)
    go func() {
        io.Copy(p.from, p.to)
        cancel()
    }()

    go func() {
        io.Copy(p.to, p.from)
        cancel()
    }()

    <-ctx.Done()

    p.from.Close()
    p.to.Close()
}

func main() {
	// No way to get proxy information...
	// hard coded
	var proxyPort uint
	flag.UintVar(&proxyPort, "port", 0, "the port of the service to proxy too")
	flag.Parse()

	assert.Assert(proxyPort != 0, "please provide --port")
	toPort := uint16(proxyPort)

	prettylog.SetProgramLevelPrettyLogger(prettylog.NewParams(os.Stderr))
	slog.SetDefault(slog.Default().With("process", "DummyServer"))

	ll := slog.Default().With("area", "reverse-proxy-test")
	ll.Warn("reverse-proxy-test initializing...")

	myPort, err := dummy.GetFreePort()
	assert.NoError(err, "cannot get port")

	hostAndPort := fmt.Sprintf(":%d", myPort)
	ll.Warn("reverse-proxy-test port", "port", myPort)

	l, err := net.Listen("tcp4", hostAndPort)
	assert.NoError(err, "cannot listen to port")

	toAddr := fmt.Sprintf(":%d", toPort)
	ll.Warn("to server", "addr", toAddr)

	id := 0

    ctx := context.Background()
	for {
		conn, err := l.Accept()
		assert.NoError(err, "unable to accept any more connections")

		newId := id
		id++
		ll.Warn("new connection", "id", newId)
        proxy := ProxiedFromConn(conn, toAddr)

        go proxy.Run(ctx)
	}
}
