package db

import (
	"context"

	"github.com/ChevalRouting/routier/pkg/db/generated"
	"github.com/ChevalRouting/routier/pkg/types"
)

func InsertSystemStats(ctx context.Context, db *DB, ts int64, s *types.SystemStats) error {
	memUsed, err := sqliteInteger("mem_used", s.MemUsed)
	if err != nil {
		return err
	}

	memTotal, err := sqliteInteger("mem_total", s.MemTotal)
	if err != nil {
		return err
	}

	return db.queries.InsertSystemStat(ctx, generated.InsertSystemStatParams{
		Ts: ts, CpuPct: s.CPUPercent, MemUsed: memUsed, MemTotal: memTotal, Load1: s.Load1,
	})
}

func SystemHistory(ctx context.Context, db *DB, cutoff, bucket int64) []types.SystemHistoryPoint {
	var out []types.SystemHistoryPoint
	if bucket < 1 {
		bucket = 1
	}

	rows, err := db.queries.SystemHistory(ctx, generated.SystemHistoryParams{Cutoff: cutoff, Bucket: bucket})
	if err != nil {
		return out
	}

	for _, row := range rows {
		out = append(out, types.SystemHistoryPoint{
			TS: row.Ts, CPUPct: floatValue(row.CpuPct), MemUsed: row.MemUsed,
			MemTotal: row.MemTotal, Load1: floatValue(row.Load1),
		})
	}

	return out
}
