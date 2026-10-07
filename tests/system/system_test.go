package systemtest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/daemon/keepalived"
	"github.com/ChevalRouting/routier/pkg/daemon/vtysh"
	"github.com/ChevalRouting/routier/pkg/net/iproute"
	"github.com/ChevalRouting/routier/pkg/telemetry/conntrack"
)

func TestConntrackParsing(t *testing.T) {
	restore := conntrack.SetCommandRunner(conntrackParsingHandler)
	defer restore()

	count, err := conntrack.Count()
	if err != nil || count != 42 {
		t.Fatalf("count = %d, err %v", count, err)
	}

	bd, err := conntrack.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if bd.ByProto["tcp"] != 2 || bd.ByProto["udp"] != 1 {
		t.Fatalf("byProto = %+v", bd.ByProto)
	}

	if bd.TCPStates["ESTABLISHED"] != 1 || bd.TCPStates["TIME_WAIT"] != 1 {
		t.Fatalf("tcpStates = %+v", bd.TCPStates)
	}
}

func TestIProuteParsing(t *testing.T) {
	restore := iproute.SetCommandRunner(iProuteParsingHandler)
	defer restore()

	routes := iproute.ShowRoutes()
	if len(routes) != 1 || routes[0].Dst != "10.0.0.0/24" || routes[0].Protocol != "routier" {
		t.Fatalf("routes = %+v", routes)
	}

	neighbors := iproute.ShowNeighbors()
	if len(neighbors) != 1 || neighbors[0].State != "REACHABLE" || neighbors[0].LLAddr != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("neighbors = %+v", neighbors)
	}
}

func TestVtyshPassthrough(t *testing.T) {
	restore := vtysh.SetRunner(func(cmd string) ([]byte, error) {
		return []byte(`{"cmd":"` + cmd + `"}`), nil
	})
	defer restore()

	out, err := vtysh.BGPSummaryJSON()
	if err != nil || !strings.Contains(string(out), "show bgp summary json") {
		t.Fatalf("bgp summary = %s, err %v", out, err)
	}

	out, err = vtysh.RoutesJSON("ip")
	if err != nil || !strings.Contains(string(out), "show ip route json") {
		t.Fatalf("routes = %s, err %v", out, err)
	}
}

func TestKeepalivedParseStates(t *testing.T) {
	dump := `------< VRRP Topology >------
 VRRP Instance = VI_lan_10
   State = MASTER
   Master router = 10.0.0.1
 Peers:
   10.0.0.3 received message at priority 100

 VRRP Instance = VI_wan_20
   State = BACKUP
   Master router = 198.51.100.1
`

	states := keepalived.ParseStates(strings.NewReader(dump))
	if len(states) != 2 {
		t.Fatalf("expected 2 instances, got %+v", states)
	}

	if states["VI_lan_10"].State != "MASTER" || states["VI_lan_10"].MasterIP != "10.0.0.1" {
		t.Fatalf("VI_lan_10 = %+v", states["VI_lan_10"])
	}

	if states["VI_wan_20"].State != "BACKUP" {
		t.Fatalf("VI_wan_20 = %+v", states["VI_wan_20"])
	}
}

func conntrackParsingHandler(name string, args ...string) ([]byte, error) {
	if len(args) == 1 && args[0] == "-C" {
		return []byte("42\n"), nil
	}

	return []byte(
		"tcp      6 431999 ESTABLISHED src=10.0.0.1 dst=10.0.0.2\n" +
			"tcp      6 120 TIME_WAIT src=10.0.0.3 dst=10.0.0.4\n" +
			"udp      17 29 src=10.0.0.5 dst=10.0.0.6\n"), nil
}

func iProuteParsingHandler(name string, args ...string) ([]byte, error) {
	v6 := false
	for _, a := range args {
		if a == "-6" {
			v6 = true
		}
	}

	if v6 {
		return []byte("[]"), nil
	}

	for _, a := range args {
		if a == "neighbor" {
			return []byte(`[{"dst":"10.0.0.2","dev":"eth0","lladdr":"aa:bb:cc:dd:ee:ff","state":["REACHABLE"]}]`), nil
		}
	}

	return []byte(`[{"dst":"10.0.0.0/24","dev":"eth0","protocol":"100","metric":100}]`), nil
}
