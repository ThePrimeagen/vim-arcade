package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"vim-arcade.theprimeagen.com/cmd/test-matchmaking/sim"
	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/ctrlc"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	"vim-arcade.theprimeagen.com/pkg/matchmaking"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)

func createMatchMaking() (servermanagement.LocalServers, *gameserverstats.Sqlite, *matchmaking.MatchMakingServer) {
    _, port := dummy.GetHostAndPort()

    path := "/tmp/sim.db"
    gameserverstats.ClearSQLiteFiles(path)

    path = gameserverstats.EnsureSqliteURI(path)
    db := gameserverstats.NewSqlite(path)
    os.Setenv("SQLITE", path)
    db.SetSqliteModes()
    err := db.CreateGameServerConfigs()
    assert.NoError(err, "unable to create game server configs")

    configs, err := db.GetAllGameServerConfigs()
    assert.NoError(err, "unable to get server configs")
    assert.Assert(len(configs) == 0, "expected the server to be free on configs", "configs", configs)

    local := servermanagement.NewLocalServers(db, servermanagement.ServerParams{
        MaxLoad: 0.9,
    })

    return local, db, matchmaking.NewMatchMakingServer(matchmaking.MatchMakingServerParams{
        Port: port,
        GameServer: &local,
    })
}

func main() {
    var inline bool
    flag.BoolVar(&inline, "inline", false, "if logging and display output should both go to stdout")
    flag.Parse()

    fh := os.Stderr
    if inline {
        fh = os.Stdout
    }

    logger := prettylog.SetProgramLevelPrettyLogger(prettylog.NewParams(fh))

    slog.SetDefault(logger.With("process", "sim"))
    logger = slog.Default().With("area", "TestMatchMaking")
    local, db, mm := createMatchMaking()
    ctx, cancel := context.WithCancel(context.Background())

    defer mm.Close()
    go func() {
        err := mm.Run(ctx)
        if err != nil {
            logger.Error("MatchMaking Run exited with an error", "err", err)
        }
        cancel()
    }()
    go db.Run(ctx)
    go ctrlc.HandleCtrlC(cancel)
    mm.WaitForReady(ctx)
    s := sim.NewSimulation(sim.SimulationParams{
        Seed: 69,
        Rounds: 50000,
        Host: "",
        Port: uint16(mm.Params.Port),
        Stats: db,
        StdConnections: 500,
        TimeToConnectionCountMS: 5000,
        ConnectionSleepMinMS: 50,
        ConnectionSleepMaxMS: 75,
    })
    go s.RunSimulation(ctx)
    go local.Run(ctx)

    if !inline {
        fmt.Printf("[2J[1;1H\n")
    }
    count := 0
    var ticker *time.Ticker
    if inline {
        ticker = time.NewTicker(time.Second * 2)
    } else {
        ticker = time.NewTicker(time.Millisecond * 500)
    }

    for !s.Done {
        <-ticker.C
        count++
        if !inline {
            fmt.Printf("[2J[1;1H\n")
        }
        fmt.Printf("%s\n", s.String())
        fmt.Printf("%s\n", mm.String())
    }

    cancel()
}
