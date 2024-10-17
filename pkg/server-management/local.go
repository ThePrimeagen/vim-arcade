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
        fmt.Sprintf("SQLITE=%s", os.Getenv("SQLITE")),
        fmt.Sprintf("DEBUG_TYPE=%s", os.Getenv("DEBUG_TYPE")),
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
    dummyServer := os.Getenv("GAME_SERVER")
    if dummyServer == "" {
        dummyServer = "./cmd/api-server/main.go"
    }
	outId := id
    // TODO i bet there is a better way of doing this...
    // i just don't know other than straight passthrough?
    // i feel like i need more intelligent passing of logs from inner to outer
	cmdr := cmd.NewCmder("go", ctx).
		AddVArgv([]string{"run", dummyServer}).
		WithOutFn(func(b []byte) (int, error) {
			fmt.Fprintf(os.Stdout, "%s", string(b))
			return len(b), nil
		}).
		WithErrFn(func(b []byte) (int, error) {
			fmt.Fprintf(os.Stderr, "%s", string(b))
			return len(b), nil
		})

	id++

	go func() {
        vars := getEnvVars()
        vars = append(vars,
            fmt.Sprintf("ID=%d", outId),

            // subprocesses should not have the log file as it will cause odd
            // log file truncation
            fmt.Sprintf("DEBUG_LOG="),
        )

		err := cmdr.Run(vars)
        cancelled := false
        select {
        case <-ctx.Done():
            cancelled = true
        default:
        }

		if cancelled {
			l.logger.Error("cmdr context killed")
        } else if !cancelled && err != nil {
			l.logger.Error("unable to run cmdr", "err", err)
		}

		// TODO the database checking to prove that this commander has closed
		// properly
		done := false
		select {
		case <-ctx.Done():
			done = true
		default:
            config := l.stats.GetById(fmt.Sprintf("%d", outId))
            if config != nil {
                done = config.State == gameserverstats.GSStateClosed
            }
		}

		if !done {
			assert.Never("cmdr has closed unexpectedly", "id", outId)
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

func (l *LocalServers) refresh(ctx context.Context) {
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
			l.refresh(ctx)
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
