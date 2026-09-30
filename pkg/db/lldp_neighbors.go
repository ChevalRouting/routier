package db

import (
	"context"

	"github.com/ChevalRouting/routier/pkg/db/generated"
	"github.com/ChevalRouting/routier/pkg/types"
)

type LLDPNeighborRow struct {
	LocalIface, Protocol, ChassisID, ChassisName, SysDescr string
	MgmtIP, PortID, PortDescr, Capabilities, VLAN, Age     string
}

func InsertLLDPNeighbors(ctx context.Context, db *DB, ts int64, rows []LLDPNeighborRow) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	queries := db.queries.WithTx(tx)
	for _, r := range rows {
		if err := queries.InsertLLDPNeighbor(ctx, generated.InsertLLDPNeighborParams{
			Ts: ts, LocalIface: r.LocalIface, Protocol: r.Protocol,
			ChassisID: r.ChassisID, ChassisName: r.ChassisName, SysDescr: r.SysDescr,
			MgmtIp: r.MgmtIP, PortID: r.PortID, PortDescr: r.PortDescr,
			Capabilities: r.Capabilities, Vlan: r.VLAN, Age: r.Age,
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func LatestLLDP(ctx context.Context, db *DB) (int64, []types.LLDPNeighbor) {
	queries := db.queries
	maxTS, err := queries.LastLLDPTS(ctx)
	if err != nil || maxTS == 0 {
		return 0, nil
	}

	rows, err := queries.LatestLLDPNeighbors(ctx, maxTS)
	if err != nil {
		return maxTS, nil
	}

	neighbors := make([]types.LLDPNeighbor, 0, len(rows))
	for _, row := range rows {
		neighbors = append(neighbors, types.LLDPNeighbor{
			LocalIface: row.LocalIface, Protocol: row.Protocol,
			ChassisID: row.ChassisID, ChassisName: row.ChassisName, SysDescr: row.SysDescr,
			MgmtIP: row.MgmtIp, PortID: row.PortID, PortDescr: row.PortDescr,
			Capabilities: row.Capabilities, VLAN: row.Vlan, Age: row.Age,
		})
	}

	return maxTS, neighbors
}
