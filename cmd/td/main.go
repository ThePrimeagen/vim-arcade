package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
	"vim-arcade.theprimeagen.com/pkg/td"
)

func getId() string {
    return "123-local"
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

func handleCtrlC(cancel context.CancelFunc) {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)
    go func() {
        <-c
        cancel()
        time.Sleep(time.Millisecond * 250)
        // Run Cleanup
        os.Exit(1)
    }()
}

func main() {
    godotenv.Load()

    // TODO make this... well better?
    // Right now we have no local vs flyio stuff... iots just me programming
    db := gameserverstats.NewJSONMemory(os.Getenv("IN_MEMORY_JSON"))
    host, port := getHostAndPort()
    config := gameserverstats.GameServerConfig {
        Connections: 0,
        Load: 0,
        Id: getId(),
        Host: host,
        Port: port,
    }

    server := td.NewTowerDefense(&db, config)
    ctx, cancel := context.WithCancel(context.Background())
    handleCtrlC(cancel)

    defer server.Close()
    go func () {
        err := server.Run(ctx)
        if err != nil {
            slog.Error("Game Server Run came returned with an error", "error", err)
            cancel()
        }
    }()

    server.Wait()
    cancel()
}
