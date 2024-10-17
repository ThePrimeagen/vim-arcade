package dummy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"vim-arcade.theprimeagen.com/pkg/assert"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	"vim-arcade.theprimeagen.com/pkg/packet"
)

var id = 0

func getId() int {
	out := id
	id++
	return out
}

type DummyGameServer struct {
	done     bool
	doneChan     chan struct{}
	db       gameserverstats.GSSRetriever
	stats    gameserverstats.GameServerConfig
	listener net.Listener
	logger   *slog.Logger
	mutex    sync.Mutex
}

// this is bad...
// i just needed something to do reads with context
func (s *DummyGameServer) readLines(reader io.Reader, id int, out chan<- []byte) {
	bytes := make([]byte, 1000, 1000)
	for {
		s.logger.Warn("readLines waiting", "conn-id", id)
		n, err := reader.Read(bytes)
		s.logger.Warn("readLines read", "conn-id", id, "n", n, "err", err)
		if err != nil {
			break
		}

        // TODO this should be better at parsing data
		out <- bytes[0:n]
	}

	out <- []byte{}
	close(out)
}

func NewDummyGameServer(db gameserverstats.GSSRetriever, stats gameserverstats.GameServerConfig) *DummyGameServer {
	logger := slog.Default().With("area", "GameServer")
	logger.Warn("new dummy game server", "ID", os.Getenv("ID"))

	return &DummyGameServer{
		logger: logger,
		stats:  stats,
		db:     db,
        done: false,
		doneChan:   make(chan struct{}, 1),
		mutex:  sync.Mutex{},
	}
}

func (g *DummyGameServer) innerListenForConnections(listener net.Listener) <-chan net.Conn {
	ch := make(chan net.Conn, 10)
	go func() {
		for {
			c, err := listener.Accept()
            if g.done {
                break
            }

			assert.NoError(err, "DummyGameServer was unable to accept connection")
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
	g.stats.Load += float32(amount) * 0.001

	if amount >= 0 {
		g.stats.ConnectionsAdded += amount
        g.logger.Info("incConnections(added)", "stats", g.stats.String())
	} else {
		g.stats.ConnectionsRemoved -= amount
        g.logger.Info("incConnections(removed)", "stats", g.stats.String())
	}

}

var connId = 0

func (g *DummyGameServer) handleConnection(ctx context.Context, conn net.Conn) {
	_, err := conn.Write([]byte("ready"))
	if err != nil {
		conn.Close()
		return
	}

	g.incConnections(1)
	connId++

	datas := make(chan []byte, 10)
	go g.readLines(conn, connId, datas)
	go func() {
        outer:
        for {
            select {
            case <-ctx.Done():
                break outer
            case d := <-datas:
                g.logger.Info("client data received", "data", d)
                if packet.LegacyIsClientClosed(d) || packet.LegacyIsEmpty(d) {
                    g.logger.Info("client sent close command")
                    break outer
                }
            }
        }

		// TODO develop a connection struct that has an id
		g.logger.Warn("closing client")
		g.incConnections(-1)
		conn.Close()
	}()
}

func (g *DummyGameServer) Run(outerCtx context.Context) error {
    ctx, cancel := context.WithCancel(outerCtx)

	g.logger.Warn("dummy-server#Run started...")
	portStr := fmt.Sprintf(":%d", g.stats.Port)
	listener, err := net.Listen("tcp4", portStr)
    assert.NoError(err, "unable to start server")

	defer func() {
        g.done = true
        listener.Close()
		g.doneChan <- struct{}{}
	}()

    go g.handleStatUpdating(ctx)

	g.stats.State = gameserverstats.GSStateReady
	err = g.db.Update(g.stats)
	assert.NoError(err, "unable to save the stats of the dummy game server on connection")

	g.logger.Warn("dummy-server#Run running...")

	if err != nil {
        cancel()
		return err
	}

	ch := g.innerListenForConnections(listener)

outer:
	for {

		// TODO This should be configurable?
		timer := time.NewTimer(time.Second * 30)

		g.logger.Info("waiting for connection or ctx done")
		select {
		case <-timer.C:
			if g.stats.Connections == 0 {
                if g.stats.State == gameserverstats.GSStateReady {
                    g.idle()
                    break
                } else if g.stats.State == gameserverstats.GSStateIdle {
                    g.closeDown()
                    cancel()
                    break
                }
                assert.Never("i should never get to this position", "stats", g.stats)
            }
		case <-ctx.Done():
			break outer
		case c := <-ch:
            assert.Assert(g.stats.State != gameserverstats.GSStateClosed, "somehow got a connection when state became closed", "stats", g.stats)

			g.logger.Info("new dummy-server connection", "host", g.stats.Host, "port", g.stats.Port)
			go g.handleConnection(ctx, c)
            g.ready()
		}

        timer.Stop()
	}

	g.stats.State = gameserverstats.GSStateClosed
	err = g.db.Update(g.stats)
	assert.NoError(err, "unable to save the stats of the dummy game server on close")

    // lint requires me to do this despite it not being correct...
    cancel()
	return nil
}

func (g *DummyGameServer) handleStatUpdating(ctx context.Context) {
    timer := time.NewTicker(time.Millisecond * 200)
    prev := g.stats

    outer:
    for {
        select {
        case <-ctx.Done():
            break outer
        case <-timer.C:
            next := g.stats
            if !next.Equal(&prev) {
                err := g.db.Update(next)
                assert.NoError(err, "failed to update stats", "stats", next)
                prev = next
            }
        }
    }

}

func (g *DummyGameServer) closeDown() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

    g.stats.State = gameserverstats.GSStateClosed
    g.db.Update(g.stats)
    g.logger.Info("setting state to closed", "stats", g.stats)
}

func (g *DummyGameServer) ready() {
    if g.stats.State == gameserverstats.GSStateReady {
        return
    }

	g.mutex.Lock()
	defer g.mutex.Unlock()

    g.stats.State = gameserverstats.GSStateIdle
    g.db.Update(g.stats)
    g.logger.Info("setting state to ready", "stats", g.stats)
}

func (g *DummyGameServer) idle() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

    g.stats.State = gameserverstats.GSStateIdle
    g.db.Update(g.stats)
    g.logger.Info("setting state to idle", "stats", g.stats)
}

func (g *DummyGameServer) Close() {
	if g.listener != nil {
        g.done = true
		g.listener.Close()
	}
}

func (g *DummyGameServer) Wait() {
	<-g.doneChan
}

func (g *DummyGameServer) Loop() error {
	return nil
}
