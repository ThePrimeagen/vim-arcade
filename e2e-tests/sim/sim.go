package sim

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"

	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
)

type SimulationParams struct {
	Seed                    int64
	Rounds                  int
	Host                    string
	Port                    uint16
	Stats                   gameserverstats.GSSRetriever
	StdConnections          int
	TimeToConnectionCountMS int64
	ConnectionSleepMinMS    int
	ConnectionSleepMaxMS    int
}

type Simulation struct {
	params       SimulationParams
	rand         *rand.Rand
	Done         bool
	logger       *slog.Logger
	mutex        sync.Mutex
	adds         int
	removes      int
	totalAdds    int
	totalRemoves int
	currentRound int
}

func NewSimulation(params SimulationParams) Simulation {
	return Simulation{
		logger:       slog.Default().With("area", "Simulation"),
		params:       params,
		rand:         rand.New(rand.NewSource(params.Seed)),
		mutex:        sync.Mutex{},
		totalAdds:    0,
		totalRemoves: 0,
	}
}

func (s *Simulation) nextInt(min int, max int) int {
	out := s.rand.Int()
	diff := max - min
	if diff == 0 {
		return min
	}

	return min + out%diff
}

func (s *Simulation) String() string {
	return fmt.Sprintf(`----- Simulation -----
adds: %d (%d)
removes: %d (%d)
round: %d
`, s.adds, s.totalAdds, s.removes, s.totalRemoves, s.currentRound)
}

func (s *Simulation) RunSimulation(ctx context.Context) error {
	s.Done = false

    factory := NewTestingClientFactory(s.params.Host, s.params.Port, s.logger)
    connections := NewSimulationConnections(factory, s.rand)
    waiter := NewStateWaiter(s.params.Stats)
    waitTime := time.Millisecond * time.Duration(s.params.TimeToConnectionCountMS)

	s.logger.Error("starting simulation")
	// Seed the random number generator for different results each time
outer:
	for round := range s.params.Rounds {
		s.currentRound = round

		select {
		case <-ctx.Done():
			break outer
		default:
		}

        waiter.StartRound()
        connections.StartRound()

		adds := int(math.Abs(s.rand.NormFloat64() * float64(s.params.StdConnections)))
		removes := int(math.Abs(s.rand.NormFloat64() * float64(s.params.StdConnections)))
		s.logger.Info("SimRound", "round", round, "current", waiter.conns, "adds", adds, "removes", removes)

		go func() {
			for range adds {
				s.adds -= 1
				<-time.NewTimer(time.Millisecond * time.Duration(s.nextInt(s.params.ConnectionSleepMinMS, s.params.ConnectionSleepMaxMS))).C
                _ = connections.Add()
			}
		}()

		actualRemoves := 0
		go func() {
			s.removes = removes
			for range removes {
				s.removes -= 1
				if connections.Len() == 0 {
					continue
				}
				<-time.NewTimer(time.Millisecond * time.Duration(s.nextInt(s.params.ConnectionSleepMinMS, s.params.ConnectionSleepMaxMS))).C
                connections.Remove(1)
				actualRemoves++
			}
		}()

        addedConns, removedConns := connections.FinishRound()

		s.totalAdds += adds
		s.totalRemoves += actualRemoves

        waiter.WaitForRound(adds, actualRemoves, time.Duration(waitTime))
        timeTaken := waiter.AssertRound(addedConns, removedConns)

		s.logger.Info("SimRound finished", "round", round, "totalAdds", s.totalAdds, "totalRemoves", s.totalRemoves, "time taken ms", timeTaken.Milliseconds())
	}

	s.logger.Warn("Simulation Completed")
	s.Done = true
	return nil
}
