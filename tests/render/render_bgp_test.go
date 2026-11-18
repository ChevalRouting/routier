package rendertest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
)

func renderFrr(t *testing.T, cfg *config.Config) string {
	t.Helper()

	outs, err := render.All(cfg)
	if err != nil {
		t.Fatalf("All: %v", err)
	}

	for _, o := range outs {
		if o.Name == "frr/frr.conf" {
			return o.Content
		}
	}

	t.Fatal("frr/frr.conf not rendered")
	return ""
}

func TestBGPNeighborAFDisableActivate(t *testing.T) {
	mkCfg := bgpNeighborAFConfig

	enabled := renderFrr(t, mkCfg(false))
	if !strings.Contains(enabled, "neighbor 198.51.100.1 activate") || strings.Contains(enabled, "no neighbor 198.51.100.1 activate") {
		t.Fatalf("enabled AF should emit activate, got:\n%s", enabled)
	}

	disabled := renderFrr(t, mkCfg(true))
	if !strings.Contains(disabled, "no neighbor 198.51.100.1 activate") {
		t.Fatalf("disabled AF should emit `no ... activate`, got:\n%s", disabled)
	}
}

func vpnLeakAF(rd string) *config.BGPAddressFamily {
	return &config.BGPAddressFamily{
		RD:                 rd,
		RTVPNImport:        []string{rd},
		RTVPNExport:        []string{rd},
		LabelVPNExportAuto: true,
		ImportVPN:          true,
		ExportVPN:          true,
		RouteMapVPNImport:  "vpn-in",
		RouteMapVPNExport:  "vpn-out",
	}
}

func afSection(t *testing.T, out, header string) string {
	t.Helper()

	start := strings.Index(out, header)
	if start < 0 {
		t.Fatalf("section %q not found:\n%s", header, out)
	}

	rest := out[start:]
	end := strings.Index(rest, "exit-address-family")
	if end < 0 {
		t.Fatalf("section %q not terminated:\n%s", header, out)
	}

	return rest[:end]
}

func TestRenderBGPVPNRouteLeak(t *testing.T) {
	cfg := &config.Config{
		Hostname: "rtr",
		Routing: &config.Routing{
			BGP: &config.BGP{
				ASN: 65000,
				AddressFamilies: map[string]*config.BGPAddressFamily{
					"ipv4-unicast": vpnLeakAF("65000:4"),
					"ipv6-unicast": vpnLeakAF("65000:6"),
				},
			},
			VRFs: map[string]*config.VRFRouting{
				"blue": {BGP: &config.BGP{
					ASN: 65000,
					AddressFamilies: map[string]*config.BGPAddressFamily{
						"ipv4-unicast": vpnLeakAF("65000:14"),
						"ipv6-unicast": vpnLeakAF("65000:16"),
					},
				}},
			},
		},
	}

	out := renderFrr(t, cfg)

	for section, rd := range map[string]string{
		"address-family ipv4 unicast": "65000:4",
		"address-family ipv6 unicast": "65000:6",
	} {
		body := afSection(t, out, section)
		for _, want := range []string{
			"rd vpn export " + rd,
			"rt vpn import " + rd,
			"rt vpn export " + rd,
			"label vpn export auto",
			"import vpn",
			"export vpn",
			"route-map vpn import vpn-in",
			"route-map vpn export vpn-out",
		} {
			if !strings.Contains(body, want) {
				t.Errorf("%q section missing %q:\n%s", section, want, body)
			}
		}
	}

	for _, want := range []string{"rd vpn export 65000:14", "rd vpn export 65000:16"} {
		if !strings.Contains(out, want) {
			t.Errorf("VRF instance missing %q:\n%s", want, out)
		}
	}
}

func TestRenderBGPNoRIB(t *testing.T) {
	mkCfg := bgpNoRIBConfig

	on := renderFrr(t, mkCfg(true))
	if !strings.Contains(on, "\nbgp no-rib\n") {
		t.Errorf("expected global `bgp no-rib`, got:\n%s", on)
	}

	if strings.Contains(on, " bgp no-rib") {
		t.Errorf("`bgp no-rib` must be global, not indented under router bgp:\n%s", on)
	}

	off := renderFrr(t, mkCfg(false))
	if strings.Contains(off, "bgp no-rib") {
		t.Errorf("`bgp no-rib` must not render when disabled:\n%s", off)
	}
}

func TestRenderBGPL2VPNEVPN(t *testing.T) {
	cfg := &config.Config{
		Hostname: "leaf-1",
		Routing: &config.Routing{BGP: &config.BGP{
			ASN:      65001,
			RouterID: "192.0.2.1",
			Neighbors: []config.BGPNeighbor{{
				Address:   "192.0.2.2",
				RemoteASN: 65002,
				AddressFamilies: map[string]*config.BGPNeighborAF{
					"l2vpn-evpn": {},
				},
			}},
			AddressFamilies: map[string]*config.BGPAddressFamily{
				"l2vpn-evpn": {
					AdvertiseAllVNI:         true,
					AdvertiseDefaultGateway: true,
					AdvertiseSVIIP:          true,
					Advertise:               []string{"ipv4-unicast"},
					RouteTargetImport:       []string{"65001:100"},
					RouteTargetExport:       []string{"65001:100"},
				},
			},
		}},
	}

	out := renderFrr(t, cfg)
	for _, want := range []string{
		"address-family l2vpn evpn",
		"advertise-all-vni",
		"advertise-default-gw",
		"advertise-svi-ip",
		"advertise ipv4 unicast",
		"route-target import 65001:100",
		"route-target export 65001:100",
		"neighbor 192.0.2.2 activate",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("FRR config missing %q:\n%s", want, out)
		}
	}
}

func TestRenderVRFL3VNI(t *testing.T) {
	cfg := &config.Config{
		Hostname: "leaf-1",
		Interfaces: map[string]*config.Interface{
			"fabric": {Type: "vxlan-bridge", MTU: 1450, VXLAN: &config.VXLAN{Local: "192.0.2.1"}},
		},
		VRFs: map[string]*config.VRFConfig{
			"blue":  {Table: 100, L3VNI: &config.VRFL3VNI{VNI: 5000, Transport: "fabric", VLAN: 3999}},
			"green": {Table: 200},
		},
		Routing: &config.Routing{
			BGP: &config.BGP{ASN: 65001, RouterID: "192.0.2.1", AddressFamilies: map[string]*config.BGPAddressFamily{"l2vpn-evpn": {AdvertiseAllVNI: true}}},
			VRFs: map[string]*config.VRFRouting{"blue": {BGP: &config.BGP{
				ASN: 65001, RouterID: "192.0.2.1",
				AddressFamilies: map[string]*config.BGPAddressFamily{
					"ipv4-unicast": {Redistribute: []string{"connected"}},
					"l2vpn-evpn":   {Advertise: []string{"ipv4-unicast"}},
				},
			}}},
		},
	}

	out := renderFrr(t, cfg)
	block := "vrf blue\n vni 5000\nexit-vrf"
	if !strings.Contains(out, block) {
		t.Fatalf("FRR config missing the l3vni block:\n%s", out)
	}

	if strings.Contains(out, "vrf green\n vni") {
		t.Fatalf("a vrf without l3vni must not get a vni:\n%s", out)
	}

	if strings.Index(out, block) > strings.Index(out, "router bgp 65001 vrf blue") {
		t.Fatalf("the vrf vni must come before the vrf bgp instance:\n%s", out)
	}
}

func TestRenderImportVRFRouteMap(t *testing.T) {
	cfg := &config.Config{
		Hostname: "leaf-1",
		VRFs:     map[string]*config.VRFConfig{"blue": {Table: 100}},
		Routing: &config.Routing{
			BGP: &config.BGP{
				ASN: 65001, RouterID: "192.0.2.1",
				PrefixLists: map[string][]config.PrefixEntry{"DEFAULT": {{Seq: 10, Action: "permit", Prefix: "0.0.0.0/0"}}},
				RouteMaps:   map[string][]config.RouteMapEntry{"ONLY-DEFAULT": {{Seq: 10, Action: "permit", Match: map[string]string{"ip address prefix-list": "DEFAULT"}}}},
				AddressFamilies: map[string]*config.BGPAddressFamily{
					"ipv4-unicast": {Networks: []string{"0.0.0.0/0"}, ImportVRF: []string{"blue"}},
				},
			},
			VRFs: map[string]*config.VRFRouting{"blue": {BGP: &config.BGP{
				ASN: 65001, RouterID: "192.0.2.1",
				AddressFamilies: map[string]*config.BGPAddressFamily{
					"ipv4-unicast": {ImportVRF: []string{"default"}, ImportVRFRouteMap: "ONLY-DEFAULT"},
				},
			}}},
		},
	}

	if errs := config.Validate(cfg, false); len(errs) != 0 {
		t.Fatal(errs)
	}

	out := renderFrr(t, cfg)
	want := "  import vrf route-map ONLY-DEFAULT\n  import vrf default"
	if !strings.Contains(out, want) || !strings.Contains(out, "  import vrf blue") {
		t.Fatalf("FRR config missing the vrf import:\n%s", out)
	}

	cfg.Routing.VRFs["blue"].BGP.AddressFamilies["ipv4-unicast"].ImportVRFRouteMap = "MISSING"
	if errs := config.Validate(cfg, false); len(errs) == 0 {
		t.Fatal("an unknown import route map must be refused")
	}
}

func TestBGPFamilyRouteMapDefaultsApplyToNeighbors(t *testing.T) {
	for _, vrf := range []bool{false, true} {
		name := "global"
		if vrf {
			name = "vrf"
		}

		t.Run(name, func(t *testing.T) { testBGPFamilyRouteMapDefaults(t, vrf) })
	}
}

func testBGPFamilyRouteMapDefaults(t *testing.T, vrf bool) {
	t.Helper()
	bgp := &config.BGP{ASN: 65000,
		AddressFamilies: map[string]*config.BGPAddressFamily{"ipv4-unicast": {RouteMapIn: "EXTERNAL", RouteMapOut: "EXTERNAL"}},
		Neighbors: []config.BGPNeighbor{
			{Address: "192.0.2.1", RemoteASN: 65001},
			{Address: "192.0.2.2", RemoteASN: 65002, AddressFamilies: map[string]*config.BGPNeighborAF{"ipv4-unicast": {RouteMapIn: "CUSTOM"}}},
			{Address: "2001:db8::1", RemoteASN: 65003},
		},
	}

	cfg := &config.Config{Hostname: "rtr", Routing: &config.Routing{BGP: bgp}}
	if vrf {
		cfg.Routing = &config.Routing{VRFs: map[string]*config.VRFRouting{"blue": {BGP: bgp}}}
	}

	out := renderFrr(t, cfg)
	wants := []string{"neighbor 192.0.2.1 route-map EXTERNAL in", "neighbor 192.0.2.1 route-map EXTERNAL out", "neighbor 192.0.2.2 route-map CUSTOM in", "neighbor 192.0.2.2 route-map EXTERNAL out"}
	if !vrf {
		wants = []string{
			"neighbor 192.0.2.1 route-map ROUTIER-IN-192_0_2_1-ipv4-unicast in",
			"route-map ROUTIER-IN-192_0_2_1-ipv4-unicast permit 1000\n call EXTERNAL\n",
			"route-map ROUTIER-OUT-192_0_2_1-ipv4-unicast permit 1000\n call EXTERNAL\n",
			"route-map ROUTIER-IN-192_0_2_2-ipv4-unicast permit 1000\n call CUSTOM\n",
			"route-map ROUTIER-OUT-192_0_2_2-ipv4-unicast permit 1000\n call EXTERNAL\n",
			"route-map ROUTIER-IN-2001_db8__1-ipv6-unicast deny 9999\nexit",
		}
	}

	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}

	for _, bad := range []string{"\n  route-map EXTERNAL in", "\n  route-map EXTERNAL out", "neighbor 192.0.2.2 route-map EXTERNAL in", "neighbor 2001:db8::1 route-map EXTERNAL", "ROUTIER-IN-192_0_2_2-ipv4-unicast permit 1000\n call EXTERNAL"} {
		if strings.Contains(out, bad) {
			t.Fatalf("unexpected %q:\n%s", bad, out)
		}
	}
}

func TestRenderBGPPrefixListFamilies(t *testing.T) {
	cfg := bgpPrefixListFamiliesConfig()
	if errs := config.Validate(cfg, false); len(errs) != 0 {
		t.Fatal(errs)
	}

	out := renderFrr(t, cfg)
	wantLines(t, "frr.conf", out,
		"\nip prefix-list ANYCAST-V4 seq 10 permit 192.0.2.0/24 ge 32 le 32\n",
		"\nipv6 prefix-list ANYCAST-V6 seq 10 permit 2001:db8:53::10/128\n",
		"\nip prefix-list BOGONS seq 10 deny 10.0.0.0/8 le 32\n",
		"\nipv6 prefix-list BOGONS seq 20 deny fc00::/7 le 128\n",
		"route-map ANYCAST-IN permit 10\n match ip address prefix-list ANYCAST-V4\nexit",
		"route-map ANYCAST-IN permit 20\n match ipv6 address prefix-list ANYCAST-V6\nexit",
		"neighbor 2001:db8::2 route-map ROUTIER-IN-2001_db8__2-ipv6-unicast in",
		"route-map ROUTIER-IN-2001_db8__2-ipv6-unicast permit 1000\n call ANYCAST-IN\n",
	)

	wantLines(t, "ipv4 unicast", afSection(t, out, "address-family ipv4 unicast"), "neighbor 192.0.2.2 prefix-list BOGONS in")
	wantLines(t, "ipv6 unicast", afSection(t, out, "address-family ipv6 unicast"), "neighbor 2001:db8::2 prefix-list BOGONS in")

	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "ip prefix-list ") && strings.Contains(line, ":") {
			t.Errorf("IPv6 prefix rendered as an ip prefix-list: %q", line)
		}

		if strings.HasPrefix(line, "ipv6 prefix-list ") && strings.Contains(line, ".") {
			t.Errorf("IPv4 prefix rendered as an ipv6 prefix-list: %q", line)
		}
	}
}

func bgpPrefixListFamiliesConfig() *config.Config {
	return &config.Config{
		Hostname: "rtr",
		Routing: &config.Routing{BGP: &config.BGP{
			ASN:      65001,
			RouterID: "192.0.2.1",
			Neighbors: []config.BGPNeighbor{
				{Address: "192.0.2.2", RemoteASN: 65100, AddressFamilies: map[string]*config.BGPNeighborAF{
					"ipv4-unicast": {RouteMapIn: "ANYCAST-IN", PrefixListIn: "BOGONS"},
				}},
				{Address: "2001:db8::2", RemoteASN: 65100, AddressFamilies: map[string]*config.BGPNeighborAF{
					"ipv6-unicast": {RouteMapIn: "ANYCAST-IN", PrefixListIn: "BOGONS"},
				}},
			},
			PrefixLists: map[string][]config.PrefixEntry{
				"ANYCAST-V4": {{Seq: 10, Action: "permit", Prefix: "192.0.2.0/24", GE: 32, LE: 32}},
				"ANYCAST-V6": {{Seq: 10, Action: "permit", Prefix: "2001:db8:53::10/128"}},
				"BOGONS": {
					{Seq: 10, Action: "deny", Prefix: "10.0.0.0/8", LE: 32},
					{Seq: 20, Action: "deny", Prefix: "fc00::/7", LE: 128},
					{Seq: 30, Action: "permit", Prefix: "0.0.0.0/0", LE: 32},
					{Seq: 40, Action: "permit", Prefix: "::/0", LE: 128},
				},
			},
			RouteMaps: map[string][]config.RouteMapEntry{
				"ANYCAST-IN": {
					{Seq: 10, Action: "permit", Match: map[string]string{"ip address prefix-list": "ANYCAST-V4"}},
					{Seq: 20, Action: "permit", Match: map[string]string{"ipv6 address prefix-list": "ANYCAST-V6"}},
				},
			},
		}},
	}
}

func bgpNeighborAFConfig(disabled bool) *config.Config {
	return &config.Config{
		Hostname: "rtr",
		Routing: &config.Routing{
			BGP: &config.BGP{
				ASN: 65010,
				Neighbors: []config.BGPNeighbor{
					{
						Address:   "198.51.100.1",
						RemoteASN: 65020,
						AddressFamilies: map[string]*config.BGPNeighborAF{
							"ipv4-unicast": {Disabled: disabled},
						},
					},
				},
			},
		},
	}
}

func bgpNoRIBConfig(noRIB bool) *config.Config {
	return &config.Config{
		Hostname: "rr",
		Routing: &config.Routing{BGP: &config.BGP{
			ASN:      65000,
			RouterID: "10.0.0.1",
			NoRIB:    noRIB,
		}},
	}
}
