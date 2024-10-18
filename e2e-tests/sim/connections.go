package sim

import (
	"context"
	"encoding/binary"
	"log/slog"
	"math/rand"
	"slices"
	"sync"

	"vim-arcade.theprimeagen.com/pkg/api"
	"vim-arcade.theprimeagen.com/pkg/assert"
)

var clientId uint64 = 0
func getNextId() [16]byte {
    id := [16]byte{}
    binary.BigEndian.PutUint64(id[:], clientId)
    clientId++

    return id
}

type SimulationConnections struct {
	clients []*api.Client
	adds    []*api.Client
	removes []*api.Client
	factory TestingClientFactory
	m       sync.Mutex
	rand    *rand.Rand
	logger  *slog.Logger
	wait    sync.WaitGroup
}

func NewSimulationConnections(f TestingClientFactory, r *rand.Rand) SimulationConnections {
	return SimulationConnections{
		m:       sync.Mutex{},
		clients: []*api.Client{},
		adds:    []*api.Client{},
		removes: []*api.Client{},
		factory: f,
		rand:    r,
		logger:  slog.Default().With("area", "SimulationConnections"),
	}
}

func (s *SimulationConnections) Len() int {
	s.m.Lock()
	defer s.m.Unlock()
	return len(s.clients)
}

func (s *SimulationConnections) StartRound(adds int, removes int) {
	s.wait = sync.WaitGroup{}

	s.wait.Add(adds)
	s.wait.Add(removes)
}

func (s *SimulationConnections) AssertAddsAndRemoves() {
	for _, c := range s.adds {
		assert.Assert(c.State == api.CSConnected, "state of connection is not connected", "state", api.ClientStateToString(c.State))
	}

	for _, c := range s.removes {
		assert.Assert(c.State == api.CSDisconnected, "state of connection is not disconnected", "state", api.ClientStateToString(c.State))
	}

}

func (s *SimulationConnections) FinishRound() ([]*api.Client, []*api.Client) {
	s.wait.Wait()

	removes := s.removes
	adds := s.adds

	s.removes = []*api.Client{}
	s.adds = []*api.Client{}

	return adds, removes
}

func (s *SimulationConnections) AddBatch(count int) int {
    s.logger.Info("Adding connections", "count", count, "len", len(s.clients))
	clients := s.factory.CreateBatchedConnectionsWithWait(count, &s.wait)

	s.m.Lock()
	defer s.m.Unlock()

	idx := len(s.clients)
	s.clients = append(s.clients, clients...)
	s.adds = append(s.adds, clients...)
	return idx
}

func (s *SimulationConnections) Add() int {
	client := s.factory.NewWait(&s.wait)

	s.m.Lock()
	defer s.m.Unlock()

	idx := len(s.clients)
	s.clients = append(s.clients, client)
	s.adds = append(s.adds, client)
	s.logger.Info("Add", "len", len(s.adds))
	return idx
}

func (s *SimulationConnections) Remove(count int) {
    s.logger.Info("Removing", "count", count, "len", len(s.clients))
    length := s.Len()

    out := make([]int, 0, count)

    for range count {
        // obviously this could be a perf nightmare, but i am going to
        // assume its not too bad :)
        for {
            next := NextInt(s.rand, 0, length)
            if slices.Contains(out, next) {
                continue
            }
            out = append(out, next)
            break
        }
    }

    slices.Sort(out)
    slices.Reverse(out)

    removals := []*api.Client{}
    for _, idx := range out {
        s.logger.Warn("Disconnect Client", "serverId", s.clients[idx].ServerId, "addr", s.clients[idx].Addr(), "idx", idx)
        s.clients[idx].Disconnect()
        removals = append(removals, s.clients[idx])
    }

    s.m.Lock()
    defer s.m.Unlock()
    s.removes = append(s.removes, removals...)
    for _, idx := range out {
        s.clients = append(s.clients[:idx], s.clients[idx + 1:]...)
		s.wait.Done()
    }
}

type TestingClientFactory struct {
	host   string
	port   uint16
	logger *slog.Logger
}

func NewTestingClientFactory(host string, port uint16, logger *slog.Logger) TestingClientFactory {
	return TestingClientFactory{
		logger: logger.With("area", "TestClientFactory"),
		host:   host,
		port:   port,
	}
}

func (f *TestingClientFactory) CreateBatchedConnectionsWithWait(count int, wait *sync.WaitGroup) []*api.Client {
	conns := make([]*api.Client, count, count)

	f.logger.Info("creating all clients", "count", count)
    for i := range count {
		conns[i] = f.NewWait(wait)
	}
	f.logger.Info("clients all created", "count", count)

	return conns
}

func (f *TestingClientFactory) CreateBatchedConnections(count int) []*api.Client {

	wait := &sync.WaitGroup{}
    wait.Add(count)
	clients := f.CreateBatchedConnectionsWithWait(count, wait)

	f.logger.Info("CreateBatchedConnections waiting", "count", count)
	wait.Wait()
    f.logger.Info("CreateBatchedConnections finished", "count", count)

	return clients
}

func (f TestingClientFactory) WithPort(port uint16) TestingClientFactory {
	f.port = port
	return f
}

func (f *TestingClientFactory) New() *api.Client {
	client := api.NewClient(f.host, f.port, getNextId())
	f.logger.Info("factory connecting", "id", client.Id())
	client.Connect(context.Background())
    client.WaitForReady()
	f.logger.Info("factory connected", "id", client.Id())
	return &client
}

// this is getting hacky...
func (f *TestingClientFactory) NewWait(wait *sync.WaitGroup) *api.Client {
	client := api.NewClient(f.host, f.port, [16]byte(getNextId()))

    id := client.Id()
	f.logger.Info("factory new client with wait", "id", id)

	go func() {
		defer func() {
			f.logger.Info("factory client connected with wait", "id", id)
			wait.Done()
		}()

		f.logger.Info("factory client connecting with wait", "id", id)
        err := client.Connect(context.Background())
        assert.NoError(err, "unable to connect to mm", "id", id)
		client.WaitForReady()
	}()

	return &client
}
