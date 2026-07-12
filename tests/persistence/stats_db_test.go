package persistencetest

import (
	"path/filepath"
	"testing"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
)

func TestStatsInsertAndHistory(t *testing.T) {
	db, err := webdb.InitDB(filepath.Join(t.TempDir(), "web.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}

	defer db.Close()

	if err := webdb.InsertSystemStats(db, 1000, &types.SystemStats{CPUPercent: 12.5, MemUsed: 100, MemTotal: 200, Load1: 0.5}); err != nil {
		t.Fatalf("insert system: %v", err)
	}

	if err := webdb.InsertSystemStats(db, 2000, &types.SystemStats{CPUPercent: 20}); err != nil {
		t.Fatalf("insert system 2: %v", err)
	}

	if ts := webdb.LastTS(db, "system_stats"); ts != 2000 {
		t.Fatalf("LastTS = %d, want 2000", ts)
	}

	hist := webdb.SystemHistory(db, 0, 1)
	if len(hist) != 2 || hist[0].CPUPct != 12.5 {
		t.Fatalf("system history = %+v", hist)
	}

	webdb.PruneStats(db, 1500)
	if hist := webdb.SystemHistory(db, 0, 1); len(hist) != 1 || hist[0].TS != 2000 {
		t.Fatalf("after prune = %+v", hist)
	}
}

func TestSystemHistoryBucketAveragesMem(t *testing.T) {
	db, err := webdb.InitDB(filepath.Join(t.TempDir(), "web.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}

	defer db.Close()

	mem := []uint64{100, 101, 100}
	for i, m := range mem {
		ts := int64(1000 + i)
		if err := webdb.InsertSystemStats(db, ts, &types.SystemStats{CPUPercent: 10, MemUsed: m, MemTotal: 200, Load1: 1}); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	hist := webdb.SystemHistory(db, 0, 100)
	if len(hist) != 1 {
		t.Fatalf("expected 1 bucketed point, got %d: %+v", len(hist), hist)
	}

	if hist[0].MemUsed != 100 {
		t.Fatalf("MemUsed = %d, want 100 (truncated avg)", hist[0].MemUsed)
	}
}

func TestStatsIfaceHistoryAndTotals(t *testing.T) {
	db, err := webdb.InitDB(filepath.Join(t.TempDir(), "web.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}

	defer db.Close()

	rxbps := 100.0
	txbps := 50.0
	rows := []webdb.IfaceStatRow{
		{Iface: "wan", RxBytes: 1000, TxBytes: 500, RxBps: &rxbps, TxBps: &txbps, OperState: "up"},
		{Iface: "lan", RxBytes: 2000, TxBytes: 800, OperState: "up"},
	}
	if err := webdb.InsertIfaceStats(db, 1000, rows); err != nil {
		t.Fatalf("insert iface: %v", err)
	}

	hist := webdb.IfaceHistory(db, 0, 1, "")
	if len(hist["wan"]) != 1 || hist["wan"][0].RxBytes != 1000 {
		t.Fatalf("wan history = %+v", hist["wan"])
	}

	if filtered := webdb.IfaceHistory(db, 0, 1, "lan"); len(filtered) != 1 || len(filtered["lan"]) != 1 {
		t.Fatalf("filtered history = %+v", filtered)
	}

	totals := webdb.IfaceTotals(db, 0, 1)
	if len(totals) != 1 {
		t.Fatalf("totals = %+v", totals)
	}
}

func TestReplaceKernelRoutes(t *testing.T) {
	db, err := webdb.InitDB(filepath.Join(t.TempDir(), "web.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}

	defer db.Close()

	first := []types.KernelRoute{{Dst: "10.0.0.0/24", Dev: "eth0", Protocol: "kernel", Family: "ipv4"}}
	if err := webdb.ReplaceKernelRoutes(db, first); err != nil {
		t.Fatalf("replace 1: %v", err)
	}

	if _, total, _ := webdb.QueryKernelRoutes(db, webdb.Filter{}, 0, 100); total != 1 {
		t.Fatalf("after first replace total=%d", total)
	}

	second := []types.KernelRoute{
		{Dst: "0.0.0.0/0", Gateway: "10.0.0.1", Dev: "eth0", Protocol: "static", Family: "ipv4"},
		{Dst: "::/0", Dev: "eth0", Protocol: "ra", Family: "ipv6"},
	}
	if err := webdb.ReplaceKernelRoutes(db, second); err != nil {
		t.Fatalf("replace 2: %v", err)
	}

	if _, total, _ := webdb.QueryKernelRoutes(db, webdb.Filter{}, 0, 100); total != 2 {
		t.Fatalf("after second replace total=%d, want 2", total)
	}
}
