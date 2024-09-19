package servermanagement

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/cmd"
	gameserverstats "vim-arcade.theprimeagen.com/pkg/game-server-stats"
)

type LocalServers struct {
	logger  *slog.Logger
	stats   gameserverstats.GSSRetriever
	params  ServerParams
	servers []*cmd.Cmder

    load        float32
    connections float32

	lastTimeNoConnections bool

}

func getEnvVars() []string {
	return []string{
		fmt.Sprintf("GOPATH=%s", os.Getenv("GOPATH")),
	}
}

func NewLocalServers(stats gameserverstats.GSSRetriever, params ServerParams) LocalServers {
	return LocalServers{
		stats:                 stats,
		params:                params,
		servers:               []*cmd.Cmder{},
		logger:                slog.Default().With("area", "LocalServers"),
		lastTimeNoConnections: false,
	}
}

func (l *LocalServers) GetBestServer() (string, error) {
	servers := l.stats.GetServersByUtilization(float64(l.params.MaxLoad))

	if len(servers) == 0 {
		l.logger.Info("GetBestServer no servers found")
		return "", NoBestServer
	}

	l.logger.Info("GetBestServer server returned", "server", servers[0].String())
	return servers[0].Id, nil
}

var id = 0

func (l *LocalServers) CreateNewServer(ctx context.Context) (string, error) {
	outId := id
	cmdr := cmd.NewCmder("go", ctx).
		AddVArgv([]string{"run", "./cmd/dummy-server/main.go"}).
		WithOutFn(func(b []byte) (int, error) {
			l.logger.Info(string(b))
			return len(b), nil
		}).
		WithErrFn(func(b []byte) (int, error) {
			l.logger.Error(string(b))
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
		stats, err := l.stats.GetAllGameServerConfigs()
		assert.NoError(err, "unable to get the stats", "err", err)
		for _, s := range stats {
			l.logger.Info("WaitForReady#getStats", "state", s.State, "id", s.Id, "connections", s.Connections, "port", s.Port)
		}

		gs := l.stats.GetById(id)
		l.logger.Info("WaitForReady", "id", id, "gs", gs)
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
	gameServers := l.stats.GetServersByUtilization(1500)
	for _, gs := range gameServers {
		servers = append(servers, gs.String())
	}
	return strings.Join(servers, "\n")
}
