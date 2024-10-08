package e2etests

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	"vim-arcade.theprimeagen.com/pkg/matchmaking"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)

type ServerState struct {
    Sqlite *gameserverstats.Sqlite
    Server *servermanagement.LocalServers
    MatchMaking *matchmaking.MatchMakingServer
    Port int
    Factory *TestingClientFactory
    Conns ConnMap
}

type TestingClientFactory struct {
    host string
    port uint16
    logger *slog.Logger
}

func NewTestingClientFactory(host string, port uint16, logger *slog.Logger) TestingClientFactory {
    return TestingClientFactory{
        logger: logger.With("area", "TestClientFactory"),
        host: host,
        port: port,
    }
}

func (f TestingClientFactory) WithPort(port uint16) TestingClientFactory {
    f.port = port
    return f
}

// this is getting hacky...
func (f *TestingClientFactory) New(wait *sync.WaitGroup) *dummy.DummyClient {
    client := dummy.NewDummyClient(f.host, f.port)
    f.logger.Info("creating new client", "id", client.ConnId)

    go func() {
        defer wait.Done()

        f.logger.Info("client connecting", "id", client.ConnId)
        client.Connect(context.Background())
        f.logger.Info("client connected", "id", client.ConnId)
    }()

    return &client
}

func createServer(ctx context.Context, server *ServerState, logger *slog.Logger) (string, *gameserverstats.GameServerConfig) {
    logger.Info("creating server")
    sId, err := server.Server.CreateNewServer(ctx)
    logger.Info("created server", "id", sId, "err", err)
    assert.NoError(err, "unable to create server")
    logger.Info("waiting server...", "id", sId)
    server.Server.WaitForReady(ctx, sId)
    logger.Info("server ready", "id", sId)
    sConfig := server.Sqlite.GetById(sId)
    logger.Info("server config", "config", sConfig)
    assert.NotNil(sConfig, "unable to get config by id", "id", sId)
    return sId, sConfig
}

func createConnections(ctx context.Context, count int, factory *TestingClientFactory, logger *slog.Logger) []*dummy.DummyClient {
    conns := make([]*dummy.DummyClient, 0)

    wait := sync.WaitGroup{}
    wait.Add(count)
    logger.Info("creating all clients", "count", count)
    for range count {
        conns = append(conns, factory.New(&wait))
    }
    wait.Wait()
    logger.Info("clients all created", "count", count)

    return conns
}

type ConnMap map[string][]*dummy.DummyClient

func hydrateServers(ctx context.Context, server *ServerState, logger *slog.Logger) ConnMap {
    configs, err := server.Sqlite.GetAllGameServerConfigs()
    assert.NoError(err, "unable to get game server configs")

    connMap := make(ConnMap)
    for _, c := range configs {

        sId, sConfig := createServer(ctx, server, logger)
        factory := server.Factory.WithPort(uint16(sConfig.Port))
        conns := createConnections(ctx, c.Connections, &factory, logger)

        connMap[sId] = conns
    }

    return connMap
}

func copyDBFile(path string) string {
    contents, err := os.ReadFile(path)
    assert.NoError(err, "unable to read the contents")
    f, err := os.CreateTemp("/tmp", "mm-testing-")
    assert.NoError(err, "unable to create tmp")
    fName := f.Name()
    n, err := f.Write(contents)
    assert.NoError(err, "unable to write to the file")
    assert.Assert(n != len(contents), "unable to write all the data to the temp file", "n", n, "len(contents)", len(contents))
    assert.NoError(f.Close(), "unable to close file")

    return fName
}

func getDBPath(name string) string {
    return fmt.Sprintf("./e2e-tests/data/%s", name)
}

func createEnvironment(ctx context.Context, path string, params servermanagement.ServerParams) ServerState {
    path = copyDBFile(path)

    logger := slog.Default().With("area", "create-env")
    port, err := dummy.GetFreePort()
    assert.NoError(err, "unable to get a free port")

    logger.Info("creating sqlite", "path", path)
    sqlite := gameserverstats.NewSqlite(path)
    logger.Info("creating local servers", "params", params)
    local := servermanagement.NewLocalServers(sqlite, params)
    logger.Info("creating matchmaking", "port", port)

    mm := matchmaking.NewMatchMakingServer(matchmaking.MatchMakingServerParams{
        Port: port,
        GameServer: &local,
    })
    factory := NewTestingClientFactory("", uint16(port), logger)
    server := ServerState{
        Sqlite: sqlite,
        Server: &local,
        MatchMaking: mm,
        Port: port,
        Factory: &factory,
        Conns: nil,
    }

    logger.Info("hydrating servers", "port", port)
    server.Conns = hydrateServers(ctx, &server, logger)

    return server
}

