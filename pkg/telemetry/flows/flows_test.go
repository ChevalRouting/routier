package flows

import (
	"encoding/binary"
	"encoding/json"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
)

func ipv4Packet(proto uint8, src, dst string, sport, dport uint16, total uint16) []byte {
	b := make([]byte, 28)
	b[0] = 0x45
	binary.BigEndian.PutUint16(b[2:4], total)
	b[9] = proto
	copy(b[12:16], netip.MustParseAddr(src).AsSlice())
	copy(b[16:20], netip.MustParseAddr(dst).AsSlice())
	binary.BigEndian.PutUint16(b[20:22], sport)
	binary.BigEndian.PutUint16(b[22:24], dport)
	return b
}

func ipv6Packet(next uint8, ext []byte, src, dst string, sport, dport uint16, payload uint16) []byte {
	b := make([]byte, 40)
	b[0] = 0x60
	binary.BigEndian.PutUint16(b[4:6], payload)
	b[6] = next
	copy(b[8:24], netip.MustParseAddr(src).AsSlice())
	copy(b[24:40], netip.MustParseAddr(dst).AsSlice())
	b = append(b, ext...)
	return binary.BigEndian.AppendUint32(b, uint32(sport)<<16|uint32(dport))
}

func TestParseIPv4TCP(t *testing.T) {
	s, ok := ParsePacket(ipv4Packet(protoTCP, "192.0.2.1", "198.51.100.2", 40000, 443, 1500), 3, 4)
	if !ok {
		t.Fatal("packet was rejected")
	}

	want := Key{InDev: 3, OutDev: 4, Family: 4, Proto: protoTCP, Src: netip.MustParseAddr("192.0.2.1"), Dst: netip.MustParseAddr("198.51.100.2"), SrcPort: 40000, DstPort: 443}
	if s.Key != want || s.Length != 1500 {
		t.Fatalf("sampled = %+v, want key %+v length 1500", s, want)
	}
}

func TestParseIPv4FragmentHasNoPorts(t *testing.T) {
	b := ipv4Packet(protoUDP, "192.0.2.1", "198.51.100.2", 53, 53, 600)
	binary.BigEndian.PutUint16(b[6:8], 185)

	s, ok := ParsePacket(b, 1, 0)
	if !ok || s.Key.SrcPort != 0 || s.Key.DstPort != 0 {
		t.Fatalf("non-first fragment kept ports: %+v", s)
	}
}

func TestParseIPv6SkipsExtensionHeaders(t *testing.T) {
	hopByHop := []byte{protoUDP, 0, 0, 0, 0, 0, 0, 0}
	s, ok := ParsePacket(ipv6Packet(extHopByHop, hopByHop, "2001:db8::1", "2001:db8::2", 5353, 53, 100), 2, 0)
	if !ok || s.Key.Proto != protoUDP || s.Key.SrcPort != 5353 || s.Key.DstPort != 53 || s.Length != 140 {
		t.Fatalf("sampled = %+v", s)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	for _, b := range [][]byte{nil, {0x45, 0}, {0x60, 0, 0}, {0x10}} {
		if _, ok := ParsePacket(b, 1, 0); ok {
			t.Errorf("ParsePacket(%x) accepted garbage", b)
		}
	}
}

func TestCacheScalesBySamplingAndExpires(t *testing.T) {
	c := newCache(100)
	start := time.Unix(1000, 0)
	s := Sampled{Key: Key{Family: 4, Proto: protoTCP}, Length: 1000}
	c.add(s, start)
	c.add(s, start.Add(time.Second))

	if got := c.expire(start.Add(5*time.Second), false); len(got) != 0 {
		t.Fatalf("an active flow expired early: %+v", got)
	}

	got := c.expire(start.Add(time.Second+idleTimeout), false)
	if len(got) != 1 || got[0].Samples != 2 || got[0].Packets != 200 || got[0].Bytes != 200000 {
		t.Fatalf("expired = %+v", got)
	}

	if len(c.entries) != 0 {
		t.Fatal("expired flow stayed in the cache")
	}
}

func TestCacheActiveTimeoutSplitsLongFlows(t *testing.T) {
	c := newCache(1)
	start := time.Unix(1000, 0)
	s := Sampled{Key: Key{Family: 4}, Length: 1}
	for i := 0; i <= int(activeTimeout/time.Second); i += 5 {
		c.add(s, start.Add(time.Duration(i)*time.Second))
	}

	if got := c.expire(start.Add(activeTimeout), false); len(got) != 1 {
		t.Fatalf("a long-lived flow was not exported at the active timeout: %+v", got)
	}
}

func TestSettingsFromConfig(t *testing.T) {
	cfg := &config.Config{
		Interfaces: map[string]*config.Interface{"wan": {Device: "eth0"}},
		Monitoring: &config.MonitoringConfig{Flows: &config.FlowsConfig{
			Enabled:    true,
			Collectors: []config.FlowCollector{{Address: "192.0.2.10"}, {Address: "2001:db8::10", Port: 9999}},
		}},
	}

	s, ok := SettingsFromConfig(cfg)
	if !ok {
		t.Fatal("enabled flows produced no settings")
	}

	if s.Sampling != config.DefaultFlowSampling || len(s.Devices) != 1 || s.Devices[0] != "eth0" {
		t.Fatalf("settings = %+v", s)
	}

	if s.Collectors[0].String() != "192.0.2.10:6343" || s.Collectors[1].String() != "[2001:db8::10]:9999" {
		t.Fatalf("collectors = %v", s.Collectors)
	}

	if s.Retention != config.DefaultFlowRetentionHrs*3600 || s.MaxFlows != config.DefaultFlowMaxFlows {
		t.Fatalf("retention = %d, max = %d", s.Retention, s.MaxFlows)
	}

	if !s.Equal(s) {
		t.Fatal("settings should equal themselves")
	}

	if _, ok := SettingsFromConfig(&config.Config{}); ok {
		t.Fatal("missing flows config produced settings")
	}
}

func TestBuildMetadataForAkvorado(t *testing.T) {
	cfg := &config.Config{Hostname: "rt1", Interfaces: map[string]*config.Interface{"wan": {Device: "eth0"}}}
	exporters := []netip.Addr{netip.MustParseAddr("192.0.2.1"), netip.MustParseAddr("2001:db8::1")}
	ifaces := []NetInterface{{Index: 3, Name: "br-lan"}, {Index: 2, Name: "eth0"}}

	m := BuildMetadata(cfg, exporters, ifaces, func(dev string) uint64 { return 10_000_000_000 })
	if len(m.Exporters) != 2 || m.Exporters[0].ExporterSubnet != "192.0.2.1/32" || m.Exporters[1].ExporterSubnet != "2001:db8::1/128" {
		t.Fatalf("exporters = %+v", m.Exporters)
	}

	got := m.Exporters[0]
	if got.Name != "rt1" || got.Default.Name != "unknown" {
		t.Fatalf("exporter = %+v", got)
	}

	want := MetadataInterface{IfIndex: 2, MetadataLink: MetadataLink{Name: "eth0", Description: "wan", Speed: 10000}}
	if len(got.Interfaces) != 2 || got.Interfaces[0] != want {
		t.Fatalf("interfaces = %+v", got.Interfaces)
	}

	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), `"default":{"name":"unknown","description":"","speed":0}`) || !strings.Contains(string(data), `{"ifindex":2,"name":"eth0","description":"wan","speed":10000}`) {
		t.Fatalf("metadata json = %s", data)
	}
}

func TestEthernetFrameKeepsMACsAndCapsLength(t *testing.T) {
	hw := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0x81, 0x00}
	frame := ethernetFrame(hw, 6, make([]byte, 200))
	if len(frame) != maxSampledHeader {
		t.Fatalf("frame length %d, want %d", len(frame), maxSampledHeader)
	}

	if frame[0] != 1 || frame[11] != 12 || binary.BigEndian.Uint16(frame[12:14]) != etherTypeIPv6 {
		t.Fatalf("frame header = %x", frame[:14])
	}

	bare := ethernetFrame(nil, 4, []byte{0x45})
	if len(bare) != 15 || binary.BigEndian.Uint16(bare[12:14]) != etherTypeIPv4 || bare[0] != 0 {
		t.Fatalf("synthesized frame = %x", bare)
	}
}
