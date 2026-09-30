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
	mkCfg := func(disabled bool) *config.Config {
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
	mkCfg := func(noRIB bool) *config.Config {
		return &config.Config{
			Hostname: "rr",
			Routing: &config.Routing{BGP: &config.BGP{
				ASN:      65000,
				RouterID: "10.0.0.1",
				NoRIB:    noRIB,
			}},
		}
	}

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
