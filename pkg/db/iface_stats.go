package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/ChevalRouting/routier/pkg/types"
)

func ensureIfaceStatsSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS iface_stats (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		ts        INTEGER NOT NULL,
		iface     TEXT    NOT NULL,
		rx_bytes  INTEGER NOT NULL DEFAULT 0,
		tx_bytes  INTEGER NOT NULL DEFAULT 0,
		rx_pkts   INTEGER NOT NULL DEFAULT 0,
		tx_pkts   INTEGER NOT NULL DEFAULT 0,
		rx_errs   INTEGER NOT NULL DEFAULT 0,
		tx_errs   INTEGER NOT NULL DEFAULT 0,
		rx_bps    REAL,
		tx_bps    REAL,
		rx_pps    REAL,
		tx_pps    REAL,
		operstate TEXT    NOT NULL DEFAULT ''
	)`)
	if err != nil {
		return err
	}

	_, _ = db.Exec(`ALTER TABLE iface_stats ADD COLUMN rx_pps REAL`)
	_, _ = db.Exec(`ALTER TABLE iface_stats ADD COLUMN tx_pps REAL`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_iface_stats ON iface_stats(iface, ts DESC)`)

	return nil
}

type IfaceCounter struct {
	TS, RxBytes, TxBytes, RxPkts, TxPkts int64
}

func LastIfaceCounters(db *sql.DB) map[string]IfaceCounter {
	out := map[string]IfaceCounter{}
	rows, err := db.Query(
		`SELECT iface, ts, rx_bytes, tx_bytes, rx_pkts, tx_pkts FROM iface_stats
		 WHERE id IN (SELECT MAX(id) FROM iface_stats GROUP BY iface)`)
	if err != nil {
		return out
	}

	defer rows.Close()

	for rows.Next() {
		var name string
		var c IfaceCounter

		if rows.Scan(&name, &c.TS, &c.RxBytes, &c.TxBytes, &c.RxPkts, &c.TxPkts) == nil {
			out[name] = c
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

func InsertIfaceStats(db *sql.DB, ts int64, rows []IfaceStatRow) error {
	stmt, err := db.Prepare(
		`INSERT INTO iface_stats
		 (ts, iface, rx_bytes, tx_bytes, rx_pkts, tx_pkts, rx_errs, tx_errs, rx_bps, tx_bps, rx_pps, tx_pps, operstate)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}

	defer stmt.Close()

	for _, r := range rows {
		if _, err := stmt.Exec(ts, r.Iface, r.RxBytes, r.TxBytes, r.RxPkts, r.TxPkts, r.RxErrs, r.TxErrs,
			r.RxBytesPS, r.TxBytesPS, r.RxPps, r.TxPps, r.OperState); err != nil {
			return err
		}
	}

	return nil
}

func IfaceHistory(db *sql.DB, cutoff, bucket int64, ifaceFilter string) map[string][]types.IfaceHistoryPoint {
	out := make(map[string][]types.IfaceHistoryPoint)
	if bucket < 1 {
		bucket = 1
	}

	where := "WHERE ts >= ?"
	args := []any{cutoff}
	if ifaceFilter != "" {
		where += " AND iface = ?"
		args = append(args, ifaceFilter)
	}

	q := fmt.Sprintf(
		`SELECT MAX(ts) AS ts, iface,
		        MAX(rx_bytes), MAX(tx_bytes), MAX(rx_pkts), MAX(tx_pkts), MAX(rx_errs), MAX(tx_errs),
		        AVG(rx_bps), AVG(tx_bps), AVG(rx_pps), AVG(tx_pps), MAX(operstate)
		 FROM iface_stats %s
		 GROUP BY iface, ts/%d
		 ORDER BY iface, ts ASC`, where, bucket)

	rows, err := db.Query(q, args...)
	if err != nil {
		return out
	}

	defer rows.Close()

	for rows.Next() {
		var p types.IfaceHistoryPoint
		var iface string
		if rows.Scan(&p.TS, &iface, &p.RxBytes, &p.TxBytes, &p.RxPkts, &p.TxPkts,
			&p.RxErrs, &p.TxErrs, &p.RxBytesPS, &p.TxBytesPS, &p.RxPps, &p.TxPps, &p.OperState) == nil {
			out[iface] = append(out[iface], p)
		}
	}

	return out
}

func IfaceTotals(db *sql.DB, cutoff, bucket int64, include map[string]bool) []types.IfaceTotalPoint {
	var out []types.IfaceTotalPoint
	if bucket < 1 {
		bucket = 1
	}

	where := "WHERE ts >= ?"
	args := []any{cutoff}
	if len(include) > 0 {
		placeholders := make([]string, 0, len(include))
		for name := range include {
			placeholders = append(placeholders, "?")
			args = append(args, name)
		}

		where += " AND iface IN (" + strings.Join(placeholders, ",") + ")"
	}

	q := fmt.Sprintf(
		`SELECT MAX(ts) AS ts, AVG(rx), AVG(tx), AVG(rxp), AVG(txp) FROM (
		   SELECT ts, SUM(rx_bps) rx, SUM(tx_bps) tx, SUM(rx_pps) rxp, SUM(tx_pps) txp
		   FROM iface_stats %s GROUP BY ts
		 ) GROUP BY ts/%d ORDER BY ts ASC`, where, bucket)

	rows, err := db.Query(q, args...)
	if err != nil {
		return out
	}

	defer rows.Close()

	for rows.Next() {
		var p types.IfaceTotalPoint
		if rows.Scan(&p.TS, &p.RxBytesPS, &p.TxBytesPS, &p.RxPps, &p.TxPps) == nil {
			out = append(out, p)
		}
	}

	return out
}
