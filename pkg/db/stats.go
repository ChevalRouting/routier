package db

import (
	"context"
	"time"
)

const StatsRetention = 30 * 24 * time.Hour

func LastTS(ctx context.Context, db *DB, table string) int64 {
	queries := db.queries
	var ts int64
	switch table {
	case "iface_stats":
		ts, _ = queries.LastIfaceTS(ctx)
	case "bgp_peer_stats":
		ts, _ = queries.LastBGPTS(ctx)
	case "proto_stats":
		ts, _ = queries.LastProtoTS(ctx)
	case "system_stats":
		ts, _ = queries.LastSystemTS(ctx)
	case "neighbor_stats":
		ts, _ = queries.LastNeighborTS(ctx)
	case "lldp_neighbors":
		ts, _ = queries.LastLLDPTS(ctx)
	case "probe_stats":
		ts, _ = queries.LastAnyProbeTS(ctx)
	}

	return ts
}

func PruneOldStats(ctx context.Context, db *DB) error {
	return PruneStats(ctx, db, time.Now().Add(-StatsRetention).Unix())
}

func PruneStats(ctx context.Context, db *DB, cutoff int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	queries := db.queries.WithTx(tx)
	if err := queries.PruneIfaceStats(ctx, cutoff); err != nil {
		return err
	}

	if err := queries.PruneBGPStats(ctx, cutoff); err != nil {
		return err
	}

	if err := queries.PruneProtoStats(ctx, cutoff); err != nil {
		return err
	}

	if err := queries.PruneSystemStats(ctx, cutoff); err != nil {
		return err
	}

	if err := queries.PruneNeighborStats(ctx, cutoff); err != nil {
		return err
	}

	if err := queries.PruneLLDPNeighbors(ctx, cutoff); err != nil {
		return err
	}

	if err := queries.PruneProbeStats(ctx, cutoff); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, "PRAGMA incremental_vacuum")
	return err
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
