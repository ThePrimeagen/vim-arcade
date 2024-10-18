package sim

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	assert "vim-arcade.theprimeagen.com/pkg/assert"
	"vim-arcade.theprimeagen.com/pkg/ctrlc"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
)

type StatRange struct {
	Avg float64
	Std float64
	Max float64
}

var created = false
var ctx context.Context
var cancel context.CancelFunc

func CancelTopLevelContext() {
	TopLevelContext()
	cancel()
}

func TopLevelContext() context.Context {
	if created == false {
		ctx, cancel = context.WithCancel(context.Background())
		created = true
	}

	return ctx
}

func HandleCtrlC() {
	TopLevelContext()
	ctrlc.HandleCtrlC(cancel)
}

func GetNextBatch(r *rand.Rand, stat StatRange) int {
	rNorm := rand.NormFloat64()
	rando := max(min(stat.Max, stat.Avg+rNorm*stat.Std), 0.0)
	return int(rando)
}

func NextInt(r *rand.Rand, min int, max int) int {
	out := rand.Int()
	diff := max - min
	if diff == 0 {
		return min
	}

	return min + out%diff
}

func KillContext(dur time.Duration) {
	TopLevelContext()
	go func() {
		time.Sleep(dur)
		cancel()
		assert.Never("context should never be killed with KillContext")
	}()
}

func CreateLogger(name string) *slog.Logger {
	logger := prettylog.CreateLoggerFromEnv(nil)
	logger = logger.With("area", name).With("process", "sim")
	slog.SetDefault(logger)

	logger.Error("Test Logger Created")

	return logger
}
