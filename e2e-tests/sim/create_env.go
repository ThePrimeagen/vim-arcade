package sim

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path"

	amproxy "vim-arcade.theprimeagen.com/pkg/am-proxy"
	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)

type ServerCreationConfig struct {
    From gameserverstats.GameServerConfig
    To gameserverstats.GameServerConfig
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

type ConnMap map[string][]*dummy.DummyClient

func hydrateServers(ctx context.Context, server *ServerState, logger *slog.Logger) (ConnMap, []ServerCreationConfig) {
    configs, err := server.Sqlite.GetAllGameServerConfigs()
    assert.NoError(err, "unable to get game server configs")
    clearCreationConfigs(server, configs)

    connMap := make(ConnMap)
    configMapper := []ServerCreationConfig{}
    logger.Info("Hydrating Servers", "count", len(configs))
    for _, c := range configs {

        logger.Info("Creating server with the following config", "config", c)

        sId, sConfig := createServer(ctx, server, logger)
        factory := server.Factory.WithPort(uint16(sConfig.Port))
        conns := factory.CreateBatchedConnections(c.Connections)

        connMap[sId] = conns
        configMapper = append(configMapper, ServerCreationConfig{
            From: c,
            To: *sConfig,
        })
    }

    return connMap, configMapper
}

func copyFile(from string, to string) {
    toFd, err := os.OpenFile(to, os.O_RDWR|os.O_CREATE, 0644)
    assert.NoError(err, "unable to open toFile")
    defer toFd.Close()

    fromFd, err := os.Open(from)
    assert.NoError(err, "unable to open toFile")
    defer fromFd.Close()

    _, err = io.Copy(toFd, fromFd)
    assert.NoError(err, "unable to copy file")
}

func copyDBFile(path string) string {

    f, err := os.CreateTemp("/tmp", "mm-testing-")
    assert.NoError(err, "unable to create tmp")
    fName := f.Name()
    f.Close()

    copyFile(path, fName)
    copyFile(path + "-shm", fName + "-shm")
    copyFile(path + "-wal", fName + "-wal")

    return fName
}

func GetDBPath(name string) string {
    cwd, err := os.Getwd()
    assert.NoError(err, "no cwd?")

    // assert: windows sucks
    return path.Join(cwd, "data", name)
}

func clearCreationConfigs(server *ServerState, configs []gameserverstats.GameServerConfig) {
    for _, c := range configs {
        server.Sqlite.DeleteGameServerConfig(c.Id)
    }
}

func CreateEnvironment(ctx context.Context, path string, params servermanagement.ServerParams) ServerState {
    logger := slog.Default().With("area", "create-env")
    logger.Warn("copying db file", "path", path)
    path = copyDBFile(path)
    os.Setenv("SQLITE", path)
    os.Setenv("ENV", "TESTING")

    port, err := dummy.GetFreePort()
    assert.NoError(err, "unable to get a free port")

    logger.Info("creating sqlite", "path", path)
    sqlite := gameserverstats.NewSqlite(gameserverstats.EnsureSqliteURI(path))
    logger.Info("creating local servers", "params", params)
    local := servermanagement.NewLocalServers(sqlite, params)
    logger.Info("creating matchmaking", "port", port)

    proxy := amproxy.NewAMProxy(ctx, &local, amproxy.CreateTCPConnectionFrom)
    tcpProxy := amproxy.NewTCPProxy(&proxy, uint16(port))
    go tcpProxy.Run(ctx)
    tcpProxy.WaitForReady(ctx)

    logger.Info("creating client factory", "port", port)
    factory := NewTestingClientFactory("0.0.0.0", uint16(port), logger)

    logger.Info("creating server state object", "port", port)
    server := ServerState{
        Sqlite: sqlite,
        Server: &local,
        Proxy: &tcpProxy,
        Port: port,
        Factory: &factory,
        Conns: nil,
    }

    logger.Info("hydrating servers", "port", port)
    conns, configs := hydrateServers(ctx, &server, logger)
    server.Conns = conns

    AssertServerStateCreation(&server, configs)

    logger.Info("environment fully created")
    return server
}

