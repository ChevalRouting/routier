package configtest

import (
	"strings"
	"testing"
)

const kernelRouteMapValid = `version: v5.0.0
hostname: gw
routing:
  bgp:
    asn: 65000
    router_id: 10.0.0.1
    route_maps:
      LOOPBACK-SRC:
        - { seq: 10, action: permit, set: { src: 10.0.0.1 } }
    address_families:
      ipv4-unicast:
        kernel_route_map: LOOPBACK-SRC
`

const kernelRouteMapMissing = `version: v5.0.0
hostname: gw
routing:
  bgp:
    asn: 65000
    router_id: 10.0.0.1
    address_families:
      ipv6-unicast:
        kernel_route_map: NOPE
`

const kernelRouteMapEVPN = `version: v5.0.0
hostname: gw
routing:
  bgp:
    asn: 65000
    router_id: 10.0.0.1
    route_maps:
      LOOPBACK-SRC:
        - { seq: 10, action: permit }
    address_families:
      l2vpn-evpn:
        kernel_route_map: LOOPBACK-SRC
`

const kernelRouteMapVRF = `version: v5.0.0
hostname: gw
vrfs:
  fabric1:
    table: 100
routing:
  bgp:
    asn: 65000
    router_id: 10.0.0.1
    route_maps:
      LOOPBACK-SRC:
        - { seq: 10, action: permit }
  vrfs:
    fabric1:
      bgp:
        asn: 65000
        router_id: 10.0.0.1
        address_families:
          ipv4-unicast:
            kernel_route_map: LOOPBACK-SRC
`

func TestValidateBGPKernelRouteMap(t *testing.T) {
	if msg := validateErr(t, kernelRouteMapValid); msg != "" {
		t.Fatalf("expected valid config, got: %s", msg)
	}

	cases := map[string][2]string{
		"missing route map": {kernelRouteMapMissing, `kernel_route_map "NOPE" not found in route_maps`},
		"evpn family":       {kernelRouteMapEVPN, "kernel_route_map requires the ipv4-unicast or ipv6-unicast address family"},
		"vrf instance":      {kernelRouteMapVRF, "kernel_route_map is only supported in routing.bgp"},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if msg := validateErr(t, c[0]); !strings.Contains(msg, c[1]) {
				t.Fatalf("expected %q, got: %s", c[1], msg)
			}
		})
	}
}
