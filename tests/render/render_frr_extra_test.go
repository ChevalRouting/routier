package rendertest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestFRRExtraDirectivesEverywhere(t *testing.T) {
	cfg := &config.Config{
		Hostname:   "rtr",
		Interfaces: map[string]*config.Interface{"lan": {Device: "eth1"}},
		Routing: &config.Routing{
			BGP: &config.BGP{
				ASN:   65010,
				Extra: []string{"coalesce-time 1000"},
				AddressFamilies: map[string]*config.BGPAddressFamily{
					"ipv4-unicast": {Extra: []string{"aggregate-address 10.0.0.0/8 summary-only"}},
				},
			},
			OSPF: &config.OSPF{
				RouterID:   "1.1.1.1",
				Extra:      []string{"auto-cost reference-bandwidth 100000"},
				Areas:      []config.OSPFArea{{ID: "0.0.0.0", Extra: []string{"area 0.0.0.0 default-cost 10"}}},
				Interfaces: map[string]*config.OSPFInterface{"lan": {Area: "0.0.0.0", Extra: []string{"ip ospf network point-to-point"}}},
			},
			OSPF6: &config.OSPF6{
				RouterID:   "1.1.1.1",
				Extra:      []string{"maximum-paths 4"},
				Areas:      []config.OSPF6Area{{ID: "0.0.0.0", Extra: []string{"area 0.0.0.0 range 2001:db8::/32"}}},
				Interfaces: map[string]*config.OSPF6Interface{"lan": {Area: "0.0.0.0", Extra: []string{"ipv6 ospf6 network point-to-point"}}},
			},
			BFD: &config.BFD{Profiles: []config.BFDProfile{{Name: "fast", Extra: []string{"echo-interval 50"}}}},
		},
	}

	out := renderFrr(t, cfg)

	for _, want := range []string{
		"coalesce-time 1000",
		"aggregate-address 10.0.0.0/8 summary-only",
		"auto-cost reference-bandwidth 100000",
		"area 0.0.0.0 default-cost 10",
		"ip ospf network point-to-point",
		"maximum-paths 4",
		"area 0.0.0.0 range 2001:db8::/32",
		"ipv6 ospf6 network point-to-point",
		"echo-interval 50",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing extra directive %q in:\n%s", want, out)
		}
	}
}

func TestOSPFCuratedOptions(t *testing.T) {
	cfg := &config.Config{
		Hostname: "rtr",
		Routing: &config.Routing{
			OSPF: &config.OSPF{
				RouterID: "1.1.1.1", ReferenceBandwidth: 100000, Distance: 110,
				Areas:      []config.OSPFArea{{ID: "0.0.0.0", Type: "stub", StubNoSummary: true, Ranges: []string{"10.0.0.0/8"}, DefaultCost: 5}},
				Interfaces: map[string]*config.OSPFInterface{"lan": {Area: "0.0.0.0", NetworkType: "point-to-point", Priority: 200, RetransmitInterval: 7, TransmitDelay: 3, MTUIgnore: true, BFD: true}},
			},
			OSPF6: &config.OSPF6{
				RouterID: "1.1.1.1", ReferenceBandwidth: 100000, Distance: 110,
				Areas:      []config.OSPF6Area{{ID: "0.0.0.0", Type: "stub", StubNoSummary: true, DefaultCost: 5}},
				Interfaces: map[string]*config.OSPF6Interface{"lan": {Area: "0.0.0.0", NetworkType: "point-to-point", Priority: 200, MTUIgnore: true, BFD: true}},
			},
		},
		Interfaces: map[string]*config.Interface{"lan": {Device: "eth1"}},
	}
	out := renderFrr(t, cfg)
	for _, want := range []string{
		"auto-cost reference-bandwidth 100000", "distance 110",
		"area 0.0.0.0 stub no-summary", "area 0.0.0.0 range 10.0.0.0/8", "area 0.0.0.0 default-cost 5",
		"ip ospf network point-to-point", "ip ospf priority 200", "ip ospf retransmit-interval 7",
		"ip ospf transmit-delay 3", "ip ospf mtu-ignore", "ip ospf bfd",
		"ipv6 ospf6 network point-to-point", "ipv6 ospf6 priority 200", "ipv6 ospf6 mtu-ignore", "ipv6 ospf6 bfd",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
