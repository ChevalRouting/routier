package workers

import (
	"sync"
	"time"

	"github.com/ChevalRouting/routier/pkg/telemetry/stats"
	"github.com/ChevalRouting/routier/pkg/types"
)

type liveStore struct {
	mu    sync.RWMutex
	sys   *types.SystemStats
	procs []types.ProcessInfo
	bgp   *types.BGPStats
	ospf  *types.OSPFStats
}

var live liveStore

func StartLiveStats() {
	update := func() {
		sys, procs, bgp, ospf := stats.Sample()
		live.mu.Lock()
		live.sys = sys
		live.procs = procs
		live.bgp = bgp
		live.ospf = ospf
		live.mu.Unlock()
	}

	go func() {
		update()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			update()
		}
	}()
}

type LiveStats struct {
	Sys   *types.SystemStats
	Procs []types.ProcessInfo
	BGP   *types.BGPStats
	OSPF  *types.OSPFStats
}

func LiveSnapshot() LiveStats {
	live.mu.RLock()
	defer live.mu.RUnlock()
	return LiveStats{Sys: live.sys, Procs: live.procs, BGP: live.bgp, OSPF: live.ospf}
}
