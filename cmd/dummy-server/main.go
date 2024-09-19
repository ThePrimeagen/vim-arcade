package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"github.com/joho/godotenv"
	"vim-arcade.theprimeagen.com/pkg/ctrlc"
	"vim-arcade.theprimeagen.com/pkg/dummy"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
)

func getId() string {
    return os.Getenv("ID")
}

func getHostAndPort() (string, int) {

    port, err := getFreePort()
    if err != nil {
        port = 42069
    }
    return "0.0.0.0", port
}

// GetFreePort asks the kernel for a free open port that is ready to use.
func getFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

func main() {
    godotenv.Load()

    prettylog.SetProgramLevelPrettyLogger()
    ll :=  slog.Default().With("area", "dummy-server")

    // TODO make this... well better?
    // Right now we have no local vs flyio stuff... iots just me programming
    db := gameserverstats.NewJSONMemory(os.Getenv("IN_MEMORY_JSON"))
    host, port := getHostAndPort()

    config := gameserverstats.GameServerConfig {
        State: gameserverstats.GSStateReady,
        Connections: 0,
        Load: 0,
        Id: getId(),
        Host: host,
        Port: port,
    }

    ll.Info("creating server", "port", port, "host", host)
    server := dummy.NewDummyGameServer(&db, config)
    ctx, cancel := context.WithCancel(context.Background())
    ctrlc.HandleCtrlC(cancel)

    defer server.Close()
    go func () {
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
