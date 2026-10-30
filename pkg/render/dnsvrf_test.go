package render

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestDNSVRFsSeparateStateAndControls(t *testing.T) {
	cfg := &config.Config{Interfaces: map[string]*config.Interface{"red-lan": {VRF: "red", Addresses: []string{"10.0.0.1/24"}}, "blue-lan": {VRF: "blue", Addresses: []string{"10.0.0.1/24"}}}, VRFs: map[string]*config.VRFConfig{"red": {Table: 1001}, "blue": {Table: 1002}}, DNS: &config.DNS{Server: &config.DNSServer{}}}
	cfg.DNS.Server.VRFs = map[string]*config.DNSServer{}
	for _, name := range []string{"red", "blue"} {
		cfg.DNS.Server.VRFs[name] = &config.DNSServer{Enabled: true, Listen: []string{"iface(" + name + "-lan)"}, AllowFrom: []string{"10.0.0.0/24"}, Zones: []config.DNSZone{{Name: "same.test", Nameservers: []string{"ns.example.net"}, SOA: &config.DNSSOA{Email: "hostmaster.same.test"}, Records: []config.DNSRecord{{Name: "host", Type: "A", Value: "10.0.0.42"}}}}}
	}

	outputs, err := All(cfg)
	if err != nil {
		t.Fatal(err)
	}

	found := map[string]bool{}
	for _, output := range outputs {
		if !strings.HasPrefix(output.Name, "bind/vrfs/") {
			continue
		}

		found[output.Dest] = true
		if strings.HasSuffix(output.Dest, "named.conf") && (!strings.Contains(output.Content, "controls {};") || strings.Contains(output.Content, "rndc.key") || strings.Contains(output.Content, "directory \"/var/bind\"")) {
			t.Fatalf("shared state in %s: %s", output.Name, output.Content)
		}
	}

	for _, name := range []string{"red", "blue"} {
		if !found[DNSVRFDirectory(name)+"/named.conf"] || !found[DNSVRFDirectory(name)+"/zones/same.test.zone"] {
			t.Fatalf("missing resolver outputs: %v", found)
		}
	}

	cfg.DNS.Server.VRFs["red"].Listen = []string{"192.0.2.1"}
	if len(config.Validate(cfg, false)) == 0 {
		t.Fatal("foreign listener accepted")
	}
}
