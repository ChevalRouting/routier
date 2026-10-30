package workers

import (
	"context"
	"sync"
	"time"

	"github.com/ChevalRouting/routier/pkg/telemetry/stats"
	"github.com/ChevalRouting/routier/pkg/types"
)

const liveStatsInterval = 5 * time.Second

type liveStore struct {
	mu    sync.RWMutex
	sys   *types.SystemStats
	procs []types.ProcessInfo
	bgp   *types.BGPStats
	ospf  *types.OSPFStats
}

var live liveStore

type LiveStats struct {
	Sys   *types.SystemStats
	Procs []types.ProcessInfo
	BGP   *types.BGPStats
	OSPF  *types.OSPFStats
}

func StartLiveStats(ctx context.Context, g *Group) {
	g.Go(func() { every(ctx, liveStatsPeriod, sampleLiveStats) })
}

func liveStatsPeriod() time.Duration { return liveStatsInterval }

func LiveSnapshot() LiveStats {
	live.mu.RLock()
	defer live.mu.RUnlock()
	return LiveStats{Sys: live.sys, Procs: live.procs, BGP: live.bgp, OSPF: live.ospf}
}

func sampleLiveStats() {
	sys, procs, bgp, ospf := stats.Sample()
	live.mu.Lock()
	live.sys = sys
	live.procs = procs
	live.bgp = bgp
	live.ospf = ospf
	live.mu.Unlock()
}
