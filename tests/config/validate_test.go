package configtest

import (
	"github.com/ChevalRouting/routier/tests/harness"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func validateErr(t *testing.T, yaml string) string {
	t.Helper()

	errs := config.Validate(harness.LoadCfg(t, yaml), false)
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
