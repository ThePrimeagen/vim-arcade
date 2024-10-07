package sim

import (
	"context"
	"log/slog"
	"math/rand"
	"sync"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/dummy"
)

type SimulationConnections struct {
	clients []*dummy.DummyClient
	adds    []*dummy.DummyClient
	removes []*dummy.DummyClient
	factory TestingClientFactory
	m       sync.Mutex
	rand    *rand.Rand
	logger  *slog.Logger
	wait    sync.WaitGroup
}

func NewSimulationConnections(f TestingClientFactory, r *rand.Rand) SimulationConnections {
	return SimulationConnections{
		m:       sync.Mutex{},
		clients: []*dummy.DummyClient{},
		adds: []*dummy.DummyClient{},
		removes: []*dummy.DummyClient{},
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

func (s *SimulationConnections) StartRound() {
	s.wait = sync.WaitGroup{}
}

func (s *SimulationConnections) AssertAddsAndRemoves() {
    for _, c := range s.adds {
        assert.Assert(c.State == dummy.CSConnected, "state of connection is not connected", "state", dummy.ClientStateToString(c.State))
    }

    for _, c := range s.removes {
        assert.Assert(c.State == dummy.CSDisconnected, "state of connection is not disconnected", "state", dummy.ClientStateToString(c.State))
    }

}

func (s *SimulationConnections) FinishRound() ([]*dummy.DummyClient, []*dummy.DummyClient) {
	s.wait.Wait()

    removes := s.removes
    adds := s.adds

    s.removes = []*dummy.DummyClient{}
    s.adds = []*dummy.DummyClient{}

    return adds, removes
}

func (s *SimulationConnections) AddBatch(count int) int {
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
	return idx
}

func (s *SimulationConnections) Remove(count int) {
    s.wait.Add(count)
	removal := func(count int) []*dummy.DummyClient {
		out := make([]*dummy.DummyClient, 0, 5)
		s.m.Lock()
		defer s.m.Unlock()

		for range count {
			idx := s.rand.Int() % len(s.clients)
			out = append(out, s.clients[idx])
			s.clients = append(s.clients[0:idx], s.clients[idx+1:]...)
		}

        s.removes = append(s.removes, out...)
		return out
	}

	removes := removal(count)
	for _, c := range removes {
		c.Disconnect()
        s.wait.Done()
		s.logger.Warn("Disconnect Client", "addr", c.GameServerAddr())
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

func (f *TestingClientFactory) CreateBatchedConnectionsWithWait(count int, wait *sync.WaitGroup) []*dummy.DummyClient {
	conns := make([]*dummy.DummyClient, 0)

	f.logger.Info("creating all clients", "count", count)
	for range count {
		conns = append(conns, f.NewWait(wait))
	}
	f.logger.Info("clients all created", "count", count)

	return conns
}

func (f *TestingClientFactory) CreateBatchedConnections(count int) []*dummy.DummyClient {
	wait := &sync.WaitGroup{}
	clients := f.CreateBatchedConnectionsWithWait(count, wait)

    f.logger.Info("CreateBatchedConnections waiting", "count", count)
	wait.Wait()

	return clients
}

func (f TestingClientFactory) WithPort(port uint16) TestingClientFactory {
	f.port = port
	return f
}

func (f *TestingClientFactory) New() *dummy.DummyClient {
	client := dummy.NewDummyClient(f.host, f.port)
	f.logger.Info("factory connecting", "id", client.ConnId)
	client.Connect(context.Background())
	f.logger.Info("factory connected", "id", client.ConnId)
	return &client
}

// this is getting hacky...
func (f *TestingClientFactory) NewWait(wait *sync.WaitGroup) *dummy.DummyClient {
	wait.Add(1)
	client := dummy.NewDummyClient(f.host, f.port)
	f.logger.Info("factory new client with wait", "id", client.ConnId)

	go func() {
		defer wait.Done()

		f.logger.Info("factory client connecting with wait", "id", client.ConnId)
		client.Connect(context.Background())
		f.logger.Info("factory client connected with wait", "id", client.ConnId)
	}()

	return &client
}
