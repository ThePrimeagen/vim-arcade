package dummy

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"

	"vim-arcade.theprimeagen.com/pkg/assert"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
)

type DummyGameServer struct {
	done     chan struct{}
	db       gameserverstats.GSSRetriever
	stats    gameserverstats.GameServerConfig
	listener net.Listener
}
func readLines(reader *bufio.Reader, out chan<- string) {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			close(out)
			return
		}
		out <- strings.TrimSpace(line)
	}
}

func NewDummyGameServer(db gameserverstats.GSSRetriever, stats gameserverstats.GameServerConfig) *DummyGameServer {
	db.Update(stats)
	return &DummyGameServer{
		stats: stats,
		db:    db,
		done:  make(chan struct{}),
	}
}

func innerListenForConnections(listener net.Listener) <-chan net.Conn {
	ch := make(chan net.Conn, 10)
	go func() {
		for {
			c, err := listener.Accept()
			assert.NoError(err, "tcp listener has failed to accept a connection", "err", err)
			ch <- c
		}
	}()
	return ch
}

func (g *DummyGameServer) handleConnection(ctx context.Context, conn net.Conn) {
    _, err := conn.Write([]byte("ready"))
    if err != nil {
        conn.Close()
        return
    }

	g.stats.Connections++
	g.db.Update(g.stats)

	reader := bufio.NewReader(conn)
	lines := make(chan string)

	go readLines(reader, lines)

	defer func() {
		select {
		case <-ctx.Done():
        case <-lines:
		}
		conn.Close()
	}()
}

func (g *DummyGameServer) Run(ctx context.Context) error {
	portStr := fmt.Sprintf(":%d", g.stats.Port)
	listener, err := net.Listen("tcp4", portStr)
	if err != nil {
		return err
	}
	ch := innerListenForConnections(listener)

    outer:
	for {
		select {
		case <-ctx.Done():
			break outer
		case c := <-ch:
			g.handleConnection(ctx, c)
		}
	}
    g.done <- struct{}{}

	return nil
}

func (g *DummyGameServer) Close() {
	if g.listener != nil {
		g.listener.Close()
	}
}

func (g *DummyGameServer) Wait() {
	<-g.done
}

func (g *DummyGameServer) Loop() error {
	return nil
}
