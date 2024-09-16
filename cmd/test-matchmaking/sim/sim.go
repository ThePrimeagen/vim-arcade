package sim

import (
	"context"
	"log/slog"
	"math"
	"math/rand"

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
}

func NewSimulation(params SimulationParams) Simulation {
    return Simulation{
        logger: slog.Default().With("area", "Simulation"),
        params: params,
        connections: []*dummy.DummyClient{},
        rand: rand.New(rand.NewSource(params.Seed)),
    }
}

func (s *Simulation) client(ctx context.Context) *dummy.DummyClient {
    slog.Info("client connecting...")
    client := dummy.NewDummyClient(s.params.Host, s.params.Port)
    err := client.Connect(ctx)
    assert.NoError(err, "unable to connect to client", "err", err)
    client.WaitForReady()
    slog.Info("client connected")
    s.connections = append(s.connections, &client)
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

        current := s.params.Stats.GetConnectionCount()
        target := int(s.rand.NormFloat64() * float64(s.params.MaxConnections))
        diff := target - current
        diffAbs := int(math.Abs(float64(diff)))
        s.logger.Info("SimRound", "round", round, "current", current, "target", target, "diff", diff, "diffAbs", diffAbs)

        for range diffAbs {
            if diff > 0 {
                s.client(ctx)
            } else if len(s.connections) > 0 {
                idx := s.rand.Int() % len(s.connections)
                slog.Info("SimRound removing connection", "idx", idx)
                s.connections[idx].Disconnect()
                s.connections = append(s.connections[0:idx], s.connections[idx + 1:]...)
            }
        }
    }

    s.Done = true
    return nil
}

