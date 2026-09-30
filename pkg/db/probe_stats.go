package db

import (
	"context"

	"github.com/ChevalRouting/routier/pkg/db/generated"
	"github.com/ChevalRouting/routier/pkg/types"
)

func LastProbeTS(ctx context.Context, db *DB, name string) int64 {
	ts, _ := db.queries.LastProbeTS(ctx, name)
	return ts
}

func InsertProbeStat(ctx context.Context, db *DB, point types.ProbeHistoryPoint) error {
	return db.queries.InsertProbeStat(ctx, generated.InsertProbeStatParams{
		Ts: point.TS, Name: point.Name, Target: point.Target,
		Reachable: boolInt(point.Reachable), RttAvgMs: point.RTTAvgMS,
	})
}

func ProbeHistory(ctx context.Context, db *DB, cutoff, bucket int64) map[string][]types.ProbeHistoryPoint {
	out := map[string][]types.ProbeHistoryPoint{}
	if bucket < 1 {
		bucket = 1
	}
	rows, err := db.queries.ProbeHistory(ctx, generated.ProbeHistoryParams{Cutoff: cutoff, Bucket: bucket})
	if err != nil {
		return out
	}

	for _, row := range rows {
		out[row.Name] = append(out[row.Name], types.ProbeHistoryPoint{
			TS: row.Ts, Name: row.Name, Target: row.Target,
			Reachable: row.Reachable != 0, RTTAvgMS: row.RttAvgMs,
		})
	}

	return out
}
