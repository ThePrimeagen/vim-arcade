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
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
)

type SimulationParams struct {
	Seed                     int64
	Rounds                   int
	Host                     string
	Port                     uint16
	Stats                    gameserverstats.GSSRetriever
	StdConnections           int
	MaxBatchConnectionChange int
	TimeToConnectionCountMS  int64
	ConnectionSleepMinMS     int
	ConnectionSleepMaxMS     int
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

func getNextBatch(s *Simulation, remaining int) int {
    maxRemaining := min(remaining, s.params.MaxBatchConnectionChange)
    randomRemaining := s.nextInt(1, maxRemaining)
    return randomRemaining
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

	s.logger.Error("starting simulation", "waitTime", waitTime/time.Millisecond)
	// Seed the random number generator for different results each time
outer:
	for round := range s.params.Rounds {
		s.currentRound = round

		select {
		case <-ctx.Done():
			break outer
		default:
		}

		adds := int(math.Abs(s.rand.NormFloat64() * float64(s.params.StdConnections)))
		removes := min(connections.Len(), int(math.Abs(s.rand.NormFloat64()*float64(s.params.StdConnections))))

        startingConns := waiter.StartRound()
		connections.StartRound(adds, removes)

        expectedDone := startingConns
        expectedDone.Connections += adds - removes
        expectedDone.ConnectionsAdded += adds
        expectedDone.ConnectionsRemoved += removes

		s.logger.Info("SimRound", "round", round, "adds", adds, "removes", removes, "current", startingConns, "expected", expectedDone)

		go func() {
			s.adds = adds
			for s.adds > 0 {
				randomAdds := getNextBatch(s, s.adds)
				s.adds -= randomAdds

				assert.Assert(s.adds >= 0, "s.adds somehow become negative")

				<-time.NewTimer(time.Millisecond * time.Duration(s.nextInt(s.params.ConnectionSleepMinMS, s.params.ConnectionSleepMaxMS))).C
				_ = connections.AddBatch(randomAdds)
			}
		}()

		go func() {
			s.removes = removes
			for s.removes > 0 {
				randomRemoves := getNextBatch(s, s.removes)
				s.removes -= randomRemoves

				if connections.Len() == 0 {
					continue
				}

				<-time.NewTimer(time.Millisecond * time.Duration(s.nextInt(s.params.ConnectionSleepMinMS, s.params.ConnectionSleepMaxMS))).C
				connections.Remove(randomRemoves)
			}
		}()

		addedConns, removedConns := connections.FinishRound()
        waiter.WaitForRound(adds, removes, time.Duration(waitTime))

		s.logger.Error("Added and Removed Conns", "expectedAdds", adds, "expectedRemoves", removes, "addedConns", len(addedConns), "removedConns", len(removedConns))
		s.totalAdds += adds
		s.totalRemoves += removes

		timeTaken := waiter.AssertRound(addedConns, removedConns)
		s.logger.Info("SimRound finished", "round", round, "totalAdds", s.totalAdds, "totalRemoves", s.totalRemoves, "time taken ms", timeTaken.Milliseconds())
	}

	s.logger.Warn("Simulation Completed")
	s.Done = true
	return nil
}
