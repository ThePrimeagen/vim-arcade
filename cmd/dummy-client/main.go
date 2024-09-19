package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
)


func main() {
    port := uint(0)
    flag.UintVar(&port, "port", 0, "the port to connect the dummy client to")
    flag.Parse()

    assert.Assert(port > 0, "expected port to be provided", "port", port)

    // TODO logging customization through some sort of config/env
    prettylog.SetProgramLevelPrettyLogger(prettylog.NewParams(os.Stderr))

    client := dummy.NewDummyClient("", uint16(port))

    err := client.Connect(context.Background())
    if err != nil {
        slog.Error("unable to connect client", "err", err)
        return
    }

    client.WaitForDone()
}

