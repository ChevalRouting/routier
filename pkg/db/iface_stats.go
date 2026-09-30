package db

import (
	"context"
	"encoding/json"

	"github.com/ChevalRouting/routier/pkg/db/generated"
	"github.com/ChevalRouting/routier/pkg/types"
)

type IfaceCounter struct {
	TS, RxBytes, TxBytes, RxPkts, TxPkts int64
}

func LastIfaceCounters(ctx context.Context, db *DB) map[string]IfaceCounter {
	out := map[string]IfaceCounter{}
	rows, err := db.queries.LastIfaceCounters(ctx)
	if err != nil {
		return out
	}

	for _, row := range rows {
		out[row.Iface] = IfaceCounter{
			TS: row.Ts, RxBytes: row.RxBytes, TxBytes: row.TxBytes, RxPkts: row.RxPkts, TxPkts: row.TxPkts,
		}
	}

	return out
}

type IfaceStatRow struct {
	Iface                                            string
	RxBytes, TxBytes, RxPkts, TxPkts, RxErrs, TxErrs uint64
	RxBytesPS, TxBytesPS, RxPps, TxPps               *float64
	OperState                                        string
}

func InsertIfaceStats(ctx context.Context, db *DB, ts int64, rows []IfaceStatRow) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	queries := db.queries.WithTx(tx)
	for _, r := range rows {
		rxBytes, err := sqliteInteger("rx_bytes", r.RxBytes)
		if err != nil {
			return err
		}

		txBytes, err := sqliteInteger("tx_bytes", r.TxBytes)
		if err != nil {
			return err
		}

		rxPkts, err := sqliteInteger("rx_pkts", r.RxPkts)
		if err != nil {
			return err
		}

		txPkts, err := sqliteInteger("tx_pkts", r.TxPkts)
		if err != nil {
			return err
		}

		rxErrs, err := sqliteInteger("rx_errs", r.RxErrs)
		if err != nil {
			return err
		}

		txErrs, err := sqliteInteger("tx_errs", r.TxErrs)
		if err != nil {
			return err
		}

		if err := queries.InsertIfaceStat(ctx, generated.InsertIfaceStatParams{
			Ts: ts, Iface: r.Iface, RxBytes: rxBytes, TxBytes: txBytes,
			RxPkts: rxPkts, TxPkts: txPkts, RxErrs: rxErrs, TxErrs: txErrs,
			RxBps: r.RxBytesPS, TxBps: r.TxBytesPS,
			RxPps: r.RxPps, TxPps: r.TxPps, Operstate: r.OperState,
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func IfaceHistory(ctx context.Context, db *DB, cutoff, bucket int64, ifaceFilter string) map[string][]types.IfaceHistoryPoint {
	out := make(map[string][]types.IfaceHistoryPoint)
	if bucket < 1 {
		bucket = 1
	}

	rows, err := db.queries.IfaceHistory(ctx, generated.IfaceHistoryParams{
		Cutoff: cutoff, Bucket: bucket, Iface: ifaceFilter,
	})
	if err != nil {
		return out
	}

	for _, row := range rows {
		out[row.Iface] = append(out[row.Iface], types.IfaceHistoryPoint{
			TS: row.Ts, RxBytes: row.RxBytes, TxBytes: row.TxBytes, RxPkts: row.RxPkts, TxPkts: row.TxPkts,
			RxErrs: row.RxErrs, TxErrs: row.TxErrs, RxBytesPS: row.RxBps,
			TxBytesPS: row.TxBps, RxPps: row.RxPps, TxPps: row.TxPps, OperState: row.Operstate,
		})
	}

	return out
}

func IfaceTotals(ctx context.Context, db *DB, cutoff, bucket int64, include map[string]bool) []types.IfaceTotalPoint {
	var out []types.IfaceTotalPoint
	if bucket < 1 {
		bucket = 1
	}

	ifaces := make([]string, 0, len(include))
	for name := range include {
		ifaces = append(ifaces, name)
	}

	encoded, err := json.Marshal(ifaces)
	if err != nil {
		return out
	}

	rows, err := db.queries.IfaceTotals(ctx, generated.IfaceTotalsParams{
		Cutoff: cutoff, Bucket: bucket, Ifaces: string(encoded),
	})
	if err != nil {
		return out
	}

	for _, row := range rows {
		out = append(out, types.IfaceTotalPoint{
			TS: row.Ts, RxBytesPS: row.Rx, TxBytesPS: row.Tx, RxPps: row.Rxp, TxPps: row.Txp,
		})
	}

	return out
}
