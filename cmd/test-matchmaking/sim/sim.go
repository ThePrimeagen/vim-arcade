package sim

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
)

type SimulationParams struct {
    Seed int64
    Rounds int
    Host string
    Port uint16
    Stats gameserverstats.GSSRetriever
    MaxConnections int
    TimeToConnectionCountMS int
}

type Simulation struct {
    params SimulationParams
    connections []*dummy.DummyClient
    rand *rand.Rand
    Done bool
    logger *slog.Logger
    mutex sync.Mutex
    adds int
    removes int
}

func NewSimulation(params SimulationParams) Simulation {
    return Simulation{
        logger: slog.Default().With("area", "Simulation"),
        params: params,
        connections: []*dummy.DummyClient{},
        rand: rand.New(rand.NewSource(params.Seed)),
        mutex: sync.Mutex{},
    }
}

func (s *Simulation) push(client *dummy.DummyClient) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    s.connections = append(s.connections, client)
}

func (s *Simulation) nextInt(min int, max int) int {
    out := s.rand.Int()
    diff := max - min
    if diff == 0 {
        return min
    }

    return min + out % diff
}

func (s *Simulation) removeRandom() {
    s.mutex.Lock()

    defer s.mutex.Unlock()
    idx := s.rand.Int() % len(s.connections)
    slog.Info("SimRound removing connection", "idx", idx)
    s.connections[idx].Disconnect()
    s.connections = append(s.connections[0:idx], s.connections[idx + 1:]...)

    return
}

func (s *Simulation) String() string {
    return fmt.Sprintf(`----- Simulation -----
adds: %d
removes: %d
round: %d
`, s.adds, s.removes, s.params.Rounds)
}

func (s *Simulation) client(ctx context.Context) *dummy.DummyClient {
    s.logger.Log(ctx, prettylog.LevelTrace, "client connecting...")
    client := dummy.NewDummyClient(s.params.Host, s.params.Port)
    err := client.Connect(ctx)
    assert.NoError(err, "unable to connect to client", "err", err)
    client.WaitForReady()
    s.logger.Log(ctx, prettylog.LevelTrace, "client connected")
    s.push(&client)
    return &client
}

func (s *Simulation) RunSimulation(ctx context.Context) error {
    s.Done = false

    s.logger.Error("starting simulation")
    // Seed the random number generator for different results each time
    outer:
    for round := range s.params.Rounds {
        select {
        case <-ctx.Done():
            break outer
        default:
        }

        start := time.Now()
        startConnCount := s.params.Stats.GetTotalConnectionCount()
        adds := int(math.Abs(s.rand.NormFloat64() * float64(s.params.MaxConnections)))
        removes := int(math.Abs(s.rand.NormFloat64() * float64(s.params.MaxConnections)))
        s.logger.Info("SimRound", "round", round, "current", startConnCount, "adds", adds, "removes", removes)

        wait := sync.WaitGroup{}
        wait.Add(2)
        go func() {
            s.adds = adds
            for range adds {
                s.adds -= 1
                <-time.NewTimer(time.Millisecond * time.Duration(s.nextInt(50, 300))).C
                s.client(ctx);
            }
            wait.Done()
        }()

        actualRemoves := 0
        go func() {
            s.removes = removes
            for range removes {
                s.removes -= 1
                if len(s.connections) == 0 {
                    continue
                }
                <-time.NewTimer(time.Millisecond * time.Duration(s.nextInt(50, 300))).C
                s.removeRandom()
                actualRemoves++
            }
            wait.Done()
        }()

        wait.Wait();

        stateMet := false
        start = time.Now()
        expected := startConnCount + adds - actualRemoves
        for !stateMet && time.Now().Sub(start).Milliseconds() < int64(s.params.TimeToConnectionCountMS) {
            conns := s.params.Stats.GetTotalConnectionCount()
            stateMet = conns == expected
            <-time.NewTimer(time.Millisecond * 10).C
        }

        assert.Assert(stateMet, "expected to have connection and could not get there within 1 second", "startOfLoop", startConnCount, "adds", adds, "removes", actualRemoves, "expectedTotal", expected, "total", s.params.Stats.GetTotalConnectionCount())
        s.logger.Info("SimRound finished", "time taken ms", time.Now().Sub(start).Milliseconds())
    }

    s.Done = true
    return nil
}

