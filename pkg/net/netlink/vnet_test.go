//go:build linux

package netlink

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	vnl "github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func TestVNetLifecycle(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	original, err := os.Open("/proc/self/task/" + strconv.Itoa(unix.Gettid()) + "/ns/net")
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = original.Close() }()
	if err := unix.Unshare(unix.CLONE_NEWNET); err != nil {
		t.Skipf("network namespace requires CAP_SYS_ADMIN: %v", err)
	}

	defer func() {
		if err := unix.Setns(int(original.Fd()), unix.CLONE_NEWNET); err != nil {
			t.Fatal(err)
		}
	}()
	cfg := &config.Config{Interfaces: map[string]*config.Interface{
		"fabric":  {Type: "vxlan-bridge", Device: "fabric", MTU: 1450, VXLAN: &config.VXLAN{Local: "192.0.2.10"}},
		"servers": {Type: "vnet", Device: "servers", Select: "fabric", VNet: &config.VNet{VNI: 10200, VLAN: 200}, Bridge: &config.Bridge{MemberDevices: []string{"tenant"}}},
	}}
	if err := vnl.LinkAdd(&vnl.Veth{LinkAttrs: vnl.LinkAttrs{Name: "tenant"}, PeerName: "client"}); err != nil {
		t.Fatal(err)
	}

	if err := ReconcileLinks(cfg, true); err != nil {
		t.Fatal(err)
	}

	if _, err := vnl.LinkByName("fabric"); err == nil {
		t.Fatal("dry run created the transport bridge")
	}

	for i := 0; i < 2; i++ {
		if err := ReconcileLinks(cfg, false); err != nil {
			t.Fatal(err)
		}
	}

	vx, err := vnl.LinkByName(config.VNetLinkName("fabric", "x"))
	if err != nil {
		t.Fatal(err)
	}

	if !vx.(*vnl.Vxlan).FlowBased || vx.(*vnl.Vxlan).Learning {
		t.Fatal("VXLAN must use metadata and no learning")
	}

	servers, err := vnl.LinkByName("servers")
	if err != nil {
		t.Fatal(err)
	}

	if err := syncAddrs("servers", []string{"10.20.0.1/24"}, false, false); err != nil {
		t.Fatal(err)
	}

	addresses, err := vnl.AddrList(servers, vnl.FAMILY_V4)
	if err != nil || len(addresses) != 1 || addresses[0].IP.String() != "10.20.0.1" {
		t.Fatalf("vnet address: %v, %v", addresses, err)
	}

	if err := vnl.LinkAdd(&vnl.Veth{LinkAttrs: vnl.LinkAttrs{Name: "tap-external"}, PeerName: "guest-external"}); err != nil {
		t.Fatal(err)
	}

	tap, err := vnl.LinkByName("tap-external")
	if err != nil {
		t.Fatal(err)
	}

	if err := vnl.LinkSetMaster(tap, servers); err != nil {
		t.Fatal(err)
	}

	if err := ReconcileLinks(cfg, false); err != nil {
		t.Fatal(err)
	}

	tap, err = vnl.LinkByName("tap-external")
	if err != nil || tap.Attrs().MasterIndex != servers.Attrs().Index {
		t.Fatal("reconcile removed an external VM TAP")
	}

	assertVNetTunnel(t, vx, 200, 10200)
	assertVNetMACLearning(t)
	if os.Getenv("ROUTIER_TEST_FRR") == "1" {
		assertVNetFRR(t)
	}

	cfg.Interfaces["servers"].VNet = &config.VNet{VNI: 10300, VLAN: 300}
	if err := ReconcileLinks(cfg, false); err != nil {
		t.Fatal(err)
	}

	assertVNetTunnel(t, vx, 300, 10300)
	delete(cfg.Interfaces, "servers")
	if err := ReconcileLinks(cfg, false); err != nil {
		t.Fatal(err)
	}

	for _, role := range []string{"a", "b"} {
		if _, err := vnl.LinkByName(config.VNetLinkName("servers", role)); err == nil {
			t.Fatal("stale access pair remains")
		}
	}

	tunnels, err := vnl.BridgeVlanTunnelShowDev(vx)
	if err != nil || len(tunnels) != 0 {
		t.Fatalf("stale tunnel mappings: %v, %v", tunnels, err)
	}
}

func assertVNetTunnel(t *testing.T, vx vnl.Link, vid uint16, vni uint32) {
	t.Helper()
	tunnels, err := vnl.BridgeVlanTunnelShowDev(vx)
	if err != nil || len(tunnels) != 1 || tunnels[0].Vid != vid || tunnels[0].TunId != vni {
		t.Fatalf("tunnel mappings: %v, %v", tunnels, err)
	}
}

func assertVNetMACLearning(t *testing.T) {
	t.Helper()
	client, err := vnl.LinkByName("client")
	if err != nil {
		t.Fatal(err)
	}

	if err := vnl.LinkSetUp(client); err != nil {
		t.Fatal(err)
	}

	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, 0)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = unix.Close(fd) }()
	frame := make([]byte, 60)
	copy(frame, []byte{255, 255, 255, 255, 255, 255, 2, 0, 0, 0, 0, 42, 0x88, 0xb5})

	access, err := vnl.LinkByName(config.VNetLinkName("servers", "a"))
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		if err := unix.Sendto(fd, frame, 0, &unix.SockaddrLinklayer{Ifindex: client.Attrs().Index}); err != nil {
			t.Fatal(err)
		}

		entries, err := vnl.NeighList(access.Attrs().Index, unix.AF_BRIDGE)
		if err != nil {
			t.Fatal(err)
		}

		for _, entry := range entries {
			if entry.HardwareAddr.String() == "02:00:00:00:00:2a" && entry.Vlan == 200 {
				return
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	entries, _ := vnl.NeighList(0, unix.AF_BRIDGE)
	t.Logf("FDB: %+v", entries)
	links, _ := vnl.LinkList()
	for _, l := range links {
		t.Logf("link %s attrs %+v", l.Attrs().Name, l.Attrs())
	}

	t.Fatal("VM MAC was not learned on transport VLAN 200")
}

func assertVNetFRR(t *testing.T) {
	t.Helper()
	lo, err := vnl.LinkByName("lo")
	if err != nil {
		t.Fatal(err)
	}

	addr, err := vnl.ParseAddr("192.0.2.10/32")
	if err != nil {
		t.Fatal(err)
	}

	if err := vnl.AddrAdd(lo, addr); err != nil {
		t.Fatal(err)
	}

	if err := vnl.LinkSetUp(lo); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	conf := filepath.Join(dir, "frr.conf")
	body := "hostname test\nrouter bgp 65000\n bgp router-id 192.0.2.10\n address-family l2vpn evpn\n  advertise-all-vni\n exit-address-family\n"
	if err := os.WriteFile(conf, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	for _, daemon := range []string{"zebra", "bgpd"} {
		startVNetFRR(t, dir, conf, daemon)
	}

	var output []byte
	for i := 0; i < 100; i++ {
		output, err = exec.Command("vtysh", "--vty_socket", dir, "-c", "show evpn vni", "-c", "show bgp l2vpn evpn route type macip").CombinedOutput()
		if err == nil && strings.Contains(string(output), "10200") && strings.Contains(string(output), "02:00:00:00:00:2a") {
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("FRR did not discover VNI and advertise VM MAC: %s, %v", output, err)
}

func startVNetFRR(t *testing.T, dir, conf, daemon string) {
	t.Helper()
	cmd := exec.Command("/usr/lib/frr/"+daemon, "-f", conf, "-i", filepath.Join(dir, daemon+".pid"), "-z", filepath.Join(dir, "zserv.api"), "--vty_socket", dir, "-u", "root", "-g", "root")
	logFile, err := os.Create(filepath.Join(dir, daemon+".log"))
	if err != nil {
		t.Fatal(err)
	}

	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		t.Fatal(err)
	}

	t.Cleanup(func() { stopVNetFRR(t, cmd, logFile, dir, daemon) })
}

func stopVNetFRR(t *testing.T, cmd *exec.Cmd, logFile *os.File, dir, daemon string) {
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	_ = logFile.Close()
	if t.Failed() {
		data, _ := os.ReadFile(filepath.Join(dir, daemon+".log"))
		t.Logf("%s: %s", daemon, data)
	}
}
