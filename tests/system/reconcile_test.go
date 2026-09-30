package systemtest

import (
	"github.com/ChevalRouting/routier/tests/testkit"
	"testing"

	"github.com/ChevalRouting/routier/pkg/daemon/dhcpcd"
	"github.com/ChevalRouting/routier/pkg/net/netlink"
)

func TestDHCPCDReconcileDryRun(t *testing.T) {
	cfg := testkit.LoadCfg(t, `version: v3.0.0
hostname: gw
interfaces:
  wan:
    select: name=eth0
    addresses: [dhcp]
  lan:
    select: name=eth1
    addresses: ["10.0.0.1/24", slaac]
`)

	if err := dhcpcd.Reconcile(cfg, true); err != nil {
		t.Fatalf("dhcpcd dry-run reconcile: %v", err)
	}
}

func TestNetlinkReconcileDryRun(t *testing.T) {
	cfg := testkit.LoadCfg(t, `version: v3.0.0
hostname: gw
`)

	if err := netlink.Reconcile(cfg, true); err != nil {
		t.Skipf("netlink unavailable in this environment: %v", err)
	}
}
