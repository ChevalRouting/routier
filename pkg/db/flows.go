package db

import (
	"context"
	"encoding/json"

	"github.com/ChevalRouting/routier/pkg/db/generated"
	"github.com/ChevalRouting/routier/pkg/types"
)

func InsertFlows(ctx context.Context, db *DB, flows []types.Flow) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	queries := db.queries.WithTx(tx)
	for _, f := range flows {
		if err := queries.InsertFlow(ctx, generated.InsertFlowParams{
			StartTs: f.Start, EndTs: f.End, InIf: f.InIf, OutIf: f.OutIf,
			Family: int64(f.Family), Proto: int64(f.Proto), Src: f.Src, Dst: f.Dst,
			SrcPort: int64(f.SrcPort), DstPort: int64(f.DstPort),
			Packets: f.Packets, Bytes: f.Bytes, Samples: f.Samples,
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func PruneFlows(ctx context.Context, db *DB, before int64, keep int) error {
	if err := db.queries.DeleteFlowsEndedBefore(ctx, before); err != nil {
		return err
	}

	if err := db.queries.DeleteFlowNamesResolvedBefore(ctx, before); err != nil {
		return err
	}

	return db.queries.TrimFlowsToNewest(ctx, int64(keep))
}

func CountFlows(ctx context.Context, db *DB) (int64, error) {
	return db.queries.CountFlows(ctx)
}

func RecentFlows(ctx context.Context, db *DB, since int64, address string, limit int) ([]types.Flow, error) {
	rows, err := db.queries.RecentFlows(ctx, generated.RecentFlowsParams{Since: since, Address: address, RowLimit: int64(limit)})
	if err != nil {
		return nil, err
	}

	flows := make([]types.Flow, 0, len(rows))
	for _, r := range rows {
		flows = append(flows, types.Flow{
			ID: r.ID, Start: r.StartTs, End: r.EndTs, InIf: r.InIf, OutIf: r.OutIf,
			Family: int(r.Family), Proto: int(r.Proto), Src: r.Src, Dst: r.Dst,
			SrcPort: int(r.SrcPort), DstPort: int(r.DstPort),
			Packets: r.Packets, Bytes: r.Bytes, Samples: r.Samples,
		})
	}

	return flows, nil
}

func TopFlows(ctx context.Context, db *DB, dimension string, since int64, limit int) ([]types.FlowTopEntry, error) {
	rows, err := db.queries.TopFlows(ctx, generated.TopFlowsParams{Dimension: dimension, Since: since, RowLimit: int64(limit)})
	if err != nil {
		return nil, err
	}

	entries := make([]types.FlowTopEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, types.FlowTopEntry{Key: r.GroupKey, Bytes: r.Bytes, Packets: r.Packets, Flows: r.Flows})
	}

	return entries, nil
}

func FlowSeries(ctx context.Context, db *DB, since, bucket int64) ([]types.FlowSeriesPoint, error) {
	rows, err := db.queries.FlowSeries(ctx, generated.FlowSeriesParams{Since: since, Bucket: bucket})
	if err != nil {
		return nil, err
	}

	points := make([]types.FlowSeriesPoint, 0, len(rows))
	for _, r := range rows {
		points = append(points, types.FlowSeriesPoint{Ts: r.Ts, Iface: r.InIf, Bytes: r.Bytes, Packets: r.Packets})
	}

	return points, nil
}

func FlowPaths(ctx context.Context, db *DB, dimensions []string, since int64, limit int) ([]types.FlowPath, error) {
	var slots [3]string
	copy(slots[:], dimensions)
	rows, err := db.queries.FlowPaths(ctx, generated.FlowPathsParams{First: slots[0], Second: slots[1], Third: slots[2], Since: since, RowLimit: int64(limit)})
	if err != nil {
		return nil, err
	}

	paths := make([]types.FlowPath, 0, len(rows))
	for _, r := range rows {
		keys := []string{r.FirstKey, r.SecondKey, r.ThirdKey}[:len(dimensions)]
		paths = append(paths, types.FlowPath{Keys: keys, Bytes: r.Bytes, Packets: r.Packets})
	}

	return paths, nil
}

func FlowBytesSince(ctx context.Context, db *DB, since int64) (int64, error) {
	return db.queries.FlowBytesSince(ctx, since)
}

func SaveFlowName(ctx context.Context, db *DB, address, name string, resolvedAt int64) error {
	return db.queries.UpsertFlowName(ctx, generated.UpsertFlowNameParams{Address: address, Name: name, ResolvedAt: resolvedAt})
}

func FlowNames(ctx context.Context, db *DB, addresses []string) (map[string]string, error) {
	names := map[string]string{}
	if len(addresses) == 0 {
		return names, nil
	}

	encoded, err := json.Marshal(addresses)
	if err != nil {
		return nil, err
	}

	rows, err := db.queries.FlowNames(ctx, string(encoded))
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		names[r.Address] = r.Name
	}

	return names, nil
}
