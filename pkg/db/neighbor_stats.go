package db

import (
	"database/sql"

	"github.com/ChevalRouting/routier/pkg/types"
)

func ensureNeighborStatsSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS neighbor_stats (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		ts       INTEGER NOT NULL,
		ip       TEXT    NOT NULL,
		mac      TEXT    NOT NULL DEFAULT '',
		dev      TEXT    NOT NULL DEFAULT '',
		state    TEXT    NOT NULL DEFAULT '',
		family   TEXT    NOT NULL DEFAULT 'ipv4',
		hostname TEXT    NOT NULL DEFAULT ''
	)`)
	if err != nil {
		return err
	}

	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_neighbor_stats ON neighbor_stats(ts DESC)`)

	return nil
}

type NeighborStatRow struct {
	IP, MAC, Dev, State, Family, Hostname string
}

func InsertNeighborStats(db *sql.DB, ts int64, rows []NeighborStatRow) error {
	stmt, err := db.Prepare(`INSERT INTO neighbor_stats (ts, ip, mac, dev, state, family, hostname) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}

	defer stmt.Close()

	for _, r := range rows {
		if _, err := stmt.Exec(ts, r.IP, r.MAC, r.Dev, r.State, r.Family, r.Hostname); err != nil {
			return err
		}
	}

	return nil
}

func LatestNeighbors(db *sql.DB) (int64, []types.NeighborStat) {
	var maxTS int64
	if err := db.QueryRow("SELECT COALESCE(MAX(ts), 0) FROM neighbor_stats").Scan(&maxTS); err != nil || maxTS == 0 {
		return 0, nil
	}

	rows, err := db.Query(
		`SELECT ip, mac, dev, state, family, hostname FROM neighbor_stats WHERE ts = ? ORDER BY ip ASC`, maxTS)
	if err != nil {
		return maxTS, nil
	}

	defer rows.Close()

	var neighbors []types.NeighborStat
	for rows.Next() {
		var n types.NeighborStat
		if rows.Scan(&n.IP, &n.MAC, &n.Dev, &n.State, &n.Family, &n.Hostname) == nil {
			neighbors = append(neighbors, n)
		}
	}

	return maxTS, neighbors
}
