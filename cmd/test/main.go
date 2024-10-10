package main

import (
	"fmt"
	"net"

	"vim-arcade.theprimeagen.com/pkg/assert"
)

func main() {
    l, err := net.Listen("tcp", "0.0.0.0:42069")
    assert.NoError(err, "oops?")

    for {
        conn, err := l.Accept()
        fmt.Printf("conn received? l %s r %s\n", conn.LocalAddr().String(), conn.RemoteAddr().String())
        assert.NoError(err, "oops again")

        d := make([]byte, 100,  100)
        n, err := conn.Read(d)
        assert.NoError(err, "oops again again")
        fmt.Printf("hello %s\n", string(d[:n]))
        conn.Close()
    }
}

