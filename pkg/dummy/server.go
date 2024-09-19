package dummy

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"

	"vim-arcade.theprimeagen.com/pkg/assert"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
)

var id = 0

func getId() int {
	out := id
	id++
	return out
}

type DummyGameServer struct {
	done     chan struct{}
	db       gameserverstats.GSSRetriever
	stats    gameserverstats.GameServerConfig
	listener net.Listener
	logger   *slog.Logger
    mutex sync.Mutex
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
    logger := slog.Default().With("area", fmt.Sprintf("GameServer-%s", os.Getenv("ID")))
    logger.Warn("new dummy game server", "ID", os.Getenv("ID"))

    err := db.Update(stats)
    assert.NoError(err, "unable to save the stats of the dummy game server", err)
	return &DummyGameServer{
		logger: logger,
		stats:  stats,
		db:     db,
		done:   make(chan struct{}, 1),
        mutex: sync.Mutex{},
	}
}

func innerListenForConnections(listener net.Listener) <-chan net.Conn {
	ch := make(chan net.Conn, 10)
	go func() {
		for {
			c, err := listener.Accept()
			assert.NoError(err, "DummyGameServer was unable to accept connection", "err", err)
			ch <- c
		}
	}()
	return ch
}

// this function is so bad that i need to see a doctor
// which also means i am ready to work at FAANG
func (g *DummyGameServer) incConnections(amount int) {
    g.mutex.Lock()
    defer g.mutex.Unlock()

	g.stats.Connections += amount
	g.stats.Load += float32(amount) * 0.05
    err := g.db.Update(g.stats)
    assert.NoError(err, "failed while writing to the database", "err", err)
}

func (g *DummyGameServer) handleConnection(ctx context.Context, conn net.Conn) {
	_, err := conn.Write([]byte("ready"))
	if err != nil {
		conn.Close()
		return
	}

    g.incConnections(1)

	reader := bufio.NewReader(conn)
	lines := make(chan string, 10)
	go readLines(reader, lines)
	go func() {
        select {
        case <-ctx.Done():
        case <-lines:
        }
        conn.Close()
        g.incConnections(-1)
	}()
}

func (g *DummyGameServer) Run(ctx context.Context) error {
    g.logger.Warn("dummy-server#Run started...")
	portStr := fmt.Sprintf(":%d", g.stats.Port)
	listener, err := net.Listen("tcp4", portStr)
    g.logger.Warn("dummy-server#Run running...")


	if err != nil {
		return err
	}
	ch := innerListenForConnections(listener)

    outer:
	for {
        g.logger.Info("waiting for connection or ctx done")
		select {
		case <-ctx.Done():
			break outer
		case c := <-ch:
            g.logger.Info("new connection")
			go g.handleConnection(ctx, c)
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
