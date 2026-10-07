package db

import (
	"context"

	"github.com/ChevalRouting/routier/pkg/db/generated"
	"github.com/ChevalRouting/routier/pkg/types"
)

type NeighborStatRow struct {
	IP, MAC, Dev, State, Family, Hostname string
}

func InsertNeighborStats(ctx context.Context, db *DB, ts int64, rows []NeighborStatRow) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func(action func() error) { _ = action() }(tx.Rollback)

	queries := db.queries.WithTx(tx)
	for _, r := range rows {
		if err := queries.InsertNeighborStat(ctx, generated.InsertNeighborStatParams{
			Ts: ts, Ip: r.IP, Mac: r.MAC, Dev: r.Dev, State: r.State, Family: r.Family, Hostname: r.Hostname,
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func LatestNeighbors(ctx context.Context, db *DB) (int64, []types.NeighborStat) {
	queries := db.queries
	maxTS, err := queries.LastNeighborTS(ctx)
	if err != nil || maxTS == 0 {
		return 0, nil
	}

	rows, err := queries.LatestNeighbors(ctx, maxTS)
	if err != nil {
		return maxTS, nil
	}

	neighbors := make([]types.NeighborStat, 0, len(rows))
	for _, row := range rows {
		neighbors = append(neighbors, types.NeighborStat{
			IP: row.Ip, MAC: row.Mac, Dev: row.Dev, State: row.State, Family: row.Family, Hostname: row.Hostname,
		})
	}

	return maxTS, neighbors
}
