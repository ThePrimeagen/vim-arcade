package sim

import (
	"context"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
)

type SimulationParams struct {
    Seed int64
    Rounds int
    Host string
    Port uint16
    Stats gameserverstats.GSSRetriever
    MaxConnections int
}

type Simulation struct {
    params SimulationParams
    connections []*dummy.DummyClient
    rand *rand.Rand
    Done bool
    logger *slog.Logger
    mutex sync.Mutex
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

func (s *Simulation) client(ctx context.Context) *dummy.DummyClient {
    slog.Info("client connecting...")
    client := dummy.NewDummyClient(s.params.Host, s.params.Port)
    err := client.Connect(ctx)
    assert.NoError(err, "unable to connect to client", "err", err)
    client.WaitForReady()
    slog.Info("client connected")
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
        current := s.params.Stats.GetTotalConnectionCount()
        adds := int(math.Abs(s.rand.NormFloat64() * float64(s.params.MaxConnections)))
        removes := int(math.Abs(s.rand.NormFloat64() * float64(s.params.MaxConnections)))
        s.logger.Info("SimRound", "round", round, "current", current, "adds", adds, "removes", removes)

        wait := sync.WaitGroup{}
        wait.Add(2)
        go func() {
            for range adds {
                <-time.NewTimer(time.Duration(s.nextInt(50, 800))).C
                s.client(ctx);
            }
            wait.Done()
        }()

        go func() {
            for range removes {
                if len(s.connections) == 0 {
                    continue
                }
                <-time.NewTimer(time.Duration(s.nextInt(50, 800))).C
                s.removeRandom()
            }
            wait.Done()
        }()

        wait.Wait();
        s.logger.Info("SimRound finished", "time taken ms", time.Now().Sub(start).Milliseconds())
    }

    s.Done = true
    return nil
}

