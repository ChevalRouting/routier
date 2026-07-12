package db

import (
	"database/sql"
	"fmt"

	"github.com/ChevalRouting/routier/pkg/types"
)

func ensureSystemStatsSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS system_stats (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		ts         INTEGER NOT NULL,
		cpu_pct    REAL    NOT NULL DEFAULT 0,
		mem_used   INTEGER NOT NULL DEFAULT 0,
		mem_total  INTEGER NOT NULL DEFAULT 0,
		load1      REAL    NOT NULL DEFAULT 0
	)`)
	if err != nil {
		return err
	}

	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_system_stats ON system_stats(ts DESC)`)

	return nil
}

func InsertSystemStats(db *sql.DB, ts int64, s *types.SystemStats) error {
	_, err := db.Exec(
		`INSERT INTO system_stats (ts, cpu_pct, mem_used, mem_total, load1) VALUES (?, ?, ?, ?, ?)`,
		ts, s.CPUPercent, s.MemUsed, s.MemTotal, s.Load1,
	)
	return err
}

func SystemHistory(db *sql.DB, cutoff, bucket int64) []types.SystemHistoryPoint {
	var out []types.SystemHistoryPoint
	if bucket < 1 {
		bucket = 1
	}

	q := fmt.Sprintf(
		`SELECT MAX(ts) AS ts, AVG(cpu_pct), CAST(AVG(mem_used) AS INTEGER), MAX(mem_total), AVG(load1)
		 FROM system_stats WHERE ts >= ? GROUP BY ts/%d ORDER BY ts ASC`, bucket)

	rows, err := db.Query(q, cutoff)
	if err != nil {
		return out
	}

	defer rows.Close()

	for rows.Next() {
		var p types.SystemHistoryPoint
		if rows.Scan(&p.TS, &p.CPUPct, &p.MemUsed, &p.MemTotal, &p.Load1) == nil {
			out = append(out, p)
		}
	}

	return out
}
