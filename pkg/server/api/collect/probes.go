package collect

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/net/probe"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/types"
)

func RunProbes(ctx context.Context, dbPath, configPath string) error {
	database, err := webdb.InitDB(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer func(action func() error) { _ = action() }(database.Close)
	cfg, err := cfgstore.Read(configPath, "")
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	if cfg.Monitoring == nil {
		return nil
	}

	now := time.Now().Unix()
	due := make([]config.PingProbe, 0, len(cfg.Monitoring.Probes))
	for _, configured := range cfg.Monitoring.Probes {
		interval := configured.Interval
		if interval < 60 {
			interval = 60
		}

		if now-webdb.LastProbeTS(ctx, database, configured.Name) < int64(interval) {
			continue
		}

		due = append(due, configured)
	}

	results := make(chan types.ProbeHistoryPoint, len(due))
	var group sync.WaitGroup
	for _, configured := range due {
		group.Add(1)
		go func() { runProbesCallback(now, results, &group, &configured) }()
	}

	group.Wait()
	close(results)

	for point := range results {
		if err := webdb.InsertProbeStat(ctx, database, point); err != nil {
			return fmt.Errorf("store probe %s: %w", point.Name, err)
		}
	}

	return nil
}

func runProbesCallback(now int64, results chan types.ProbeHistoryPoint, group *sync.WaitGroup, configured *config.PingProbe) {
	defer (*group).Done()
	point := types.ProbeHistoryPoint{TS: now, Name: (*configured).Name, Target: (*configured).Target}
	timeout := time.Duration((*configured).Timeout) * time.Millisecond
	if average, pingErr := probe.Ping((*configured).Target, timeout); pingErr == nil {
		point.Reachable = true
		point.RTTAvgMS = &average
	}

	results <- point
}
