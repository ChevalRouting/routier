package rendertest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
)

func TestDNSTransferPolicy(t *testing.T) {
	for _, view := range []bool{false, true} {
		zone := config.DNSZone{Name: "example.net", Nameservers: []string{"ns.example.net"}, SOA: &config.DNSSOA{Email: "hostmaster@example.net"}, Records: []config.DNSRecord{{Name: "ns", Type: "A", Value: "192.0.2.1"}}, NotifySource: "198.51.100.11", NotifySourceV6: "fd00:42::1", Secondaries: []string{"192.0.2.2"}, TransferKey: &config.DNSTransferKey{Name: "transfer.example.net", Secret: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}}
		server := &config.DNSServer{Enabled: true, Listen: []string{"127.0.0.1"}, AllowFrom: []string{"192.0.2.0/24"}, Zones: []config.DNSZone{zone}}
		if view {
			server.Views = []config.DNSView{{Name: "internal", Zones: server.Zones}}
			server.Zones = nil
		}

		outputs, err := render.All(&config.Config{Hostname: "gw", DNS: &config.DNS{Server: server}})
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, output := range outputs {
			if output.Name != render.NamedConfName {
				continue
			}

			found = true
			for _, want := range []string{`notify-source 198.51.100.11;`, `notify-source-v6 fd00:42::1;`, `algorithm hmac-sha256;`, `allow-transfer { { !{ !192.0.2.2; any; }; key "transfer.example.net"; }; };`, `also-notify { 192.0.2.2 key "transfer.example.net"; };`, `notify explicit;`} {
				if !strings.Contains(output.Content, want) {
					t.Errorf("missing %q in config: %s", want, output.Content)
				}
			}
		}

		if !found {
			t.Fatal("named config missing")
		}
	}
}

func TestAdvanceZoneSerial(t *testing.T) {
	current := "@ IN SOA ns.example. hostmaster.example. ( 4294967295 3600 600 604800 300 )\n@ IN A 192.0.2.1\n"
	unchanged := strings.Replace(current, "4294967295", "123", 1)
	if got := render.AdvanceZoneSerial(current, unchanged); got != current {
		t.Fatalf("unchanged zone rewritten: %s", got)
	}

	changed := strings.Replace(unchanged, "192.0.2.1", "192.0.2.2", 1)
	if got := render.AdvanceZoneSerial(current, changed); !strings.Contains(got, "( 0 3600") || !strings.Contains(got, "192.0.2.2") {
		t.Fatalf("serial did not wrap: %s", got)
	}
}

func TestDNSSecondaryTransferRendering(t *testing.T) {
	zone := config.DNSZone{Name: "example.net", TransferSource: "198.51.100.11", TransferSourceV6: "fd00:42::1", Primaries: []string{"192.0.2.1"}, TransferKey: &config.DNSTransferKey{Name: "transfer.example.net", Secret: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}}
	outputs, err := render.All(&config.Config{Hostname: "gw", DNS: &config.DNS{Server: &config.DNSServer{Enabled: true, Listen: []string{"127.0.0.1"}, AllowFrom: []string{"192.0.2.0/24"}, Zones: []config.DNSZone{zone}}}})
	if err != nil {
		t.Fatal(err)
	}

	for _, output := range outputs {
		if output.Name == render.NamedConfName {
			for _, want := range []string{`transfer-source 198.51.100.11;`, `transfer-source-v6 fd00:42::1;`, `type secondary;`, `192.0.2.1 key "transfer.example.net";`, `file "/var/bind/example.net.zone";`} {
				if !strings.Contains(output.Content, want) {
					t.Errorf("missing %q", want)
				}
			}
		}

		if strings.HasPrefix(output.Name, render.NamedZoneName) {
			t.Fatal("secondary must not render a local zone file")
		}
	}
}
