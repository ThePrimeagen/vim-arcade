package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"vim-arcade.theprimeagen.com/pkg/assert"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
)

type E2EConfig struct {
    servers []gameserverstats.GameServerConfig
}

func main() {
    nameStr := ""
    flag.StringVar(&nameStr, "name", "", "the name of the data file")
    flag.Parse()

    assert.Assert(nameStr != "", "expected --name to be provided")
    name := fmt.Sprintf("e2e-tests/data/%s", nameStr)
    configPath := fmt.Sprintf("e2e-tests/run/configs/%s", nameStr)

    configBytes, err := os.ReadFile(configPath)
    assert.NoError(err, "unable to read config file")

    var config E2EConfig
    err = json.Unmarshal(configBytes, &config)
    assert.NoError(err, "unable to unmarshal config")

    sqlite := gameserverstats.NewSqlite("file:" + name)
    sqlite.SetSqliteModes()
    err = sqlite.CreateGameServerConfigs()
    assert.NoError(err, "unable to create game server config")

    for _, c := range config.servers {
        fmt.Printf("inserting: %+v\n", c)
        assert.NoError(sqlite.Update(c), "unable to update sqlite with config", "config", c)
    }

    time.Sleep(time.Millisecond * 500)
    assert.NoError(sqlite.Close(), "unable to close sqlite")
    fmt.Printf("sqlite configuration finished: %s\n", nameStr)
}

