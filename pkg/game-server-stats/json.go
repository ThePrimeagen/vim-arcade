package gameserverstats

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"

	"vim-arcade.theprimeagen.com/pkg/assert"
)

type JSONMemoryFile struct {
    Stats []GameServerConfig `json:"stats"`
}

func toBytes(j *JSONMemory) ([]byte, error) {
    mem := JSONMemoryFile{Stats: j.stats}
    return json.Marshal(mem)
}

type JSONMemory struct {
    file string
    stats []GameServerConfig
    mutex sync.Mutex
    logger *slog.Logger
}

func getLogger() *slog.Logger {
    return slog.Default().With("area", "JSONMemory")
}

func JSONMemoryFrom(path string, data JSONMemoryFile) (*JSONMemory, error) {
    bytes, err := json.Marshal(data)
    if err != nil {
        return nil, err
    }

    err = os.WriteFile(path, bytes, 0644);
    if err != nil {
        return nil, err
    }

    return &JSONMemory{file: path}, nil
}

func NewJSONMemoryAndClear(path string) (*JSONMemory, error) {
    err := os.WriteFile(path, []byte(`{"stats": []}`), 0644);
    if err != nil {
        return nil, err
    }
    return &JSONMemory{file: path, logger: getLogger()}, nil
}

func NewJSONMemory(path string) (*JSONMemory, error) {
    if _, err := os.Stat(path); err != nil {
        return NewJSONMemoryAndClear(path)
    }
    return &JSONMemory{
        file: path,
        logger: getLogger(),
    }, nil
}

func (j *JSONMemory) GetServerCount() int {
    j.mutex.Lock()
    count := len(j.stats)
    j.mutex.Unlock()
    return count
}

func (j *JSONMemory) GetConnectionCount() int {
    count := 0
    for _, s := range j.Iter() {
        count += s.Connections
    }
    return count
}

func (j *JSONMemory) Update(stat GameServerConfig) error {
    update := false
    for i, s := range j.Iter() {
        if s.Id == stat.Id {
            j.stats[i] = stat
            update = true
        }
    }

    slog.Info("Update", "stat", stat, "update", update)
    if !update {
        j.stats = append(j.stats, stat)
    }
    j.save()
    return nil
}

func (j *JSONMemory) Run(ctx context.Context) {
    timer := time.NewTicker(time.Second)
    defer timer.Stop()
    outer:
    for {
        j.logger.Warn("Run#forLoop")
        select {
        case <-timer.C:
            j.refresh()

        case <-ctx.Done():
            j.logger.Warn("ctx done")
            break outer
        }
    }
}

func (j *JSONMemory) save() {
    data, err := toBytes(j)
    assert.NoError(err, "json storage serializing failed", "err", err)
    err = os.WriteFile(j.file, data, 0644)
    assert.NoError(err, "json storage writing failed", "err", err)
}

func (j *JSONMemory) refresh() {
    contents, err := os.ReadFile(j.file)
    if err != nil {
        slog.Error("unable to read json file", "error", err)
        return;
    }

    var data JSONMemoryFile
    err = json.Unmarshal(contents, &data)
    if err != nil {
        slog.Error("unable to decode json file", "error", err)
        return;
    }

    j.logger.Info("refresh", "stats", data.Stats)

    j.mutex.Lock()
    j.stats = data.Stats
    j.mutex.Unlock()
}

func (j *JSONMemory) Iter() func(yield func(i int, s GameServerConfig) bool) {
	return func(yield func(i int, s GameServerConfig) bool) {
        j.mutex.Lock()
        defer j.mutex.Unlock()
		for i, s := range j.stats {
			if !yield(i, s) {
				return
			}
		}
	}
}

func (j *JSONMemory) GetById(id string) *GameServerConfig {
    var g *GameServerConfig
    for _, gs := range j.Iter() {
        if gs.Id == id {
            g = &gs
        }
    }
    slog.Info("GetById", "id", id, "stat", g)
    return g
}
