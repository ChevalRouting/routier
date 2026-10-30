package switchcache

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/switches"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

const readTimeout = 20 * time.Second

type Entry struct {
	State   *types.SwitchState
	Error   string
	ReadAt  int64
	started time.Time
}

var (
	mu          sync.RWMutex
	entries     = map[string]Entry{}
	artifactDir = render.SwitchArtifactDir
)

func Get(name string) (Entry, bool) {
	mu.RLock()
	defer mu.RUnlock()

	entry, ok := entries[name]
	return entry, ok
}

func store(name string, entry Entry) {
	mu.Lock()
	defer mu.Unlock()

	entries[name] = entry
}

func forgetMissing(cfg *config.Config) {
	mu.Lock()
	defer mu.Unlock()

	for name := range entries {
		if _, ok := cfg.Switches[name]; !ok {
			delete(entries, name)
		}
	}
}

func Refresh(ctx context.Context, engine *switches.Engine, name string, sw *config.Switch) Entry {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	started := time.Now()
	entry := Entry{ReadAt: started.Unix(), started: started}
	state, err := engine.State(ctx, name, sw)
	if err != nil {
		entry.Error = err.Error()
		if previous, ok := Get(name); ok {
			entry.State = previous.State
		}
	} else {
		entry.State = &state
	}

	store(name, entry)
	return entry
}

func Current(ctx context.Context, engine *switches.Engine, name string, sw *config.Switch) (Entry, bool) {
	entry, ok := Get(name)
	if ok && !appliedSince(name, entry.started) {
		return entry, true
	}

	if !ok && !appliedSince(name, time.Time{}) {
		return Entry{}, false
	}

	return Refresh(ctx, engine, name, sw), true
}

func appliedSince(name string, read time.Time) bool {
	info, err := os.Stat(artifactDir + name + ".json")
	if err != nil {
		return false
	}

	return info.ModTime().After(read)
}

func Poll(ctx context.Context, engine *switches.Engine, configPath string) {
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Debug().Err(err).Msg("switch poll: load config")
		return
	}

	forgetMissing(cfg)
	var wg sync.WaitGroup
	for name, sw := range cfg.Switches {
		wg.Go(func() { Refresh(ctx, engine, name, sw) })
	}

	wg.Wait()
}
