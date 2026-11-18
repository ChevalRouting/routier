package config

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func collectErrors(validate func(add addfunc)) string {
	var messages []string
	validate(func(msg string, args ...any) { messages = append(messages, fmt.Sprintf(msg, args...)) })

	return strings.Join(messages, "\n")
}

func TestIndexedDevicesSortsNumerically(t *testing.T) {
	names := []string{"eth10", "eth2", "eth0", "eth1", "eth0v10", "ethx", "lo"}
	want := []string{"eth0", "eth1", "eth2", "eth10"}

	if got := indexedDevices(names, "eth"); !slices.Equal(got, want) {
		t.Fatalf("indexedDevices = %v, want %v", got, want)
	}
}

func TestValidatePBRNilGroup(t *testing.T) {
	pbr := &PBR{NexthopGroups: map[string]*PBRNexthopGroup{"...": nil}}

	got := collectErrors(func(add addfunc) { validatePBR(pbr, nil, add) })
	if !strings.Contains(got, "at least one nexthop is required") {
		t.Fatalf("nil nexthop group not reported: %q", got)
	}
}

func TestValidateKeaPoolOrder(t *testing.T) {
	cfg := &Config{DHCP: &DHCP{Enabled: true, Subnets4: []KeaSubnet{{Subnet: "192.0.2.0/24", Pools: []string{"192.0.2.200-192.0.2.100"}}}}}

	got := collectErrors(func(add addfunc) { validateDHCP(cfg, add) })
	if !strings.Contains(got, "is after end") {
		t.Fatalf("reversed pool not reported: %q", got)
	}
}

func TestValidateRADVDRejectsIPv4(t *testing.T) {
	radvd := &RADVDConfig{Interfaces: map[string]*RADVDInterface{"lan": {
		Prefixes: []RADVDPrefix{{Prefix: "192.0.2.0/24"}},
		RDNSS:    &RADVDRDNSS{Servers: []string{"192.0.2.53"}},
	}}}

	got := collectErrors(func(add addfunc) {
		validateRADVD(radvd, map[string]*Interface{"lan": {}}, add)
	})

	for _, want := range []string{"is not a valid IPv6 CIDR", "is not a valid IPv6 address"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestValidateNftNameCollision(t *testing.T) {
	cfg := &Config{Interfaces: map[string]*Interface{"lan-a": {}, "lan_a": {}}}

	got := collectErrors(func(add addfunc) { validateNftNames(cfg, add) })
	if !strings.Contains(got, "same nftables variable prefix lan_a") {
		t.Fatalf("collision not reported: %q", got)
	}
}

func TestVRFBGPPolicies(t *testing.T) {
	global := &BGP{PrefixLists: map[string][]PrefixEntry{"SHARED": nil}}
	vrf := &BGP{PrefixLists: map[string][]PrefixEntry{"SHARED": nil, "BLUE": nil}}

	var policies bgpPolicies

	got := collectErrors(func(add addfunc) { policies = vrfBGPPolicies(vrf, global, "routing.vrfs.blue.bgp", add) })
	if !strings.Contains(got, "prefix_lists.SHARED: name is already used") {
		t.Fatalf("collision not reported: %q", got)
	}

	if _, ok := policies.prefixLists["BLUE"]; !ok {
		t.Fatal("vrf prefix list is not referenceable")
	}
}
