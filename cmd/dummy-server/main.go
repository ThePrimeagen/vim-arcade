package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/ctrlc"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
)

func getId() string {
    return os.Getenv("ID")
}

func main() {
    godotenv.Load()

    if os.Getenv("DEBUG_LOG") != "" {
        assert.Never("debug log should never be specified for a dummy server")
    }

    sqlitePath := os.Getenv("SQLITE")
    assert.Assert(sqlitePath != "", "you must provide a sqlite env variable to run the simulation dummy server")
    sqlitePath = gameserverstats.EnsureSqliteURI(sqlitePath)

    prettylog.CreateLoggerFromEnv(os.Stderr)
    slog.SetDefault(slog.Default().With("process", "DummyServer"))

    ll :=  slog.Default().With("area", "dummy-server")
    ll.Warn("dummy-server initializing...")

    db := gameserverstats.NewSqlite(sqlitePath)
    db.SetSqliteModes()
    host, port := dummy.GetHostAndPort()

    config := gameserverstats.GameServerConfig {
        State: gameserverstats.GSStateReady,
        Connections: 0,
        Load: 0,
        Id: getId(),
        Host: host,
        Port: port,
    }

    ll.Info("creating server", "port", port, "host", host)
    server := dummy.NewDummyGameServer(db, config)
    ctx, cancel := context.WithCancel(context.Background())
    ctrlc.HandleCtrlC(cancel)

    defer server.Close()
    go db.Run(ctx)
    go func () {
        ll.Info("running server", "port", port, "host", host)
        err := server.Run(ctx)
        if err != nil {
            ll.Error("Game Server Run came returned with an error", "error", err)
            cancel()
        }
    }()

    server.Wait()
    cancel()
    ll.Error("dummy game server finished")
}
