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
)

type Sin struct {
	Amplitude float64
	Period    float64
	Offset    float64
}

type SimulationParams struct {
	Seed   int64
	Rounds int

	Host string
	Port uint16

	ConnectionAddRem  StatRange
	ConnectionAdds    Sin
	ConnectionRemoves Sin
	RoundsToPeriod    int

	TimeToConnectionCountMS int64
}

func (s *SimulationParams) roundToPeriod(current int) float64 {
	return 2 * math.Pi * (float64(current) / float64(s.RoundsToPeriod))
}

type Simulation struct {
	params       SimulationParams
	state        *ServerState
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

func NewSimulation(params SimulationParams, state *ServerState) Simulation {
    assert.Assert(params.RoundsToPeriod > 0, "rounds to period needs to be 1 or more", "params", params)
	return Simulation{
		logger:       slog.Default().With("area", "Simulation"),
		state:        state,
		params:       params,
		rand:         rand.New(rand.NewSource(params.Seed)),
		mutex:        sync.Mutex{},
		totalAdds:    0,
		totalRemoves: 0,
	}
}

func (s *Simulation) String() string {
	return fmt.Sprintf(`----- Simulation -----
adds: %d (%d)
removes: %d (%d)
round: %d
`, s.adds, s.totalAdds, s.removes, s.totalRemoves, s.currentRound)
}

func (s *Simulation) getBatch(x float64, sin Sin) int {
	mag := sin.Amplitude + sin.Amplitude*math.Sin(x*sin.Period + sin.Offset)
	batch := mag * float64(GetNextBatch(
		s.rand, s.params.ConnectionAddRem,
	))
	return int(batch)
}

func (s *Simulation) RunSimulation(ctx context.Context) error {
	s.Done = false

	factory := NewTestingClientFactory(s.params.Host, s.params.Port, s.logger)
	connections := NewSimulationConnections(factory, s.rand)
	waiter := NewStateWaiter(s.state.Sqlite)
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

		adds := s.getBatch(
			s.params.roundToPeriod(s.currentRound),
			s.params.ConnectionAdds,
		)

		removes := max(0, min(
			connections.Len(),
			s.getBatch(
				s.params.roundToPeriod(s.currentRound),
				s.params.ConnectionRemoves,
			),
		))

		startingConns := waiter.StartRound()
		connections.StartRound(adds, removes)

		expectedDone := startingConns
		expectedDone.Connections += adds - removes
		expectedDone.ConnectionsAdded += adds
		expectedDone.ConnectionsRemoved += removes

		s.logger.Info("SimRound", "round", round, "adds", adds, "removes", removes, "current", startingConns, "expected", expectedDone)

		go connections.AddBatch(adds)
		go connections.Remove(removes)

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
