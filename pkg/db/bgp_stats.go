package db

import (
	"context"

	"github.com/ChevalRouting/routier/pkg/db/generated"
	"github.com/ChevalRouting/routier/pkg/types"
)

func InsertBGPStats(ctx context.Context, db *DB, ts int64, b *types.BGPStats) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	queries := db.queries.WithTx(tx)
	for _, p := range b.Peers {
		if err := queries.InsertBGPStat(ctx, generated.InsertBGPStatParams{
			Ts: ts, Peer: p.Address, RemoteAsn: int64(p.ASN), State: p.State, Uptime: p.Uptime,
			MsgRcvd: int64(p.MsgRcvd), MsgSent: int64(p.MsgSent), Prefixes: int64(p.Prefixes),
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func BGPHistory(ctx context.Context, db *DB, cutoff, bucket int64) map[string][]types.BGPHistoryPoint {
	out := make(map[string][]types.BGPHistoryPoint)
	if bucket < 1 {
		bucket = 1
	}

	rows, err := db.queries.BGPHistory(ctx, generated.BGPHistoryParams{Cutoff: cutoff, Bucket: bucket})
	if err != nil {
		return out
	}

	for _, row := range rows {
		out[row.Peer] = append(out[row.Peer], types.BGPHistoryPoint{
			TS: row.Ts, State: row.State, Uptime: row.Uptime,
			MsgRcvd: row.MsgRcvd, MsgSent: row.MsgSent, Prefixes: row.Prefixes,
		})
	}

	return out
}
