package db

import (
	"database/sql"
	"fmt"

	"github.com/ChevalRouting/routier/pkg/types"
)

func ensureBGPStatsSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS bgp_peer_stats (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		ts         INTEGER NOT NULL,
		peer       TEXT    NOT NULL,
		remote_asn INTEGER NOT NULL DEFAULT 0,
		state      TEXT    NOT NULL DEFAULT '',
		uptime     TEXT    NOT NULL DEFAULT '',
		msg_rcvd   INTEGER NOT NULL DEFAULT 0,
		msg_sent   INTEGER NOT NULL DEFAULT 0,
		prefixes   INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		return err
	}

	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_bgp_peer_stats ON bgp_peer_stats(peer, ts DESC)`)

	return nil
}

func InsertBGPStats(db *sql.DB, ts int64, b *types.BGPStats) error {
	stmt, err := db.Prepare(
		`INSERT INTO bgp_peer_stats (ts, peer, remote_asn, state, uptime, msg_rcvd, msg_sent, prefixes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}

	defer stmt.Close()

	for _, p := range b.Peers {
		if _, err := stmt.Exec(ts, p.Address, p.ASN, p.State, p.Uptime, p.MsgRcvd, p.MsgSent, p.Prefixes); err != nil {
			return err
		}
	}

	return nil
}

func BGPHistory(db *sql.DB, cutoff, bucket int64) map[string][]types.BGPHistoryPoint {
	out := make(map[string][]types.BGPHistoryPoint)
	if bucket < 1 {
		bucket = 1
	}

	q := fmt.Sprintf(
		`SELECT MAX(ts) AS ts, peer, MAX(state), MAX(uptime), MAX(msg_rcvd), MAX(msg_sent), MAX(prefixes)
		 FROM bgp_peer_stats WHERE ts >= ? GROUP BY peer, ts/%d ORDER BY peer, ts ASC`, bucket)

	rows, err := db.Query(q, cutoff)
	if err != nil {
		return out
	}

	defer rows.Close()

	for rows.Next() {
		var p types.BGPHistoryPoint
		var peer string
		if rows.Scan(&p.TS, &peer, &p.State, &p.Uptime, &p.MsgRcvd, &p.MsgSent, &p.Prefixes) == nil {
			out[peer] = append(out[peer], p)
		}
	}

	return out
}
