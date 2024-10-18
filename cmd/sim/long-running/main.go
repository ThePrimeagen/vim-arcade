package main

import (
	"flag"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path"
	"time"

	"vim-arcade.theprimeagen.com/e2e-tests/sim"
	"vim-arcade.theprimeagen.com/pkg/assert"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)

func main() {
    var inline bool
    flag.BoolVar(&inline, "inline", false, "if logging and display output should both go to stdout")
    flag.Parse()

    fh := os.Stderr
    if inline {
        fh = os.Stdout
    }

    logger := prettylog.CreateLoggerFromEnv(fh)
    slog.SetDefault(logger.With("process", "sim").With("area", "long-running"))

    ctx := sim.TopLevelContext()
    sim.HandleCtrlC()

    cwd, err := os.Getwd()
    assert.NoError(err, "unable to get cwd")
    p := path.Join(cwd, "e2e-tests/data/no_server")
    state := sim.CreateEnvironment(ctx, p, servermanagement.ServerParams{
        MaxLoad: 0.9,
    })

    defer state.Close()

    s := sim.NewSimulation(sim.SimulationParams{
        Seed: 69,
        Rounds: 3600,
        Host: "",
        Port: uint16(state.Port),

        ConnectionAdds: sim.Sin{
            Amplitude: 4.0,
            Period: 2 * math.Pi,
            Offset: 0,
        },

        ConnectionRemoves: sim.Sin{
            Amplitude: 4.0,
            Period: 2 * math.Pi,
            Offset: math.Pi,
        },

        ConnectionAddRem: sim.StatRange{
            Std: 4,
            Avg: 15,
            Max: 30,
        },

        RoundsToPeriod: 360,

        TimeToConnectionCountMS: 5000,

    }, &state)

    go s.RunSimulation(ctx)

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
        fmt.Printf("%s\n", state.String())
    }

    sim.CancelTopLevelContext()
}
