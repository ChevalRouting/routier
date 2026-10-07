package db

import (
	"context"
	"encoding/json"
	"math"

	"github.com/ChevalRouting/routier/pkg/db/generated"
	"github.com/ChevalRouting/routier/pkg/types"
)

type Filter struct {
	Query       string
	Family      string
	Protocols   []string
	DefaultOnly bool
}

func ReplaceKernelRoutes(ctx context.Context, db *DB, routes []types.KernelRoute) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func(action func() error) { _ = action() }(tx.Rollback)

	queries := generated.New(tx)
	if err := queries.DeleteKernelRoutes(ctx); err != nil {
		return err
	}

	for _, rt := range routes {
		if err := queries.InsertKernelRoute(ctx, generated.InsertKernelRouteParams{
			Dst: rt.Dst, Gateway: rt.Gateway, Dev: rt.Dev, Protocol: rt.Protocol,
			Metric: int64(rt.Metric), Family: rt.Family,
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func QueryKernelRoutes(ctx context.Context, db *DB, f Filter, offset, limit int) ([]types.KernelRoute, int, error) {
	protocols, err := json.Marshal(f.Protocols)
	if err != nil {
		return nil, 0, err
	}

	queries := db.queries
	filter := generated.CountKernelRoutesParams{
		DefaultOnly: boolInt(f.DefaultOnly), Protocols: string(protocols), Family: f.Family, Query: f.Query,
	}
	total, err := queries.CountKernelRoutes(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if !f.DefaultOnly {
		if limit < 0 {
			limit = 0
		}
	} else {
		limit = math.MaxInt
		offset = 0
	}

	rows, err := queries.QueryKernelRoutes(ctx, generated.QueryKernelRoutesParams{
		DefaultOnly: filter.DefaultOnly, Protocols: filter.Protocols, Family: filter.Family, Query: filter.Query,
		OffsetCount: int64(offset), LimitCount: int64(limit),
	})
	if err != nil {
		return nil, 0, err
	}

	routes := make([]types.KernelRoute, 0, len(rows))
	for _, row := range rows {
		routes = append(routes, types.KernelRoute{
			Dst: row.Dst, Gateway: row.Gateway, Dev: row.Dev, Protocol: row.Protocol,
			Metric: int(row.Metric), Family: row.Family,
		})
	}

	return routes, int(total), nil
}
