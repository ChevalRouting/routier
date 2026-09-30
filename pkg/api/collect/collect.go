package collect

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	webdb "github.com/ChevalRouting/routier/pkg/db"

	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/net/iproute"
	"github.com/ChevalRouting/routier/pkg/lldp"
	"github.com/ChevalRouting/routier/pkg/stats"
	"github.com/ChevalRouting/routier/pkg/types"
)

func Run(ctx context.Context, dbPath, configPath string) error {
	db, err := webdb.InitDB(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer db.Close()

	var intervals config.CollectionIntervals
	if configPath != "" {
		if cfg, err := cfgstore.Read(configPath, ""); err == nil && cfg.Monitoring != nil {
			intervals = cfg.Monitoring.Collection
		}
	}

	return collectAndStore(ctx, db, intervals)
}

func intervalOrDefault(v, def int) int64 {
	if v > 0 {
		return int64(v)
	}

	return int64(def)
}

func shouldCollect(ctx context.Context, db *webdb.DB, table string, intervalSec int64, now int64) bool {
	return now-webdb.LastTS(ctx, db, table) >= intervalSec
}

func collectAndStore(ctx context.Context, db *webdb.DB, iv config.CollectionIntervals) error {
	now := time.Now().Unix()

	if shouldCollect(ctx, db, "iface_stats", intervalOrDefault(iv.Iface, 60), now) {
		if err := storeIfaceStats(ctx, db, now); err != nil {
			return fmt.Errorf("iface stats: %w", err)
		}
	}

	if shouldCollect(ctx, db, "bgp_peer_stats", intervalOrDefault(iv.BGP, 60), now) {
		if err := storeBGPStats(ctx, db, now); err != nil {
			return fmt.Errorf("bgp stats: %w", err)
		}
	}

	if shouldCollect(ctx, db, "proto_stats", intervalOrDefault(iv.Proto, 60), now) {
		if err := storeProtoStats(ctx, db, now); err != nil {
			return fmt.Errorf("proto stats: %w", err)
		}
	}

	if shouldCollect(ctx, db, "system_stats", intervalOrDefault(iv.System, 60), now) {
		if err := storeSystemStats(ctx, db, now); err != nil {
			return fmt.Errorf("system stats: %w", err)
		}
	}

	if shouldCollect(ctx, db, "neighbor_stats", intervalOrDefault(iv.Neighbors, 60), now) {
		if err := storeNeighborStats(ctx, db, now); err != nil {
			return fmt.Errorf("neighbor stats: %w", err)
		}
	}

	if shouldCollect(ctx, db, "lldp_neighbors", intervalOrDefault(iv.LLDP, 300), now) {
		if err := storeLLDPNeighbors(ctx, db, now); err != nil {
			return fmt.Errorf("lldp neighbors: %w", err)
		}
	}

	return nil
}

func storeLLDPNeighbors(ctx context.Context, db *webdb.DB, now int64) error {
	raw := lldp.ShowNeighbors()
	if len(raw) == 0 {
		return nil
	}

	rows := make([]webdb.LLDPNeighborRow, 0, len(raw))
	for _, n := range raw {
		rows = append(rows, webdb.LLDPNeighborRow{
			LocalIface: n.LocalIface, Protocol: n.Protocol,
			ChassisID: n.ChassisID, ChassisName: n.ChassisName, SysDescr: n.SysDescr,
			MgmtIP: n.MgmtIP, PortID: n.PortID, PortDescr: n.PortDescr,
			Capabilities: n.Capabilities, VLAN: n.VLAN, Age: n.Age,
		})
	}

	return webdb.InsertLLDPNeighbors(ctx, db, now, rows)
}

type dnsResult struct {
	n        iproute.Neighbor
	hostname string
}

func storeNeighborStats(ctx context.Context, db *webdb.DB, now int64) error {
	raw := iproute.ShowNeighbors()
	if len(raw) == 0 {
		return nil
	}

	ch := make(chan dnsResult, len(raw))
	for _, n := range raw {
		go func(n iproute.Neighbor) {
			hostname := ""
			lookupCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
			defer cancel()
			if names, err := net.DefaultResolver.LookupAddr(lookupCtx, n.Dst); err == nil && len(names) > 0 {
				hostname = strings.TrimSuffix(names[0], ".")
			}

			ch <- dnsResult{n: n, hostname: hostname}
		}(n)
	}

	rows := make([]webdb.NeighborStatRow, 0, len(raw))
	for range raw {
		r := <-ch
		rows = append(rows, webdb.NeighborStatRow{
			IP: r.n.Dst, MAC: r.n.LLAddr, Dev: r.n.Dev,
			State: r.n.State, Family: r.n.Family, Hostname: r.hostname,
		})
	}

	return webdb.InsertNeighborStats(ctx, db, now, rows)
}

func storeIfaceStats(ctx context.Context, db *webdb.DB, now int64) error {
	current := stats.ReadIfaceStats()
	if len(current) == 0 {
		return nil
	}

	prevByIface := webdb.LastIfaceCounters(ctx, db)

	rows := make([]webdb.IfaceStatRow, 0, len(current))
	for name, st := range current {
		var rxBytesPS, txBytesPS, rxPps, txPps *float64
		if p, ok := prevByIface[name]; ok && p.TS > 0 && now > p.TS {
			dt := float64(now - p.TS)
			if int64(st.RxBytes) >= p.RxBytes {
				v := float64(int64(st.RxBytes)-p.RxBytes) / dt
				rxBytesPS = &v
			}

			if int64(st.TxBytes) >= p.TxBytes {
				v := float64(int64(st.TxBytes)-p.TxBytes) / dt
				txBytesPS = &v
			}

			if int64(st.RxPackets) >= p.RxPkts {
				v := float64(int64(st.RxPackets)-p.RxPkts) / dt
				rxPps = &v
			}

			if int64(st.TxPackets) >= p.TxPkts {
				v := float64(int64(st.TxPackets)-p.TxPkts) / dt
				txPps = &v
			}
		}

		rows = append(rows, webdb.IfaceStatRow{
			Iface:   name,
			RxBytes: st.RxBytes, TxBytes: st.TxBytes, RxPkts: st.RxPackets, TxPkts: st.TxPackets,
			RxErrs: st.RxErrors, TxErrs: st.TxErrors,
			RxBytesPS: rxBytesPS, TxBytesPS: txBytesPS, RxPps: rxPps, TxPps: txPps,
			OperState: st.OperState,
		})
	}

	return webdb.InsertIfaceStats(ctx, db, now, rows)
}

func storeSystemStats(ctx context.Context, db *webdb.DB, now int64) error {
	sys := stats.ReadSystemStats()
	if sys == nil {
		return nil
	}

	return webdb.InsertSystemStats(ctx, db, now, sys)
}

func storeBGPStats(ctx context.Context, db *webdb.DB, now int64) error {
	bgp := stats.ReadBGPStats()
	if bgp == nil {
		return nil
	}

	return webdb.InsertBGPStats(ctx, db, now, bgp)
}

func storeProtoStats(ctx context.Context, db *webdb.DB, now int64) error {
	ps := readProtoStats()
	if ps == nil {
		return nil
	}

	return webdb.InsertProtoStats(ctx, db, now, ps)
}

func readProtoStats() *types.ProtoStats {
	kv, err := parseProcSnmp("/proc/net/snmp")
	if err != nil {
		return nil
	}

	ps := &types.ProtoStats{}
	if tcp := kv["Tcp"]; tcp != nil {
		ps.TCPActiveOpens = tcp["ActiveOpens"]
		ps.TCPPassiveOpens = tcp["PassiveOpens"]
		ps.TCPAttemptFails = tcp["AttemptFails"]
		ps.TCPEstabResets = tcp["EstabResets"]
		ps.TCPCurrEstab = tcp["CurrEstab"]
		ps.TCPInSegs = tcp["InSegs"]
		ps.TCPOutSegs = tcp["OutSegs"]
		ps.TCPRetransSegs = tcp["RetransSegs"]
	}

	if udp := kv["Udp"]; udp != nil {
		ps.UDPInDatagrams = udp["InDatagrams"]
		ps.UDPOutDatagrams = udp["OutDatagrams"]
		ps.UDPInErrors = udp["InErrors"]
		ps.UDPNoPorts = udp["NoPorts"]
	}

	if icmp := kv["Icmp"]; icmp != nil {
		ps.ICMPInMsgs = icmp["InMsgs"]
		ps.ICMPOutMsgs = icmp["OutMsgs"]
	}

	return ps
}

func parseProcSnmp(path string) (map[string]map[string]int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	kv := make(map[string]map[string]int64)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for i := 0; i+1 < len(lines); i += 2 {
		headers := strings.Fields(lines[i])
		values := strings.Fields(lines[i+1])
		if len(headers) < 2 {
			continue
		}

		proto := strings.TrimSuffix(headers[0], ":")
		m := make(map[string]int64, len(headers)-1)
		for j := 1; j < len(headers) && j < len(values); j++ {
			v, _ := strconv.ParseInt(values[j], 10, 64)
			m[headers[j]] = v
		}

		kv[proto] = m
	}

	return kv, nil
}
