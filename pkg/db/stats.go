package db

import (
	"database/sql"
	"time"
)

const StatsRetention = 30 * 24 * time.Hour

var statsTables = []string{"iface_stats", "bgp_peer_stats", "proto_stats", "system_stats", "neighbor_stats"}

func LastTS(db *sql.DB, table string) int64 {
	var ts int64
	_ = db.QueryRow("SELECT COALESCE(MAX(ts), 0) FROM " + table).Scan(&ts)
	return ts
}

func PruneOldStats(db *sql.DB) {
	PruneStats(db, time.Now().Add(-StatsRetention).Unix())
}

func PruneStats(db *sql.DB, cutoff int64) {
	for _, table := range statsTables {
		_, _ = db.Exec("DELETE FROM "+table+" WHERE ts < ?", cutoff)
	}

	_, _ = db.Exec("PRAGMA incremental_vacuum")
}

func HistoryBucket(windowSec int64, maxPoints int) int64 {
	if windowSec <= 0 || maxPoints <= 0 {
		return 1
	}

	bucket := windowSec / int64(maxPoints)
	if bucket < 1 {
		return 1
	}

	return bucket
}
