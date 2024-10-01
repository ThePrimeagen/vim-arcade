package sim

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"math"
	"math/rand"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
)

type SimulationParams struct {
	Seed                    int64
	Rounds                  int
	Host                    string
	Port                    uint16
	Stats                   gameserverstats.GSSRetriever
	StdConnections          int
	TimeToConnectionCountMS int
	ConnectionSleepMinMS    int
	ConnectionSleepMaxMS    int
}

type Simulation struct {
	params       SimulationParams
	connections  []*dummy.DummyClient
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

type ConnectionValidator map[string]int

func sumConfigConns(configs []gameserverstats.GameServerConfig) ConnectionValidator {
    out := make(map[string]int)
    for _, c := range configs {
        out[c.Addr()] = c.Connections
    }
    return out
}

func (c *ConnectionValidator) Add(conns []*dummy.DummyClient) {
    for _, conn := range conns {
        fmt.Fprintf(os.Stderr, "ConnectionValidator#Add: %s\n", conn.GameServerAddr())
        (*c)[conn.GameServerAddr()] += 1
    }
}

func (c *ConnectionValidator) Remove(conns []*dummy.DummyClient) {
    for _, conn := range conns {
        fmt.Fprintf(os.Stderr, "ConnectionValidator#Remove: %s\n", conn.GameServerAddr())
        (*c)[conn.GameServerAddr()] -= 1
    }
}

func (c *ConnectionValidator) String() string {
    out := make([]string, 0, len(*c))
    for k, v := range *c {
        out = append(out, fmt.Sprintf("%s = %d", k, v))
    }
    return strings.Join(out, "\n")
}

func NewSimulation(params SimulationParams) Simulation {
	return Simulation{
		logger:       slog.Default().With("area", "Simulation"),
		params:       params,
		connections:  []*dummy.DummyClient{},
		rand:         rand.New(rand.NewSource(params.Seed)),
		mutex:        sync.Mutex{},
		totalAdds:    0,
		totalRemoves: 0,
	}
}

func (s *Simulation) push(client *dummy.DummyClient) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
    fmt.Fprintf(os.Stderr, "push: %s\n", client.GameServerAddr())
	s.connections = append(s.connections, client)
}

func (s *Simulation) nextInt(min int, max int) int {
	out := s.rand.Int()
	diff := max - min
	if diff == 0 {
		return min
	}

	return min + out%diff
}

func (s *Simulation) removeRandom() *dummy.DummyClient {
	s.mutex.Lock()

	defer s.mutex.Unlock()
	idx := s.rand.Int() % len(s.connections)
	s.logger.Info("SimRound removing connection", "idx", idx)
    client := s.connections[idx]
	client.Disconnect()

    fmt.Fprintf(os.Stderr, "removeRandom: %s\n", client.GameServerAddr())
	s.connections = append(s.connections[0:idx], s.connections[idx+1:]...)

	return client
}

func (s *Simulation) String() string {
	return fmt.Sprintf(`----- Simulation -----
adds: %d (%d)
removes: %d (%d)
round: %d
`, s.adds, s.totalAdds, s.removes, s.totalRemoves, s.currentRound)
}

func (s *Simulation) client(ctx context.Context, wait *sync.WaitGroup) *dummy.DummyClient {
	s.logger.Log(ctx, prettylog.LevelTrace, "client connecting...")
	client := dummy.NewDummyClient(s.params.Host, s.params.Port)

    go func() {
        err := client.Connect(ctx)
        assert.NoError(err, "unable to connect to client")
        client.WaitForReady()
        s.logger.Log(ctx, prettylog.LevelTrace, "client connected")
        wait.Done()
    }()

	return &client
}

func compareServerStates(before []gameserverstats.GameServerConfig, after []gameserverstats.GameServerConfig, adds []*dummy.DummyClient, removes []*dummy.DummyClient) {
    beforeValidator := sumConfigConns(before)
    afterValidator := sumConfigConns(after)

    beforeValidator.Add(adds)
    beforeValidator.Remove(removes)

    beforeKeysIter := maps.Keys(beforeValidator)
    afterKeysIter := maps.Keys(afterValidator)

    beforeKeys := slices.SortedFunc(beforeKeysIter, func(a, b string) int {
        return strings.Compare(a, b)
    })
    afterKeys := slices.SortedFunc(afterKeysIter, func(a, b string) int {
        return strings.Compare(a, b)
    })

    assert.Assert(len(beforeKeys) == len(afterKeys), "before and after keys have different lengths", "before", beforeKeys, "after", afterKeys)
    for i, v := range beforeKeys {
        assert.Assert(afterKeys[i] == v, "before and after key order doesn't match", "i", i, "before", v, "after", afterKeys[i])
        if beforeValidator[v] != afterValidator[v] {
            fmt.Fprintf(os.Stderr, "--------------- Validation Failed ---------------\n")

            b := sumConfigConns(before)
            fmt.Fprintf(os.Stderr, "server state before:\n%s\n", b.String())
            fmt.Fprintf(os.Stderr, "server state after:\n%s\n", afterValidator.String())
            fmt.Fprintf(os.Stderr, "Adds:\n")
            for i, c := range adds {
                fmt.Fprintf(os.Stderr, "%d: %s\n", i, c.GameServerAddr())
            }
            fmt.Fprintf(os.Stderr, "Removes:\n")
            for i, c := range removes {
                fmt.Fprintf(os.Stderr, "%d: %s\n", i, c.GameServerAddr())
            }
            assert.Never("expected vs received connection count mismatch", "failedOn", v, "expected", afterValidator, "received", beforeValidator)
        }
    }
}

func (s *Simulation) RunSimulation(ctx context.Context) error {
	s.Done = false

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

		start := time.Now()
		servers, err := s.params.Stats.GetAllGameServerConfigs()
        assert.NoError(err, "unable to get all game servers")
		startConnCount := s.params.Stats.GetTotalConnectionCount()
		adds := int(math.Abs(s.rand.NormFloat64() * float64(s.params.StdConnections)))
		removes := int(math.Abs(s.rand.NormFloat64() * float64(s.params.StdConnections)))
		s.logger.Info("SimRound", "round", round, "current", startConnCount, "adds", adds, "removes", removes)
		addedConns := []*dummy.DummyClient{}
		removedConns := []*dummy.DummyClient{}

		wait := sync.WaitGroup{}
		wait.Add(2)
		go func() {
			s.adds = adds

            addWait := sync.WaitGroup{}
            addWait.Add(adds)

			for range adds {
				s.adds -= 1

                // So here is the problem

                // 1. the server is starting up and there are many connections being added.
                // this function will finish shortly and the wait.Done function will be called...
				<-time.NewTimer(time.Millisecond * time.Duration(s.nextInt(s.params.ConnectionSleepMinMS, s.params.ConnectionSleepMaxMS))).C
				addedConns = append(addedConns, s.client(ctx, &addWait))
			}

            // ok i wonder if more than one wait can work with wait groups...
            // there we go... i hope this works?
            addWait.Wait()
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
				<-time.NewTimer(time.Millisecond * time.Duration(s.nextInt(s.params.ConnectionSleepMinMS, s.params.ConnectionSleepMaxMS))).C
				removedConns = append(removedConns, s.removeRandom())
				actualRemoves++
			}
			wait.Done()
		}()

		wait.Wait()

		start = time.Now()
		expected := startConnCount.Connections + adds - actualRemoves
		s.totalAdds += adds
		s.totalRemoves += actualRemoves

		for time.Now().Sub(start).Milliseconds() < int64(s.params.TimeToConnectionCountMS) {
			conns := s.params.Stats.GetTotalConnectionCount()
			if conns.Connections == expected &&
                conns.ConnectionsAdded == startConnCount.ConnectionsAdded + adds &&
                conns.ConnectionsRemoved == startConnCount.ConnectionsRemoved + actualRemoves {
                break
            }
			<-time.NewTimer(time.Millisecond * 10).C
		}

		for _, c := range addedConns {
			assert.Assert(c.State == dummy.CSConnected, "state of connection is not connected", "state", dummy.ClientStateToString(c.State))
            s.push(c)
		}

		for _, c := range removedConns {
			assert.Assert(c.State == dummy.CSDisconnected, "state of connection is not disconnected", "state", dummy.ClientStateToString(c.State))
		}

		serversAfter, err := s.params.Stats.GetAllGameServerConfigs()
        assert.NoError(err, "unable to get all game servers")
        compareServerStates(servers, serversAfter, addedConns, removedConns)
		s.logger.Info("SimRound finished", "round", round, "totalAdds", s.totalAdds, "totalRemoves", s.totalRemoves, "time taken ms", time.Now().Sub(start).Milliseconds())
	}

    s.logger.Warn("Simulation Completed")
	s.Done = true
	return nil
}
