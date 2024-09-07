package servermanagement

import (
	"context"
	"time"

	"vim-arcade.theprimeagen.com/pkg/cmd"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
)

type LocalServers struct {
    stats gameserverstats.GSSRetriever
    params ServerParams
    servers []*cmd.Cmder
}

func NewLocalServers(stats gameserverstats.GSSRetriever, params ServerParams) LocalServers {
    return LocalServers{
        stats: stats,
        params: params,
        servers: []*cmd.Cmder{},
    }
}

func (l *LocalServers) GetBestServer() (string, error) {
    var bestServer gameserverstats.GameServerConfig
    found := false

    for _, s := range l.stats.Iter() {
        if s.Connections < l.params.MaxConnections {
            if found && bestServer.Connections < s.Connections || !found {
                bestServer = s;
                found = true
            }
        }
    }

    if !found {
        return "", NO_BEST_SERVER
    }

    return bestServer.Addr(), nil
}

func (l *LocalServers) CreateNewServer(ctx context.Context) {
    cmdr := cmd.NewCmder("go", ctx).
        AddVArgv([]string{"run", "./cmd/td/main.go"})

    go cmdr.Run()

    l.servers = append(l.servers, cmdr)
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
