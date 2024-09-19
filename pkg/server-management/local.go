package servermanagement

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"vim-arcade.theprimeagen.com/pkg/cmd"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
)

type LocalServers struct {
    logger *slog.Logger
    stats gameserverstats.GSSRetriever
    params ServerParams
    servers []*cmd.Cmder
}

func getEnvVars() []string {
    return []string{
        fmt.Sprintf("GOPATH=%s", os.Getenv("GOPATH")),
    }
}

func NewLocalServers(stats gameserverstats.GSSRetriever, params ServerParams) LocalServers {
    return LocalServers{
        stats: stats,
        params: params,
        servers: []*cmd.Cmder{},
        logger: slog.Default().With("area", "LocalServers"),
    }
}

func (l *LocalServers) GetBestServer() (string, error) {
    var bestServer gameserverstats.GameServerConfig
    found := false

    for _, s := range l.stats.Iter() {
        if s.Load < l.params.MaxLoad {
            if found && bestServer.Load < s.Load || !found {
                bestServer = s;
                found = true
            }
        }
    }

    if !found {
        return "", NoBestServer
    }

    return bestServer.Addr(), nil
}

var id = 0
func (l *LocalServers) CreateNewServer(ctx context.Context) (string, error) {
    outId := id
    cmdr := cmd.NewCmder("go", ctx).
        AddVArgv([]string{"run", "./cmd/dummy-server/main.go"}).
        WithOutFn(func(b []byte) (int, error) {
            l.logger.Info("local server stdout", "stdout", string(b), "id", outId)
            return len(b), nil
        }).
        WithErrFn(func(b []byte) (int, error) {
            l.logger.Error("local server stderr", "stderr", string(b), "id", outId)
            return len(b), nil
        })

    id++

    go func() {
        err := cmdr.Run(append(getEnvVars(), fmt.Sprintf("ID=%d", outId)))
        if err != nil {
            l.logger.Error("unable to run cmdr", "err", err)
        }
    }()

    l.servers = append(l.servers, cmdr)
    return fmt.Sprintf("%d", outId), nil
}

// TODO Add timeout...?
func (l *LocalServers) WaitForReady(ctx context.Context, id string) error {
    for {
        time.Sleep(time.Millisecond * 50)
        gs := l.stats.GetById(id)
        if gs != nil {
            if gs.State == gameserverstats.GSStateReady {
                return nil
            } else if gs.State == gameserverstats.GSStateClosed {
                // TODO Add closed error
                return nil
            }
        }
    }
}

func (l *LocalServers) GetConnectionString(id string) (string, error) {
    gs := l.stats.GetById(id)
    if gs == nil {
        // TODO Handle DNE error
        return "", nil
    }
    return fmt.Sprintf("%s:%d", gs.Host, gs.Port), nil
}


func (l *LocalServers) refresh() {
    // TODO determine if i should kill any server
}

func (l *LocalServers) Run(ctx context.Context) {

    // TODO(v1) make this configurable
    timer := time.NewTicker(time.Second * 30)
    defer timer.Stop()

    outer:
    for {
        select {
        case <-ctx.Done():
            break outer
        case <-timer.C:
            l.refresh()
        }
    }
}

func (l *LocalServers) Close() {
    for _, c := range l.servers {
        c.Close()
    }
}

func (l *LocalServers) Ready() {
    // TODO i should maybe create one server IF there are no servers
}

func (l *LocalServers) String() string {
    servers := []string{}
    for i, s := range l.stats.Iter() {
        servers = append(servers, fmt.Sprintf("%d: %s", i, s.String()))
    }

    return strings.Join(servers, "\n")
}
