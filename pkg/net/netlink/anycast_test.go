//go:build linux

package netlink

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	vnl "github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func gatewayConfig(mac string) *config.Config {
	return &config.Config{Interfaces: map[string]*config.Interface{
		"fabric": {Type: "vxlan-bridge", Device: "fabric", MTU: 1450, VXLAN: &config.VXLAN{Local: "192.0.2.10"}},
		"campus": {
			Type: "vnet", Device: "campus", Select: "fabric", Addresses: []string{"10.50.0.2/24", "fd50::2/64"},
			VNet: &config.VNet{VNI: 10040, VLAN: 40, Gateway: &config.VNetGateway{IPs: []string{"10.50.0.1", "fd50::1"}, MAC: mac}},
		},
	}}
}

func readSysctl(t *testing.T, key string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(sysctlRoot, strings.ReplaceAll(key, ".", "/")))
	if err != nil {
		t.Fatal(err)
	}

	return strings.TrimSpace(string(data))
}

func TestVNetAnycastGateway(t *testing.T) {
	enterTestNetns(t)
	cfg := gatewayConfig("")
	for range 2 {
		if err := ReconcileLinks(cfg, false); err != nil {
			t.Fatal(err)
		}

		if err := reconcileAddrs(cfg, false); err != nil {
			t.Fatal(err)
		}
	}

	name := config.VNetLinkName("campus", "g")
	gateway, err := vnl.LinkByName(name)
	if err != nil {
		t.Fatal(err)
	}

	bridge, _ := vnl.LinkByName("campus")
	if gateway.Type() != "macvlan" || gateway.Attrs().ParentIndex != bridge.Attrs().Index || gateway.Attrs().HardwareAddr.String() != "02:cb:00:27:38:01" {
		t.Fatalf("gateway link %s parent %d mac %s", gateway.Type(), gateway.Attrs().ParentIndex, gateway.Attrs().HardwareAddr)
	}

	addresses, err := vnl.AddrList(gateway, vnl.FAMILY_ALL)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]int{}
	for _, address := range addresses {
		got[address.IPNet.String()] = address.Flags
	}

	if _, ok := got["10.50.0.1/32"]; !ok || got["fd50::1/128"]&unix.IFA_F_NODAD == 0 {
		t.Fatalf("gateway addresses %v", got)
	}

	if readSysctl(t, "net.ipv4.conf.campus.arp_ignore") != "1" || readSysctl(t, "net.ipv4.conf."+name+".arp_ignore") != "1" || readSysctl(t, "net.ipv4.conf."+name+".rp_filter") != "2" {
		t.Fatal("the bridge would answer ARP for the gateway, or the gateway drops traffic returning through the bridge")
	}

	if err := ReconcileLinks(gatewayConfig("02:00:00:00:00:40"), false); err != nil {
		t.Fatal(err)
	}

	if gateway, _ = vnl.LinkByName(name); gateway.Attrs().HardwareAddr.String() != "02:00:00:00:00:40" {
		t.Fatalf("mac not updated: %s", gateway.Attrs().HardwareAddr)
	}

	cfg.Interfaces["campus"].VNet.Gateway = nil
	if err := ReconcileLinks(cfg, false); err != nil {
		t.Fatal(err)
	}

	if _, err := vnl.LinkByName(name); err == nil {
		t.Fatal("removing the gateway left its link")
	}
}
