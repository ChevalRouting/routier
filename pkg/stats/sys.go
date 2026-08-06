package stats

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/ChevalRouting/routier/pkg/vtysh"
)

func ReadIfaceStats() map[string]*types.IfaceStats {
	out := make(map[string]*types.IfaceStats)

	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return out
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			continue
		}

		line := strings.TrimSpace(scanner.Text())
		before, after, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		name := strings.TrimSpace(before)
		fields := strings.Fields(after)
		if len(fields) < 16 {
			continue
		}

		st := &types.IfaceStats{
			RxBytes:   parseU64(fields[0]),
			RxPackets: parseU64(fields[1]),
			RxErrors:  parseU64(fields[2]),
			TxBytes:   parseU64(fields[8]),
			TxPackets: parseU64(fields[9]),
			TxErrors:  parseU64(fields[10]),
		}

		if data, err := os.ReadFile(filepath.Join("/sys/class/net", name, "operstate")); err == nil {
			st.OperState = strings.TrimSpace(string(data))
		} else {
			st.OperState = "unknown"
		}

		out[name] = st
	}

	_ = scanner.Err()
	return out
}

func PhysicalIfaces() map[string]bool {
	out := map[string]bool{}
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return out
	}

	for _, e := range entries {
		if _, err := os.Stat(filepath.Join("/sys/class/net", e.Name(), "device")); err == nil {
			out[e.Name()] = true
		}
	}

	return out
}

func parseU64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

type cpuTimes struct{ user, nice, system, idle, iowait, irq, softirq uint64 }

func readCPUTimes() (cpuTimes, bool) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuTimes{}, false
	}

	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			return cpuTimes{}, false
		}

		return cpuTimes{
			user:    parseU64(fields[1]),
			nice:    parseU64(fields[2]),
			system:  parseU64(fields[3]),
			idle:    parseU64(fields[4]),
			iowait:  parseU64(fields[5]),
			irq:     parseU64(fields[6]),
			softirq: parseU64(fields[7]),
		}, true
	}

	return cpuTimes{}, false
}

func buildSystemStats(t1 cpuTimes, ok1 bool, t2 cpuTimes, ok2 bool) *types.SystemStats {
	st := &types.SystemStats{}

	if ok1 && ok2 {
		idle1 := t1.idle + t1.iowait
		idle2 := t2.idle + t2.iowait
		total1 := t1.user + t1.nice + t1.system + t1.idle + t1.iowait + t1.irq + t1.softirq
		total2 := t2.user + t2.nice + t2.system + t2.idle + t2.iowait + t2.irq + t2.softirq
		dTotal := float64(total2 - total1)
		dIdle := float64(idle2 - idle1)
		if dTotal > 0 {
			st.CPUPercent = 100.0 * (dTotal - dIdle) / dTotal
		}
	}

	if f, err := os.Open("/proc/meminfo"); err == nil {
		defer f.Close()
		kv := map[string]uint64{}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			parts := strings.Fields(scanner.Text())
			if len(parts) >= 2 {
				key := strings.TrimSuffix(parts[0], ":")
				kv[key] = parseU64(parts[1]) * 1024
			}
		}

		st.MemTotal = kv["MemTotal"]
		st.MemBuffers = kv["Buffers"]
		st.MemCached = kv["Cached"] + kv["SReclaimable"]
		st.MemUsed = st.MemTotal - kv["MemFree"] - st.MemBuffers - st.MemCached
		st.SwapTotal = kv["SwapTotal"]
		st.SwapUsed = st.SwapTotal - kv["SwapFree"]
	}

	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 4 {
			st.Load1, _ = strconv.ParseFloat(fields[0], 64)
			st.Load5, _ = strconv.ParseFloat(fields[1], 64)
			st.Load15, _ = strconv.ParseFloat(fields[2], 64)
			parts := strings.Split(fields[3], "/")
			if len(parts) == 2 {
				st.Processes, _ = strconv.Atoi(parts[1])
			}
		}
	}

	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 1 {
			secs, _ := strconv.ParseFloat(fields[0], 64)
			st.Uptime = uint64(secs)
		}
	}

	return st
}

func ReadSystemStats() *types.SystemStats {
	t1, ok1 := readCPUTimes()
	time.Sleep(200 * time.Millisecond)
	t2, ok2 := readCPUTimes()
	return buildSystemStats(t1, ok1, t2, ok2)
}

type frrPeer struct {
	RemoteAs   int    `json:"remoteAs"`
	BgpState   string `json:"state"`
	PeerUptime string `json:"peerUptime"`
	MsgRcvd    int    `json:"msgRcvd"`
	MsgSent    int    `json:"msgSent"`
	PfxRcd     int    `json:"pfxRcd"`
}

type frr struct {
	RouterID string             `json:"routerId"`
	AS       int                `json:"as"`
	Peers    map[string]frrPeer `json:"peers"`
}

func ReadBGPStats() *types.BGPStats {
	out, err := vtysh.BGPSummaryJSON()
	if err != nil || len(out) == 0 {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil
	}

	stats := &types.BGPStats{}
	seen := map[string]bool{}

	for _, af := range []string{"ipv4Unicast", "ipv6Unicast", "l2VpnEvpn"} {
		raw, ok := raw[af]
		if !ok {
			continue
		}

		var af frr
		if err := json.Unmarshal(raw, &af); err != nil {
			continue
		}

		if stats.RouterID == "" {
			stats.RouterID = af.RouterID
			stats.LocalASN = af.AS
		}

		for addr, p := range af.Peers {
			if seen[addr] {
				continue
			}

			seen[addr] = true
			stats.Peers = append(stats.Peers, types.BGPPeerSummary{
				Address:  addr,
				ASN:      p.RemoteAs,
				State:    p.BgpState,
				Uptime:   p.PeerUptime,
				MsgRcvd:  p.MsgRcvd,
				MsgSent:  p.MsgSent,
				Prefixes: p.PfxRcd,
			})
		}
	}

	if stats.RouterID == "" && len(stats.Peers) == 0 {
		return nil
	}

	sort.Slice(stats.Peers, func(i, j int) bool {
		stateRank := func(s string) int {
			if s == "Established" {
				return 0
			}

			return 1
		}
		ri, rj := stateRank(stats.Peers[i].State), stateRank(stats.Peers[j].State)
		if ri != rj {
			return ri < rj
		}

		return stats.Peers[i].ASN < stats.Peers[j].ASN
	})

	return stats
}

type nbrEntry struct {
	NbrState  string `json:"nbrState"`
	Address   string `json:"address"`
	IfaceName string `json:"ifaceName"`
	AreaID    string `json:"areaId"`
}

func ReadOSPFStats() *types.OSPFStats {
	out, err := vtysh.OSPFNeighborsJSON()
	if err != nil || len(out) == 0 {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil
	}

	neighborRaw, ok := raw["neighbors"]
	if !ok {
		for _, v := range raw {
			var inner map[string]json.RawMessage
			if json.Unmarshal(v, &inner) == nil {
				if nb, found := inner["neighbors"]; found {
					neighborRaw = nb
					ok = true
					break
				}
			}
		}
	}

	if !ok {
		neighborRaw, _ = json.Marshal(raw)
	}

	var neighborMap map[string][]nbrEntry
	if err := json.Unmarshal(neighborRaw, &neighborMap); err != nil {
		return nil
	}

	stats := &types.OSPFStats{}
	for routerID, entries := range neighborMap {
		for _, e := range entries {
			iface := strings.SplitN(e.IfaceName, ":", 2)[0]
			stats.Neighbors = append(stats.Neighbors, types.OSPFNeighborSummary{
				NeighborID: routerID,
				Interface:  iface,
				State:      e.NbrState,
				Address:    e.Address,
				Area:       e.AreaID,
			})
		}
	}

	if len(stats.Neighbors) == 0 {
		return nil
	}

	sort.Slice(stats.Neighbors, func(i, j int) bool {
		return stats.Neighbors[i].NeighborID < stats.Neighbors[j].NeighborID
	})

	return stats
}

type procSnap struct {
	name  string
	state string
	cpu   uint64
	rss   uint64
	cmd   string
}

func sampleProcs() map[int]procSnap {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	m := make(map[int]procSnap, len(entries))
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}

		data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			continue
		}

		snap, ok := parseProcStat(string(data))
		if !ok {
			continue
		}

		if raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid)); err == nil && len(raw) > 0 {
			for i, b := range raw {
				if b == 0 {
					raw[i] = ' '
				}
			}

			cmd := strings.TrimSpace(string(raw))
			if len(cmd) > 200 {
				cmd = cmd[:200]
			}

			snap.cmd = cmd
		}

		m[pid] = snap
	}

	return m
}

func parseProcStat(s string) (procSnap, bool) {
	start := strings.Index(s, "(")
	end := strings.LastIndex(s, ")")
	if start < 0 || end <= start {
		return procSnap{}, false
	}

	name := s[start+1 : end]
	fields := strings.Fields(strings.TrimSpace(s[end+1:]))
	if len(fields) < 22 {
		return procSnap{}, false
	}

	return procSnap{
		name:  name,
		state: fields[0],
		cpu:   parseU64(fields[11]) + parseU64(fields[12]),
		rss:   parseU64(fields[21]),
	}, true
}

func buildProcList(s1, s2 map[int]procSnap, totalDelta float64) []types.ProcessInfo {
	pageSize := uint64(os.Getpagesize())
	procs := make([]types.ProcessInfo, 0, len(s2))
	for pid, p2 := range s2 {
		var cpuPct float64
		if p1, ok := s1[pid]; ok && totalDelta > 0 && p2.cpu >= p1.cpu {
			cpuPct = float64(p2.cpu-p1.cpu) / totalDelta * 100
		}

		procs = append(procs, types.ProcessInfo{
			PID:    pid,
			Name:   p2.name,
			State:  p2.state,
			CPUPct: cpuPct,
			MemRSS: p2.rss * pageSize,
			Cmd:    p2.cmd,
		})
	}

	sort.Slice(procs, func(i, j int) bool {
		if procs[i].CPUPct != procs[j].CPUPct {
			return procs[i].CPUPct > procs[j].CPUPct
		}

		return procs[i].PID < procs[j].PID
	})
	return procs
}

func Sample() (*types.SystemStats, []types.ProcessInfo, *types.BGPStats, *types.OSPFStats) {
	s1 := sampleProcs()
	cpu1, ok1 := readCPUTimes()
	time.Sleep(200 * time.Millisecond)
	s2 := sampleProcs()
	cpu2, ok2 := readCPUTimes()

	var totalDelta float64
	if ok1 && ok2 {
		sum := func(t cpuTimes) uint64 {
			return t.user + t.nice + t.system + t.idle + t.iowait + t.irq + t.softirq
		}
		totalDelta = float64(sum(cpu2) - sum(cpu1))
	}

	sys := buildSystemStats(cpu1, ok1, cpu2, ok2)
	procs := buildProcList(s1, s2, totalDelta)
	bgp := ReadBGPStats()
	ospf := ReadOSPFStats()
	return sys, procs, bgp, ospf
}
