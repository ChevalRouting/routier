package configtest

import (
	"github.com/ChevalRouting/routier/tests/testkit"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func validateErr(t *testing.T, yaml string) string {
	t.Helper()

	errs := config.Validate(testkit.LoadCfg(t, yaml), false)
	if len(errs) == 0 {
		return ""
	}

	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}

	return strings.Join(msgs, "; ")
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	yaml := `version: v3.0.0
hostname: gw
interfaces:
  lan:
    select: name=eth0
    addresses: ["192.168.1.1/24"]
wireguard:
  wg0:
    private_key: dGVzdA==
    listen_port: 51820
    addresses: ["10.10.0.1/24"]
    peers:
      - public_key: cGVlcg==
        allowed_ips: ["10.10.0.2/32"]
        endpoint: peer.example.com:51820
friends:
  - name: edge
    url: https://10.0.0.2:8080
    token: secret
    identity:
      fingerprint: SHA256:abc
    sync:
      sections: [vrrp, conntrackd]
`

	if msg := validateErr(t, yaml); msg != "" {
		t.Fatalf("expected valid config, got: %s", msg)
	}
}

func TestValidateWireguard(t *testing.T) {
	cases := map[string]string{
		"missing private key": `version: v3.0.0
hostname: gw
wireguard:
  wg0:
    addresses: ["10.0.0.1/24"]
`,
		"bad address cidr": `version: v3.0.0
hostname: gw
wireguard:
  wg0:
    private_key: dGVzdA==
    addresses: ["not-a-cidr"]
`,
		"peer without public key": `version: v3.0.0
hostname: gw
wireguard:
  wg0:
    private_key: dGVzdA==
    addresses: ["10.0.0.1/24"]
    peers:
      - allowed_ips: ["10.0.0.2/32"]
`,
		"friend tag without friend": `version: v3.0.0
hostname: gw
wireguard:
  wg0:
    friend: ghost
    private_key: dGVzdA==
    addresses: ["10.0.0.1/24"]
`,
	}

	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) {
			if validateErr(t, yaml) == "" {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateFriends(t *testing.T) {
	cases := map[string]string{
		"missing token": `version: v3.0.0
hostname: gw
friends:
  - name: edge
    url: https://10.0.0.2:8080
`,
		"missing url": `version: v3.0.0
hostname: gw
friends:
  - name: edge
    token: secret
`,
		"bad fingerprint": `version: v3.0.0
hostname: gw
friends:
  - name: edge
    url: https://10.0.0.2:8080
    token: secret
    identity:
      fingerprint: not-sha256
`,
		"unreplicable sync section": `version: v3.0.0
hostname: gw
friends:
  - name: edge
    url: https://10.0.0.2:8080
    token: secret
    sync:
      sections: [routing]
`,
		"duplicate name": `version: v3.0.0
hostname: gw
friends:
  - name: edge
    url: https://10.0.0.2:8080
    token: a
  - name: edge
    url: https://10.0.0.3:8080
    token: b
`,
	}

	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) {
			if validateErr(t, yaml) == "" {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateDuplicateSeq(t *testing.T) {
	cases := map[string]string{
		"prefix-list duplicate seq": `version: v3.0.0
hostname: gw
routing:
  bgp:
    asn: 65000
    router_id: 10.0.0.1
    prefix_lists:
      L1:
        - { seq: 10, action: permit, prefix: "10.0.0.0/8" }
        - { seq: 10, action: deny, prefix: "192.168.0.0/16" }
`,
		"route-map duplicate seq": `version: v3.0.0
hostname: gw
routing:
  bgp:
    asn: 65000
    router_id: 10.0.0.1
    route_maps:
      M1:
        - { seq: 5, action: permit }
        - { seq: 5, action: deny }
`,
		"vrf bgp route-map duplicate seq": `version: v3.0.0
hostname: gw
vrfs:
  blue:
    table: 100
routing:
  vrfs:
    blue:
      bgp:
        asn: 65000
        router_id: 10.0.0.1
        route_maps:
          M1:
            - { seq: 5, action: permit }
            - { seq: 5, action: deny }
`,
		"pbr map duplicate seq": `version: v3.0.0
hostname: gw
routing:
  pbr:
    nexthop_groups:
      G1:
        nexthops:
          - address: 10.0.0.2
    maps:
      P1:
        - { seq: 10, set_nexthop_group: G1 }
        - { seq: 10, set_nexthop_group: G1 }
`,
	}

	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) {
			msg := validateErr(t, yaml)
			if !strings.Contains(msg, "duplicate seq") {
				t.Fatalf("expected duplicate seq error, got: %q", msg)
			}
		})
	}
}

func TestValidateDistinctSeqAccepted(t *testing.T) {
	yaml := `version: v3.0.0
hostname: gw
routing:
  bgp:
    asn: 65000
    router_id: 10.0.0.1
    prefix_lists:
      L1:
        - { seq: 10, action: permit, prefix: "10.0.0.0/8" }
        - { seq: 20, action: deny, prefix: "192.168.0.0/16" }
    route_maps:
      M1:
        - { seq: 5, action: permit }
        - { seq: 10, action: deny }
`

	if msg := validateErr(t, yaml); msg != "" {
		t.Fatalf("expected distinct seqs to pass, got: %s", msg)
	}
}

func TestValidateDHCPValid(t *testing.T) {
	yaml := `version: v3.0.0
hostname: gw
interfaces:
  lan:
    select: name=eth0
    addresses: ["10.0.0.1/24"]
dhcp:
  enabled: true
  subnets4:
    - subnet: 10.0.0.0/24
      interface: lan
      gateway: 10.0.0.1
      pools: ["10.0.0.100-10.0.0.200"]
      exclusions: ["10.0.0.2-10.0.0.20", "10.0.0.50"]
      dns: ["1.1.1.1"]
      reservations:
        - hostname: printer
          hw_address: aa:bb:cc:dd:ee:ff
          ip_address: 10.0.0.5
  subnets6:
    - subnet: 2001:db8::/64
      reservations:
        - duid: "00:03:00:01:aa:bb:cc:dd:ee:ff"
          ip_address: 2001:db8::5
`

	if msg := validateErr(t, yaml); msg != "" {
		t.Fatalf("expected valid dhcp config, got: %s", msg)
	}
}

func TestValidateDHCP(t *testing.T) {
	base := `version: v3.0.0
hostname: gw
interfaces:
  lan:
    select: name=eth0
    addresses: ["10.0.0.1/24"]
dhcp:
  enabled: true
  subnets4:
    - subnet: 10.0.0.0/24
`

	cases := map[string]string{
		"reservation outside subnet": base + `      reservations:
        - hw_address: aa:bb:cc:dd:ee:ff
          ip_address: 10.1.0.5
`,
		"pool outside subnet": base + `      pools: ["10.0.0.100-10.9.0.200"]
`,
		"pool must be a range": base + `      pools: ["10.0.0.100"]
`,
		"exclusion outside subnet": base + `      exclusions: ["10.9.0.5"]
`,
		"bad mac": base + `      reservations:
        - hw_address: nope
`,
		"reservation without identifier": base + `      reservations:
        - ip_address: 10.0.0.5
`,
		"gateway on v6": `version: v3.0.0
hostname: gw
dhcp:
  enabled: true
  subnets6:
    - subnet: 2001:db8::/64
      gateway: 2001:db8::1
`,
		"family mismatch": `version: v3.0.0
hostname: gw
dhcp:
  enabled: true
  subnets4:
    - subnet: 2001:db8::/64
`,
		"unknown interface": base + `      interface: wan
`,
	}

	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) {
			if validateErr(t, yaml) == "" {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateRejectsBadDeviceNames(t *testing.T) {
	cases := map[string]string{
		"wireguard too long": `version: v3.0.0
hostname: gw
wireguard:
  nuketown_fr_par_2:
    private_key: dGVzdA==
    addresses: ["172.31.0.33/29"]
`,
		"wireguard with colon": `version: v3.0.0
hostname: gw
wireguard:
  wg:0:
    private_key: dGVzdA==
`,
		"tunnel too long": `version: v3.0.0
hostname: gw
tunnels:
  gre-to-the-other-site:
    mode: gre
    local: 10.0.0.1
    remote: 10.0.0.2
`,
		"vrf too long": `version: v3.0.0
hostname: gw
vrfs:
  a-very-long-vrf-name:
    table: 100
`,
		"dummy too long": `version: v3.0.0
hostname: gw
interfaces:
  loopback-anycast0:
    type: dummy
    addresses: ["10.255.0.53/32"]
`,
	}

	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) {
			if validateErr(t, yaml) == "" {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateRejectsReservedDeviceNames(t *testing.T) {
	cases := map[string]string{
		"tunnel gre0": `version: v3.0.0
hostname: gw
tunnels:
  gre0:
    mode: gre
    local: 10.0.0.1
    remote: 10.0.0.2
`,
		"tunnel gretap0": `version: v3.0.0
hostname: gw
tunnels:
  gretap0:
    mode: gretap
    local: 10.0.0.1
    remote: 10.0.0.2
`,
		"tunnel sit0": `version: v3.0.0
hostname: gw
tunnels:
  sit0:
    mode: sit
    local: 10.0.0.1
    remote: 10.0.0.2
`,
		"wireguard tunl0": `version: v3.0.0
hostname: gw
wireguard:
  tunl0:
    private_key: dGVzdA==
`,
		"vrf ip6tnl0": `version: v3.0.0
hostname: gw
vrfs:
  ip6tnl0:
    table: 100
`,
		"dummy erspan0": `version: v3.0.0
hostname: gw
interfaces:
  erspan0:
    type: dummy
    addresses: ["10.255.0.53/32"]
`,
	}

	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) {
			got := validateErr(t, yaml)
			if got == "" {
				t.Fatal("expected validation error")
			}

			if !strings.Contains(got, "reserved kernel device name") {
				t.Fatalf("expected reserved-name error, got %q", got)
			}
		})
	}
}

func TestValidateAcceptsNonReservedTunnelName(t *testing.T) {
	yaml := `version: v3.0.0
hostname: gw
tunnels:
  gre1:
    mode: gre
    local: 10.0.0.1
    remote: 10.0.0.2
    addresses: ["10.255.255.1/30"]
`

	if got := validateErr(t, yaml); got != "" {
		t.Fatalf("expected no validation error, got %q", got)
	}
}

func TestValidateVXLANExternalAccepted(t *testing.T) {
	cases := map[string]string{
		"external without vni": `version: v3.0.0
hostname: gw
interfaces:
  vx0:
    type: vxlan
    vxlan:
      external: true
`,
		"two externals share a port with vnifilter": `version: v3.0.0
hostname: gw
interfaces:
  vx0:
    type: vxlan
    vxlan:
      external: true
      vnifilter: true
  vx1:
    type: vxlan
    vxlan:
      external: true
      vnifilter: true
`,
	}

	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) {
			if err := validateErr(t, yaml); err != "" {
				t.Fatalf("expected valid config, got: %s", err)
			}
		})
	}
}

func TestValidateVXLANExternalRejected(t *testing.T) {
	cases := map[string]struct{ yaml, want string }{
		"external with vni": {yaml: `version: v3.0.0
hostname: gw
interfaces:
  vx0:
    type: vxlan
    vxlan:
      external: true
      vni: 100
`, want: "vni is ignored when external"},
		"vnifilter without external": {yaml: `version: v3.0.0
hostname: gw
interfaces:
  vx0:
    type: vxlan
    vxlan:
      vni: 100
      vnifilter: true
`, want: "vnifilter requires external"},
		"non-external missing vni": {yaml: `version: v3.0.0
hostname: gw
interfaces:
  vx0:
    type: vxlan
    vxlan: {}
`, want: "vni must be between"},
		"two externals share a port without vnifilter": {yaml: `version: v3.0.0
hostname: gw
interfaces:
  vx0:
    type: vxlan
    vxlan:
      external: true
  vx1:
    type: vxlan
    vxlan:
      external: true
`, want: "cannot share the port"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			msg := validateErr(t, tc.yaml)
			if !strings.Contains(msg, tc.want) {
				t.Fatalf("expected error containing %q, got: %s", tc.want, msg)
			}
		})
	}
}

func TestValidateBondAccepted(t *testing.T) {
	cases := map[string]string{
		"lacp with hash policy": `version: v3.0.0
hostname: gw
interfaces:
  eth0:
    select: name=eth0
  eth1:
    select: name=eth1
  bond0:
    type: bond
    addresses: ["10.0.0.1/24"]
    bond:
      mode: 802.3ad
      members: [eth0, eth1]
      miimon: 100
      xmit_hash_policy: layer3+4
      lacp_rate: fast
`,
		"active-backup with primary": `version: v3.0.0
hostname: gw
interfaces:
  eth0:
    select: name=eth0
  eth1:
    select: name=eth1
  bond0:
    type: bond
    bond:
      mode: active-backup
      members: [eth0, eth1]
      primary: eth0
`,
	}

	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) {
			if err := validateErr(t, yaml); err != "" {
				t.Fatalf("expected valid config, got: %s", err)
			}
		})
	}
}

func TestValidateBondRejected(t *testing.T) {
	cases := map[string]struct{ yaml, want string }{
		"unknown mode": {yaml: `version: v3.0.0
hostname: gw
interfaces:
  eth0:
    select: name=eth0
  bond0:
    type: bond
    bond:
      mode: turbo
      members: [eth0]
`, want: "mode \"turbo\" is not one of"},
		"no members": {yaml: `version: v3.0.0
hostname: gw
interfaces:
  bond0:
    type: bond
    bond:
      mode: balance-rr
`, want: "at least one member is required"},
		"unknown member": {yaml: `version: v3.0.0
hostname: gw
interfaces:
  bond0:
    type: bond
    bond:
      mode: balance-rr
      members: [eth9]
`, want: "member \"eth9\" not found"},
		"lacp rate on non-lacp mode": {yaml: `version: v3.0.0
hostname: gw
interfaces:
  eth0:
    select: name=eth0
  bond0:
    type: bond
    bond:
      mode: active-backup
      members: [eth0]
      lacp_rate: fast
`, want: "lacp_rate only applies to 802.3ad"},
		"hash policy on active-backup": {yaml: `version: v3.0.0
hostname: gw
interfaces:
  eth0:
    select: name=eth0
  bond0:
    type: bond
    bond:
      mode: active-backup
      members: [eth0]
      xmit_hash_policy: layer3+4
`, want: "xmit_hash_policy only applies to"},
		"primary not a member": {yaml: `version: v3.0.0
hostname: gw
interfaces:
  eth0:
    select: name=eth0
  eth1:
    select: name=eth1
  bond0:
    type: bond
    bond:
      mode: active-backup
      members: [eth0]
      primary: eth1
`, want: "primary \"eth1\" must also be a member"},
		"bond settings without type bond": {yaml: `version: v3.0.0
hostname: gw
interfaces:
  eth0:
    select: name=eth0
  bond0:
    select: name=eth1
    bond:
      mode: balance-rr
      members: [eth0]
`, want: "bond settings require type bond"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			msg := validateErr(t, tc.yaml)
			if !strings.Contains(msg, tc.want) {
				t.Fatalf("expected error containing %q, got: %s", tc.want, msg)
			}
		})
	}
}

func TestValidateRejectsBlankFRRInterfaceNames(t *testing.T) {
	cases := map[string]string{
		"ospf blank interface": `version: v3.0.0
hostname: gw
routing:
  ospf:
    router_id: 1.1.1.1
    interfaces:
      "":
        area: 0.0.0.0
`,
		"ospf6 blank interface": `version: v3.0.0
hostname: gw
routing:
  ospf6:
    router_id: 1.1.1.1
    interfaces:
      "":
        area: 0.0.0.0
`,
		"pbr blank policy interface": `version: v3.0.0
hostname: gw
routing:
  pbr:
    maps:
      m1:
        - seq: 10
          set_nexthop: 10.0.0.1
    policies:
      "": m1
`,
	}

	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(validateErr(t, yaml), "interface name is required") {
				t.Fatalf("expected a blank interface-name error, got: %s", validateErr(t, yaml))
			}
		})
	}
}

func TestValidateBGPVPNRouteMaps(t *testing.T) {
	valid := `version: v3.0.0
hostname: gw
routing:
  bgp:
    asn: 65000
    router_id: 10.0.0.1
    route_maps:
      rm:
        - seq: 10
          action: permit
    address_families:
      ipv4-unicast:
        route_map_vpn_import: rm
        route_map_vpn_export: rm
`
	if msg := validateErr(t, valid); msg != "" {
		t.Fatalf("expected valid config, got: %s", msg)
	}

	vrfRefsGlobal := `version: v3.0.0
hostname: gw
vrfs:
  fabric1:
    table: 100
routing:
  bgp:
    asn: 65000
    router_id: 10.0.0.1
    route_maps:
      IMPORT-DEFAULT:
        - seq: 10
          action: permit
  vrfs:
    fabric1:
      bgp:
        asn: 65000
        router_id: 10.0.0.1
        address_families:
          ipv6-unicast:
            route_map_vpn_import: IMPORT-DEFAULT
`
	if msg := validateErr(t, vrfRefsGlobal); msg != "" {
		t.Fatalf("vrf bgp referencing a global route-map should be valid, got: %s", msg)
	}

	cases := map[string]struct{ yaml, want string }{
		"missing route map": {yaml: `version: v3.0.0
hostname: gw
routing:
  bgp:
    asn: 65000
    router_id: 10.0.0.1
    address_families:
      ipv4-unicast:
        route_map_vpn_import: nope
`, want: "route_map_vpn_import \"nope\" not found in route_maps"},
		"vpn route map on evpn": {yaml: `version: v3.0.0
hostname: gw
routing:
  bgp:
    asn: 65000
    router_id: 10.0.0.1
    route_maps:
      rm:
        - seq: 10
          action: permit
    address_families:
      l2vpn-evpn:
        route_map_vpn_export: rm
`, want: "VPN route-leak options require"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			msg := validateErr(t, tc.yaml)
			if !strings.Contains(msg, tc.want) {
				t.Fatalf("expected error containing %q, got: %s", tc.want, msg)
			}
		})
	}
}

func TestValidateAcceptsDeviceNamesAtTheLimit(t *testing.T) {
	yaml := `version: v3.0.0
hostname: gw
wireguard:
  fr_par_2:
    private_key: dGVzdA==
    addresses: ["172.31.0.33/29"]
  abcdefghijklmno:
    private_key: dGVzdA==
interfaces:
  dum0:
    type: dummy
    addresses: ["10.255.0.53/32"]
`
	if err := validateErr(t, yaml); err != "" {
		t.Fatalf("valid device names rejected: %s", err)
	}
}
