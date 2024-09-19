package main

import (
	"context"
	"fmt"
	"log/slog"

	"vim-arcade.theprimeagen.com/cmd/test-matchmaking/sim"
	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/ctrlc"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	"vim-arcade.theprimeagen.com/pkg/matchmaking"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)

func createMatchMaking() (servermanagement.LocalServers, *gameserverstats.JSONMemory, *matchmaking.MatchMakingServer) {
    _, port := dummy.GetHostAndPort()

    db, err := gameserverstats.NewJSONMemoryAndClear("/tmp/testing.json")
    assert.NoError(err, "the json database could not be created")

    local := servermanagement.NewLocalServers(db, servermanagement.ServerParams{
        MaxLoad: 0.9,
    })

    return local, db, matchmaking.NewMatchMakingServer(matchmaking.MatchMakingServerParams{
        Port: port,
        GameServer: &local,
    })
}

func main() {
    prettylog.SetProgramLevelPrettyLogger()
    logger := slog.Default().With("area", "TestMatchMaking")
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
    go ctrlc.HandleCtrlC(cancel)
    mm.WaitForReady(ctx)
    s := sim.NewSimulation(sim.SimulationParams{
        Seed: 69,
        Rounds: 100,
        Host: "",
        Port: uint16(mm.Params.Port),
        Stats: db,
        MaxConnections: 100,
    })
    go s.RunSimulation(ctx)
    go local.Run(ctx)

    fmt.Printf("[2J[1;1H\n")
    for !s.Done {
        fmt.Printf("[;H")
        fmt.Printf("%s\n", mm.String())
    }

    cancel()
}
