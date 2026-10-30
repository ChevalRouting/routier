package render

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestAnycastGatewayTableKeepsTheGatewayLocal(t *testing.T) {
	cfg := &config.Config{Interfaces: map[string]*config.Interface{
		"fabric": {Type: "vxlan-bridge"},
		"campus": {Type: "vnet", Select: "fabric", VNet: &config.VNet{VNI: 10040, VLAN: 40, Gateway: &config.VNetGateway{IPs: []string{"10.50.0.1", "fd50::1"}}}},
		"office": {Type: "vnet", Select: "fabric", VNet: &config.VNet{VNI: 10030, VLAN: 30}},
	}}
	if got := renderAnycastGatewayTable(&config.Config{Interfaces: map[string]*config.Interface{"office": cfg.Interfaces["office"]}}); got != "" {
		t.Fatalf("a vnet without gateway rendered a table:\n%s", got)
	}

	got := renderAnycastGatewayTable(cfg)
	fabric := config.VNetLinkName("campus", "b")
	for _, want := range []string{
		"table bridge routier_anycast {",
		"type filter hook forward priority filter; policy accept;",
		"type filter hook output priority filter; policy accept;",
		`iifname "` + fabric + `" ether saddr 02:cb:00:27:38:01 drop`,
		`oifname "` + fabric + `" ether saddr 02:cb:00:27:38:01 drop`,
		`oifname "` + fabric + `" ether daddr 02:cb:00:27:38:01 drop`,
		`oifname "` + fabric + `" arp daddr ip 10.50.0.1 drop`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in\n%s", want, got)
		}
	}

	if strings.Contains(got, config.VNetLinkName("office", "b")) {
		t.Fatalf("office has no gateway:\n%s", got)
	}
}
