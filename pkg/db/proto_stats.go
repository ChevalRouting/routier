package db

import (
	"context"

	"github.com/ChevalRouting/routier/pkg/db/generated"
	"github.com/ChevalRouting/routier/pkg/types"
)

func InsertProtoStats(ctx context.Context, db *DB, ts int64, p *types.ProtoStats) error {
	return db.queries.InsertProtoStat(ctx, generated.InsertProtoStatParams{
		Ts: ts, TcpActiveOpens: p.TCPActiveOpens, TcpPassiveOpens: p.TCPPassiveOpens,
		TcpAttemptFails: p.TCPAttemptFails, TcpEstabResets: p.TCPEstabResets,
		TcpCurrEstab: p.TCPCurrEstab, TcpInSegs: p.TCPInSegs, TcpOutSegs: p.TCPOutSegs,
		TcpRetransSegs: p.TCPRetransSegs, UdpInDatagrams: p.UDPInDatagrams,
		UdpOutDatagrams: p.UDPOutDatagrams, UdpInErrors: p.UDPInErrors,
		UdpNoPorts: p.UDPNoPorts, IcmpInMsgs: p.ICMPInMsgs, IcmpOutMsgs: p.ICMPOutMsgs,
	})
}

func ProtoHistory(ctx context.Context, db *DB, cutoff, bucket int64) []types.ProtoHistoryPoint {
	var out []types.ProtoHistoryPoint
	if bucket < 1 {
		bucket = 1
	}

	rows, err := db.queries.ProtoHistory(ctx, generated.ProtoHistoryParams{Cutoff: cutoff, Bucket: bucket})
	if err != nil {
		return out
	}

	for _, row := range rows {
		out = append(out, types.ProtoHistoryPoint{
			TS: row.Ts, TCPActiveOpens: row.TcpActiveOpens, TCPPassiveOpens: row.TcpPassiveOpens,
			TCPAttemptFails: row.TcpAttemptFails, TCPEstabResets: row.TcpEstabResets,
			TCPCurrEstab: row.TcpCurrEstab, TCPInSegs: row.TcpInSegs, TCPOutSegs: row.TcpOutSegs,
			TCPRetransSegs: row.TcpRetransSegs, UDPInDatagrams: row.UdpInDatagrams,
			UDPOutDatagrams: row.UdpOutDatagrams, UDPInErrors: row.UdpInErrors,
			UDPNoPorts: row.UdpNoPorts, ICMPInMsgs: row.IcmpInMsgs, ICMPOutMsgs: row.IcmpOutMsgs,
		})
	}

	return out
}
