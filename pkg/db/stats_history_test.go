package db

import (
	"errors"
	"math"
	"path/filepath"
	"testing"

	"github.com/ChevalRouting/routier/pkg/types"
)

func testDB(t *testing.T) *DB {
	t.Helper()

	db, err := InitDB(t.Context(), filepath.Join(t.TempDir(), "web.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

func f64(v float64) *float64 {
	return &v
}

func TestIfaceHistoryUsesLatestRowPerBucket(t *testing.T) {
	db := testDB(t)
	ctx := t.Context()

	if err := InsertIfaceStats(ctx, db, 1000, []IfaceStatRow{
		{Iface: "eth0", RxBytes: 5000, OperState: "up", RxBytesPS: f64(10)},
	}); err != nil {
		t.Fatal(err)
	}

	if err := InsertIfaceStats(ctx, db, 1050, []IfaceStatRow{
		{Iface: "eth0", RxBytes: 100, OperState: "down", RxBytesPS: f64(20)},
	}); err != nil {
		t.Fatal(err)
	}

	points := IfaceHistory(ctx, db, 0, 100, "")["eth0"]
	if len(points) != 1 {
		t.Fatalf("points = %d, want 1 bucket", len(points))
	}

	p := points[0]
	if p.TS != 1050 {
		t.Errorf("ts = %d, want 1050 (latest)", p.TS)
	}

	if p.RxBytes != 100 {
		t.Errorf("rx_bytes = %d, want 100 (latest row after reset, not MAX)", p.RxBytes)
	}

	if p.OperState != "down" {
		t.Errorf("operstate = %q, want down (latest row, not lexicographic MAX)", p.OperState)
	}

	if p.RxBytesPS == nil || *p.RxBytesPS != 15 {
		t.Errorf("rx_bps = %v, want 15 (avg of 10 and 20)", p.RxBytesPS)
	}
}

func TestBGPHistoryUsesLatestRowPerBucket(t *testing.T) {
	db := testDB(t)
	ctx := t.Context()

	insert := func(ts int64, state string, msgRcvd int) {
		if err := InsertBGPStats(ctx, db, ts, &types.BGPStats{Peers: []types.BGPPeerSummary{
			{Address: "10.0.0.1", State: state, MsgRcvd: msgRcvd},
		}}); err != nil {
			t.Fatal(err)
		}
	}

	insert(1000, "Idle", 500)
	insert(1050, "Established", 20)

	points := BGPHistory(ctx, db, 0, 100)["10.0.0.1"]
	if len(points) != 1 {
		t.Fatalf("points = %d, want 1 bucket", len(points))
	}

	if points[0].State != "Established" {
		t.Errorf("state = %q, want Established (latest, not lexicographic MAX)", points[0].State)
	}

	if points[0].MsgRcvd != 20 {
		t.Errorf("msg_rcvd = %d, want 20 (latest after reset, not MAX)", points[0].MsgRcvd)
	}
}

func TestInsertIfaceStatsRollsBackOnRowError(t *testing.T) {
	db := testDB(t)
	ctx := t.Context()

	err := InsertIfaceStats(ctx, db, 1000, []IfaceStatRow{
		{Iface: "eth0", RxBytes: 100},
		{Iface: "eth1", RxBytes: math.MaxUint64},
	})
	if err == nil {
		t.Fatal("expected error from out-of-range counter")
	}

	if counters := LastIfaceCounters(ctx, db); len(counters) != 0 {
		t.Fatalf("rows persisted after failed batch: %d, want 0 (rolled back)", len(counters))
	}
}

func TestAnnouncementUpdateDeleteReportMissing(t *testing.T) {
	db := testDB(t)
	ctx := t.Context()

	missing := &types.Announcement{ID: 999, Message: "x", Level: "info"}
	if err := UpdateAnnouncement(ctx, db, missing); !errors.Is(err, ErrAnnouncementNotFound) {
		t.Errorf("update missing: err = %v, want ErrAnnouncementNotFound", err)
	}

	if err := DeleteAnnouncement(ctx, db, 999); !errors.Is(err, ErrAnnouncementNotFound) {
		t.Errorf("delete missing: err = %v, want ErrAnnouncementNotFound", err)
	}

	id, err := CreateAnnouncement(ctx, db, &types.Announcement{Message: "hi", Level: "info"})
	if err != nil {
		t.Fatal(err)
	}

	existing := &types.Announcement{ID: id, Message: "updated", Level: "info"}
	if err := UpdateAnnouncement(ctx, db, existing); err != nil {
		t.Errorf("update existing: %v", err)
	}

	if err := DeleteAnnouncement(ctx, db, id); err != nil {
		t.Errorf("delete existing: %v", err)
	}
}
