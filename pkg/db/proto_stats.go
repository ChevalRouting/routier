package db

import (
	"database/sql"
	"fmt"

	"github.com/ChevalRouting/routier/pkg/types"
)

func ensureProtoStatsSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS proto_stats (
		id                INTEGER PRIMARY KEY AUTOINCREMENT,
		ts                INTEGER NOT NULL,
		tcp_active_opens  INTEGER NOT NULL DEFAULT 0,
		tcp_passive_opens INTEGER NOT NULL DEFAULT 0,
		tcp_attempt_fails INTEGER NOT NULL DEFAULT 0,
		tcp_estab_resets  INTEGER NOT NULL DEFAULT 0,
		tcp_curr_estab    INTEGER NOT NULL DEFAULT 0,
		tcp_in_segs       INTEGER NOT NULL DEFAULT 0,
		tcp_out_segs      INTEGER NOT NULL DEFAULT 0,
		tcp_retrans_segs  INTEGER NOT NULL DEFAULT 0,
		udp_in_datagrams  INTEGER NOT NULL DEFAULT 0,
		udp_out_datagrams INTEGER NOT NULL DEFAULT 0,
		udp_in_errors     INTEGER NOT NULL DEFAULT 0,
		udp_no_ports      INTEGER NOT NULL DEFAULT 0,
		icmp_in_msgs      INTEGER NOT NULL DEFAULT 0,
		icmp_out_msgs     INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		return err
	}

	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_proto_stats ON proto_stats(ts DESC)`)

	return nil
}

func InsertProtoStats(db *sql.DB, ts int64, p *types.ProtoStats) error {
	_, err := db.Exec(
		`INSERT INTO proto_stats
		 (ts, tcp_active_opens, tcp_passive_opens, tcp_attempt_fails, tcp_estab_resets,
		  tcp_curr_estab, tcp_in_segs, tcp_out_segs, tcp_retrans_segs,
		  udp_in_datagrams, udp_out_datagrams, udp_in_errors, udp_no_ports,
		  icmp_in_msgs, icmp_out_msgs)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ts,
		p.TCPActiveOpens, p.TCPPassiveOpens, p.TCPAttemptFails, p.TCPEstabResets,
		p.TCPCurrEstab, p.TCPInSegs, p.TCPOutSegs, p.TCPRetransSegs,
		p.UDPInDatagrams, p.UDPOutDatagrams, p.UDPInErrors, p.UDPNoPorts,
		p.ICMPInMsgs, p.ICMPOutMsgs,
	)
	return err
}

func ProtoHistory(db *sql.DB, cutoff, bucket int64) []types.ProtoHistoryPoint {
	var out []types.ProtoHistoryPoint
	if bucket < 1 {
		bucket = 1
	}

	q := fmt.Sprintf(
		`SELECT MAX(ts) AS ts, MAX(tcp_active_opens), MAX(tcp_passive_opens), MAX(tcp_attempt_fails), MAX(tcp_estab_resets),
		        MAX(tcp_curr_estab), MAX(tcp_in_segs), MAX(tcp_out_segs), MAX(tcp_retrans_segs),
		        MAX(udp_in_datagrams), MAX(udp_out_datagrams), MAX(udp_in_errors), MAX(udp_no_ports),
		        MAX(icmp_in_msgs), MAX(icmp_out_msgs)
		 FROM proto_stats WHERE ts >= ? GROUP BY ts/%d ORDER BY ts ASC`, bucket)

	rows, err := db.Query(q, cutoff)
	if err != nil {
		return out
	}

	defer rows.Close()

	for rows.Next() {
		var p types.ProtoHistoryPoint
		if rows.Scan(&p.TS,
			&p.TCPActiveOpens, &p.TCPPassiveOpens, &p.TCPAttemptFails, &p.TCPEstabResets,
			&p.TCPCurrEstab, &p.TCPInSegs, &p.TCPOutSegs, &p.TCPRetransSegs,
			&p.UDPInDatagrams, &p.UDPOutDatagrams, &p.UDPInErrors, &p.UDPNoPorts,
			&p.ICMPInMsgs, &p.ICMPOutMsgs,
		) == nil {
			out = append(out, p)
		}
	}

	return out
}
