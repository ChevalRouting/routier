package exporter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ChevalRouting/routier/pkg/types"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRegistryWritesTextFormat(t *testing.T) {
	r := NewRegistry()
	r.Counter("b_total", "Second.", 2, Label{"name", `a"b\c`})
	r.Gauge("a", "First\nline.", 1.5)
	r.Counter("b_total", "Second.", 3, Label{"name", "d"})

	var out strings.Builder
	if _, err := r.WriteTo(&out); err != nil {
		t.Fatal(err)
	}

	want := `# HELP a First\nline.
# TYPE a gauge
a 1.5
# HELP b_total Second.
# TYPE b_total counter
b_total{name="a\"b\\c"} 2
b_total{name="d"} 3
`
	if out.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestParseFRRUptime(t *testing.T) {
	cases := map[string]time.Duration{
		"01:02:03":  time.Hour + 2*time.Minute + 3*time.Second,
		"3d04h05m":  3*24*time.Hour + 4*time.Hour + 5*time.Minute,
		"2w1d03h":   15*24*time.Hour + 3*time.Hour,
		"never":     -1,
		"1x":        -1,
		"aa:bb:cc":  -1,
		"12d03hjnk": -1,
	}
	for in, want := range cases {
		got, ok := parseFRRUptime(in)
		if want < 0 {
			if ok {
				t.Errorf("%q: want no value, got %v", in, got)
			}

			continue
		}

		if !ok || got != want {
			t.Errorf("%q: got %v %v, want %v", in, got, ok, want)
		}
	}
}

func TestCollect(t *testing.T) {
	dir := t.TempDir()
	sysClassNet, procSys = filepath.Join(dir, "net"), filepath.Join(dir, "proc")
	t.Cleanup(func() { sysClassNet, procSys = "/sys/class/net", "/proc/sys" })

	writeFile(t, filepath.Join(sysClassNet, "eth0", "operstate"), "up")
	writeFile(t, filepath.Join(sysClassNet, "eth0", "statistics", "rx_bytes"), "1000")
	writeFile(t, filepath.Join(sysClassNet, "eth0", "statistics", "tx_dropped"), "7")
	writeFile(t, filepath.Join(sysClassNet, "eth1", "operstate"), "down")
	writeFile(t, filepath.Join(procSys, "net/netfilter/nf_conntrack_count"), "42")
	writeFile(t, filepath.Join(procSys, "net/netfilter/nf_conntrack_max"), "65536")

	rtt := 12.5
	r := Collect(Snapshot{
		Version:    "1.0.0-r0",
		System:     &types.SystemStats{CPUPercent: 25, MemTotal: 2048, Uptime: 60},
		BGP:        &types.BGPStats{Peers: []types.BGPPeerSummary{{Address: "192.0.2.1", ASN: 64500, State: "Established", Uptime: "00:01:40", Prefixes: 10}, {Address: "2001:db8::1", ASN: 64501, State: "Active", Uptime: "never"}}},
		OSPF:       &types.OSPFStats{Neighbors: []types.OSPFNeighborSummary{{NeighborID: "10.0.0.2", Interface: "eth1", State: "Full/DR", Area: "0.0.0.0"}}},
		Probes:     []types.ProbeHistoryPoint{{TS: 1700000000, Name: "uplink", Target: "192.0.2.254", Reachable: true, RTTAvgMS: &rtt}},
		Interfaces: map[string]string{"eth0": "wan"},
	})

	var out strings.Builder
	if _, err := r.WriteTo(&out); err != nil {
		t.Fatal(err)
	}

	for _, line := range []string{
		`routier_build_info{version="1.0.0-r0"} 1`,
		`routier_interface_up{device="eth0",interface="wan"} 1`,
		`routier_interface_up{device="eth1",interface=""} 0`,
		`routier_interface_receive_bytes_total{device="eth0",interface="wan"} 1000`,
		`routier_interface_transmit_drops_total{device="eth0",interface="wan"} 7`,
		`routier_cpu_usage_ratio 0.25`,
		`routier_memory_total_bytes 2048`,
		`routier_conntrack_entries 42`,
		`routier_conntrack_entries_max 65536`,
		`routier_bgp_peer_up{peer="192.0.2.1",asn="64500"} 1`,
		`routier_bgp_peer_up{peer="2001:db8::1",asn="64501"} 0`,
		`routier_bgp_peer_uptime_seconds{peer="192.0.2.1",asn="64500"} 100`,
		`routier_bgp_peer_prefixes_received{peer="192.0.2.1",asn="64500"} 10`,
		`routier_ospf_neighbor_full{neighbor="10.0.0.2",interface="eth1",area="0.0.0.0"} 1`,
		`routier_probe_up{probe="uplink",target="192.0.2.254"} 1`,
		`routier_probe_rtt_seconds{probe="uplink",target="192.0.2.254"} 0.0125`,
	} {
		if !strings.Contains(out.String(), line+"\n") {
			t.Errorf("missing %s", line)
		}
	}

	if strings.Contains(out.String(), `routier_bgp_peer_uptime_seconds{peer="2001:db8::1"`) {
		t.Error("a session that is not established must not report an uptime")
	}
}
