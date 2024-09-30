package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
)

func main() {
    prettylog.SetProgramLevelPrettyLogger(prettylog.NewParams(os.Stderr))
    slog.SetDefault(slog.Default().With("process", "DummyServer"))

    ll :=  slog.Default().With("area", "origin-server-test")
    ll.Warn("origin-server-test initializing...")

    port, err := dummy.GetFreePort()
    assert.NoError(err, "cannot get port")

    hostAndPort := fmt.Sprintf(":%d", port)
    ll.Warn("origin-server-test port", "port", port)

    l, err := net.Listen("tcp4", hostAndPort)
    assert.NoError(err, "cannot listen to port")

    id := 0

    for {
        conn, err := l.Accept()
        assert.NoError(err, "unable to accept any more connections")
        thisId := id
        id++

        bytes := make([]byte, 1024, 1024)
        n, err := conn.Read(bytes)
        assert.NoError(err, "unable to read from connection")

        ll.Warn("connection data", "id", thisId, "data", string(bytes[:n]))
        conn.Write(bytes[:n])
        ll.Warn("closing down connection", "id", thisId)
        conn.Close()
    }
}

