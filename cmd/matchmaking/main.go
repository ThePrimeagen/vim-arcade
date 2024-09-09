package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"vim-arcade.theprimeagen.com/pkg/ctrlc"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	"vim-arcade.theprimeagen.com/pkg/pretty-log"
	"vim-arcade.theprimeagen.com/pkg/matchmaking"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)


func main() {
    godotenv.Load()
    prettylog.SetProgramLevelPrettyLogger()
    port, err := strconv.Atoi(os.Getenv("MM_PORT"))
    logger := slog.Default().With("area", "MatchMakingMain")

    if err != nil {
        slog.Error("port parsing error", "port", port)
        os.Exit(1)
    }

    db := gameserverstats.NewJSONMemory(os.Getenv("IN_MEMORY_JSON"))

    local := servermanagement.NewLocalServers(&db, servermanagement.ServerParams{
        MaxConnections: 10,
        MaxLoad: 0.9,
    })

    mm := matchmaking.NewMatchMakingServer(matchmaking.MatchMakingServerParams{
        Port: port,
        GameServer: &local,
    })

    ctx, cancel := context.WithCancel(context.Background())
    ctrlc.HandleCtrlC(cancel)
    err = mm.Run(ctx)
    logger.Warn("mm main finished", "error", err)
}

