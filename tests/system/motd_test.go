package systemtest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/motd"
)

func TestMOTDRender(t *testing.T) {
	routes := motd.RouteSummary{BGP: 3, OSPF: 2, Static: 1, Total: 9}
	bw := &motd.Bandwidth{RxBps: 125000, TxBps: 250000}
	out := motd.Render("1.2.3", []string{"eth0: 10.0.0.1/24"}, routes, bw, "admin", "s3cret")

	for _, want := range []string{
		"1.2.3", "Network:", "eth0: 10.0.0.1/24",
		"9 total", "bgp 3", "ospf 2", "static 1",
		"Traffic (5m avg):", "rx 1.0 Mbps", "tx 2.0 Mbps",
		"username: admin", "password: s3cret",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("motd missing %q in:\n%s", want, out)
		}
	}

	if strings.Contains(motd.Render("1.2.3", nil, motd.RouteSummary{}, nil, "", ""), "username") {
		t.Fatal("did not expect credentials block")
	}
}
