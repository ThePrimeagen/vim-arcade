package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
)

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

    ll :=  slog.Default().With("area", "reverse-proxy-test")
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

    for {
        conn, err := l.Accept()
        assert.NoError(err, "unable to accept any more connections")

        newId := id
        id++
        ll.Warn("new connection", "id", newId)

        toConn, err := net.Dial("tcp4", toAddr)
        assert.NoError(err, "unable to connect to origin server")

        // Normally i would have to use a context to control closing..
        go func() {
            group := sync.WaitGroup{}
            group.Add(2)
            go func() {
                io.Copy(conn, toConn)
                group.Done()
            }()
            go func() {
                io.Copy(toConn, conn)
                group.Done()
            }()

            group.Wait()
            ll.Warn("io copying is done", "id", newId)
            conn.Close()
        }()
    }
}


