package exporter

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/types"
)

const iffUp = 0x1

var (
	sysClassNet = "/sys/class/net"
	procSys     = "/proc/sys"
)

type Snapshot struct {
	Version    string
	System     *types.SystemStats
	BGP        *types.BGPStats
	OSPF       *types.OSPFStats
	Probes     []types.ProbeHistoryPoint
	Interfaces map[string]string
}

func Collect(s Snapshot) *Registry {
	r := NewRegistry()
	r.Gauge("routier_build_info", "Routier build, always 1.", 1, Label{"version", s.Version})
	collectInterfaces(r, s.Interfaces)
	collectSystem(r, s.System)
	collectConntrack(r)
	collectBGP(r, s.BGP)
	collectOSPF(r, s.OSPF)
	collectProbes(r, s.Probes)
	return r
}

type ifaceCounter struct {
	file   string
	metric string
	help   string
}

var ifaceCounters = []ifaceCounter{
	{"rx_bytes", "routier_interface_receive_bytes_total", "Bytes received by the interface."},
	{"tx_bytes", "routier_interface_transmit_bytes_total", "Bytes transmitted by the interface."},
	{"rx_packets", "routier_interface_receive_packets_total", "Packets received by the interface."},
	{"tx_packets", "routier_interface_transmit_packets_total", "Packets transmitted by the interface."},
	{"rx_errors", "routier_interface_receive_errors_total", "Receive errors on the interface."},
	{"tx_errors", "routier_interface_transmit_errors_total", "Transmit errors on the interface."},
	{"rx_dropped", "routier_interface_receive_drops_total", "Received packets dropped by the interface."},
	{"tx_dropped", "routier_interface_transmit_drops_total", "Transmitted packets dropped by the interface."},
}

func collectInterfaces(r *Registry, names map[string]string) {
	entries, err := os.ReadDir(sysClassNet)
	if err != nil {
		return
	}

	for _, entry := range entries {
		device := entry.Name()
		labels := []Label{{"device", device}, {"interface", names[device]}}
		r.Gauge("routier_interface_up", "1 when the interface operational state is up.", boolValue(interfaceUp(device)), labels...)
		for _, c := range ifaceCounters {
			if v, ok := readUint(filepath.Join(sysClassNet, device, "statistics", c.file)); ok {
				r.Counter(c.metric, c.help, float64(v), labels...)
			}
		}
	}
}

func collectSystem(r *Registry, sys *types.SystemStats) {
	if sys == nil {
		return
	}

	r.Gauge("routier_cpu_usage_ratio", "CPU busy time over the last sample, between 0 and 1.", sys.CPUPercent/100)
	r.Gauge("routier_memory_total_bytes", "Total memory.", float64(sys.MemTotal))
	r.Gauge("routier_memory_used_bytes", "Memory in use, without buffers and cache.", float64(sys.MemUsed))
	r.Gauge("routier_memory_buffers_bytes", "Memory used by kernel buffers.", float64(sys.MemBuffers))
	r.Gauge("routier_memory_cached_bytes", "Memory used by the page cache.", float64(sys.MemCached))
	r.Gauge("routier_swap_total_bytes", "Total swap.", float64(sys.SwapTotal))
	r.Gauge("routier_swap_used_bytes", "Swap in use.", float64(sys.SwapUsed))
	r.Gauge("routier_load1", "Load average over 1 minute.", sys.Load1)
	r.Gauge("routier_load5", "Load average over 5 minutes.", sys.Load5)
	r.Gauge("routier_load15", "Load average over 15 minutes.", sys.Load15)
	r.Gauge("routier_uptime_seconds", "Time since boot.", float64(sys.Uptime))
	r.Gauge("routier_processes", "Number of processes.", float64(sys.Processes))
}

func collectConntrack(r *Registry) {
	if v, ok := readUint(filepath.Join(procSys, "net/netfilter/nf_conntrack_count")); ok {
		r.Gauge("routier_conntrack_entries", "Connection tracking entries in use.", float64(v))
	}

	if v, ok := readUint(filepath.Join(procSys, "net/netfilter/nf_conntrack_max")); ok {
		r.Gauge("routier_conntrack_entries_max", "Connection tracking table size.", float64(v))
	}
}

func collectBGP(r *Registry, bgp *types.BGPStats) {
	if bgp == nil {
		return
	}

	for _, p := range bgp.Peers {
		labels := []Label{{"peer", p.Address}, {"asn", strconv.Itoa(p.ASN)}}
		established := p.State == "Established"
		r.Gauge("routier_bgp_peer_up", "1 when the BGP session is established.", boolValue(established), labels...)
		r.Gauge("routier_bgp_peer_prefixes_received", "Prefixes received from the peer.", float64(p.Prefixes), labels...)
		r.Counter("routier_bgp_peer_messages_received_total", "BGP messages received from the peer.", float64(p.MsgRcvd), labels...)
		r.Counter("routier_bgp_peer_messages_sent_total", "BGP messages sent to the peer.", float64(p.MsgSent), labels...)
		if established {
			if d, ok := parseFRRUptime(p.Uptime); ok {
				r.Gauge("routier_bgp_peer_uptime_seconds", "Time since the BGP session was established.", d.Seconds(), labels...)
			}
		}
	}
}

func collectOSPF(r *Registry, ospf *types.OSPFStats) {
	if ospf == nil {
		return
	}

	for _, n := range ospf.Neighbors {
		labels := []Label{{"neighbor", n.NeighborID}, {"interface", n.Interface}, {"area", n.Area}}
		r.Gauge("routier_ospf_neighbor_full", "1 when the OSPF adjacency is full.", boolValue(strings.HasPrefix(n.State, "Full")), labels...)
	}
}

func collectProbes(r *Registry, probes []types.ProbeHistoryPoint) {
	for _, p := range probes {
		labels := []Label{{"probe", p.Name}, {"target", p.Target}}
		r.Gauge("routier_probe_up", "1 when the last probe got a reply.", boolValue(p.Reachable), labels...)
		r.Gauge("routier_probe_timestamp_seconds", "Time of the last probe.", float64(p.TS), labels...)
		if p.Reachable && p.RTTAvgMS != nil {
			r.Gauge("routier_probe_rtt_seconds", "Average round-trip time of the last probe.", *p.RTTAvgMS/1000, labels...)
		}
	}
}

var frrUptimePart = regexp.MustCompile(`(\d+)([wdhm])`)

func parseFRRUptime(s string) (time.Duration, bool) {
	if parts := strings.Split(s, ":"); len(parts) == 3 {
		var total time.Duration
		for i, unit := range []time.Duration{time.Hour, time.Minute, time.Second} {
			n, err := strconv.Atoi(parts[i])
			if err != nil {
				return 0, false
			}

			total += time.Duration(n) * unit
		}

		return total, true
	}

	matches := frrUptimePart.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 || strings.Join(flatten(matches), "") != s {
		return 0, false
	}

	units := map[string]time.Duration{"w": 7 * 24 * time.Hour, "d": 24 * time.Hour, "h": time.Hour, "m": time.Minute}
	var total time.Duration
	for _, m := range matches {
		n, _ := strconv.Atoi(m[1])
		total += time.Duration(n) * units[m[2]]
	}

	return total, true
}

func flatten(matches [][]string) []string {
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = m[0]
	}

	return out
}

func readString(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

func readUint(path string) (uint64, bool) {
	v, err := strconv.ParseUint(readString(path), 10, 64)
	return v, err == nil
}

func boolValue(b bool) float64 {
	if b {
		return 1
	}

	return 0
}

func interfaceUp(device string) bool {
	switch readString(filepath.Join(sysClassNet, device, "operstate")) {
	case "up":
		return true
	case "unknown":
		flags, err := strconv.ParseUint(strings.TrimPrefix(readString(filepath.Join(sysClassNet, device, "flags")), "0x"), 16, 32)
		return err == nil && flags&iffUp != 0
	default:
		return false
	}
}
