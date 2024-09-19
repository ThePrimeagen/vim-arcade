package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"vim-arcade.theprimeagen.com/pkg/ctrlc"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	"vim-arcade.theprimeagen.com/pkg/matchmaking"
	"vim-arcade.theprimeagen.com/pkg/pretty-log"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)


func main() {
    err := godotenv.Load()
    if err != nil {
        slog.Error("unable to load env", "err", err)
        return
    }

    prettylog.SetProgramLevelPrettyLogger(prettylog.NewParams(os.Stderr))
    slog.SetDefault(slog.Default().With("process", "MatchMaking"))
    slog.Error("Hello world")

    port, err := strconv.Atoi(os.Getenv("MM_PORT"))
    logger := slog.Default().With("area", "MatchMakingMain")

    if err != nil {
        slog.Error("port parsing error", "port", port)
        os.Exit(1)
    }

    db := gameserverstats.NewSqlite("file:/tmp/sim.db")
    db.SetSqliteModes()
    local := servermanagement.NewLocalServers(db, servermanagement.ServerParams{
        MaxLoad: 0.9,
    })
    mm := matchmaking.NewMatchMakingServer(matchmaking.MatchMakingServerParams{
        Port: port,
        GameServer: &local,
    })

    ctx, cancel := context.WithCancel(context.Background())
    ctrlc.HandleCtrlC(cancel)

    go db.Run(ctx)
    err = mm.Run(ctx)

    logger.Warn("mm main finished", "error", err)
}

