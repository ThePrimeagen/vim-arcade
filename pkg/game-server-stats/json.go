package gameserverstats

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"
)

type JSONMemoryFile struct {
    Stats []GameServerConfig `json:"stats"`
}

type JSONMemory struct {
    file string
    stats []GameServerConfig
    mutex sync.Mutex
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
    return &JSONMemory{file: path}, nil
}

func NewJSONMemory(path string) JSONMemory {
    return JSONMemory{file: path}
}

func (j *JSONMemory) Update(stat GameServerConfig) error {
    update := false
    for i, s := range j.Iter() {
        if s.Id == stat.Id {
            j.stats[i] = stat
            update = true
        }
    }

    if !update {
        j.stats = append(j.stats, stat)
    }
    return nil
}

func (j *JSONMemory) Run(ctx context.Context) {
    timer := time.NewTicker(time.Second)
    defer timer.Stop()
    outer:
    for {
        select {
        case <-timer.C:
            j.refresh()

        case <-ctx.Done():
            break outer
        }
    }
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
    for _, gs := range j.Iter() {
        if gs.Id == id {
            return &gs
        }
    }
    return nil
}
