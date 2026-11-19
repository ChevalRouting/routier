package render

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestRenderRADVDNAT64Prefix(t *testing.T) {
	cfg := &config.Config{
		Hostname:   "edge",
		Interfaces: map[string]*config.Interface{"lan": {Select: "name=eth1"}},
		Routing: &config.Routing{RADVD: &config.RADVDConfig{Interfaces: map[string]*config.RADVDInterface{"lan": {
			AdvSendAdvert: true,
			NAT64Prefixes: []config.RADVDNAT64{{Prefix: "64:ff9b::/96", Lifetime: 1800}},
		}}}},
	}

	out, err := All(cfg)
	if err != nil {
		t.Fatal(err)
	}

	for _, o := range out {
		if o.Name != "radvd/radvd.conf" {
			continue
		}

		if !strings.Contains(o.Content, "    nat64prefix 64:ff9b::/96 {\n        AdvValidLifetime 1800;\n    };") {
			t.Fatalf("nat64prefix missing:\n%s", o.Content)
		}

		return
	}

	t.Fatal("radvd.conf not rendered")
}
