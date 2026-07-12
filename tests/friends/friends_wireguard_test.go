package friendstest

import (
	"strconv"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/types"
)

func TestDeriveWireguard(t *testing.T) {
	res, err := friends.DeriveWireguard(friends.WGDeriveParams{
		FriendName:    "peer-b",
		LocalHostname: "peer-a",
		Subnet:        "169.254.50.0/31",
		LocalAddr:     "198.51.100.1",
		FriendAddr:    "198.51.100.2",
	})
	if err != nil {
		t.Fatalf("derive: %v", err)
	}

	if res.InterfaceName != "wg-fr-peer-b" {
		t.Fatalf("interface name = %q", res.InterfaceName)
	}

	if res.Local.Table != "off" || res.Counterpart.Table != "off" {
		t.Fatal("derived tunnels must be link-only (table off)")
	}

	if res.Local.Friend != "peer-b" || res.Counterpart.Friend != "peer-a" {
		t.Fatalf("provenance tags wrong: %q / %q", res.Local.Friend, res.Counterpart.Friend)
	}

	if res.Local.Addresses[0] != "169.254.50.0/31" || res.Counterpart.Addresses[0] != "169.254.50.1/31" {
		t.Fatalf("tunnel addresses wrong: %v / %v", res.Local.Addresses, res.Counterpart.Addresses)
	}

	lp := res.Local.Peers[0]
	cp := res.Counterpart.Peers[0]
	if lp.AllowedIPs[0] != "169.254.50.1/32" || cp.AllowedIPs[0] != "169.254.50.0/32" {
		t.Fatalf("allowed_ips not link-only: %v / %v", lp.AllowedIPs, cp.AllowedIPs)
	}

	if lp.Endpoint != "198.51.100.2:"+strconv.Itoa(res.Counterpart.ListenPort) {
		t.Fatalf("local peer endpoint = %q", lp.Endpoint)
	}

	if cp.Endpoint != "198.51.100.1:"+strconv.Itoa(res.Local.ListenPort) {
		t.Fatalf("counterpart peer endpoint = %q", cp.Endpoint)
	}

	if lp.PresharedKey == "" || lp.PresharedKey != cp.PresharedKey {
		t.Fatal("both sides must share one PSK")
	}

	if res.Local.PrivateKey == res.Counterpart.PrivateKey {
		t.Fatal("each side must have its own private key")
	}
}

func TestDeriveWireguardSharedPort(t *testing.T) {
	res, err := friends.DeriveWireguard(friends.WGDeriveParams{
		FriendName: "peer-b", LocalHostname: "peer-a", Subnet: "169.254.50.0/31",
		LocalAddr: "198.51.100.1", FriendAddr: "198.51.100.2",
	})
	if err != nil {
		t.Fatalf("derive: %v", err)
	}

	if res.Local.ListenPort != res.Counterpart.ListenPort {
		t.Fatalf("expected shared default port, got %d / %d", res.Local.ListenPort, res.Counterpart.ListenPort)
	}

	if res.Local.ListenPort < 10000 || res.Local.ListenPort > 65000 {
		t.Fatalf("port out of range: %d", res.Local.ListenPort)
	}
}

func TestResolveEndpointAddr(t *testing.T) {
	ifaces := []types.FriendInterface{
		{Name: "wan", Addresses: []string{"198.51.100.1"}},
		{Name: "lan", Addresses: []string{"10.0.0.1", "10.0.1.1"}},
		{Name: "dyn", Addresses: nil},
	}

	if got, err := friends.ResolveEndpointAddr(ifaces, "wan", ""); err != nil || got != "198.51.100.1" {
		t.Fatalf("wan auto = %q, %v", got, err)
	}

	if _, err := friends.ResolveEndpointAddr(ifaces, "lan", ""); err == nil {
		t.Fatal("expected error for multi-address interface without explicit address")
	}

	if got, err := friends.ResolveEndpointAddr(ifaces, "lan", "10.0.1.1"); err != nil || got != "10.0.1.1" {
		t.Fatalf("lan explicit = %q, %v", got, err)
	}

	if _, err := friends.ResolveEndpointAddr(ifaces, "dyn", ""); err == nil {
		t.Fatal("expected error for interface with no static address")
	}

	if _, err := friends.ResolveEndpointAddr(ifaces, "missing", ""); err == nil {
		t.Fatal("expected error for missing interface")
	}
}

func TestInterfaceAddressesSkipsDynamic(t *testing.T) {
	cfg := &config.Config{Interfaces: map[string]*config.Interface{
		"wan": {Addresses: []string{"dhcp"}},
		"lan": {Addresses: []string{"10.0.0.1/24", "slaac"}},
	}}

	byName := map[string][]string{}
	for _, i := range friends.InterfaceAddresses(cfg) {
		byName[i.Name] = i.Addresses
	}

	if len(byName["wan"]) != 0 {
		t.Fatalf("dhcp should be skipped: %v", byName["wan"])
	}

	if len(byName["lan"]) != 1 || byName["lan"][0] != "10.0.0.1" {
		t.Fatalf("lan addresses = %v", byName["lan"])
	}
}
