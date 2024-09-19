package test_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	"vim-arcade.theprimeagen.com/pkg/matchmaking"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)

func createMatchMaking(t *testing.T) *matchmaking.MatchMakingServer {
    _, port := dummy.GetHostAndPort()
    prettylog.SetProgramLevelPrettyLogger()

    db, err := gameserverstats.NewJSONMemoryAndClear("/tmp/testing.json")
    require.NoError(t, err, "the json database could not be created")

    local := servermanagement.NewLocalServers(db, servermanagement.ServerParams{
        MaxLoad: 0.9,
    })

    return matchmaking.NewMatchMakingServer(matchmaking.MatchMakingServerParams{
        Port: port,
        GameServer: &local,
    })
}

func connectClient(port int) *dummy.DummyClient {
    client := dummy.NewDummyClient("", uint16(port))
    return &client
}

func TestScaleToOne(t *testing.T) {
}

