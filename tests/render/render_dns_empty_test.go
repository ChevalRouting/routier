package rendertest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/render"
)

const dnsEmptyPrimaryConfig = `version: v5.0.0
hostname: gw
dns:
  server:
    enabled: true
    listen: [127.0.0.1]
    allow_from: [127.0.0.1]
    zones:
      - name: example.net
        nameservers: [ns.example.org.]
        soa: {email: hostmaster@example.net}
`

func TestRenderDNSPrimaryWithoutRecords(t *testing.T) {
	outputs := renderByDest(t, dnsEmptyPrimaryConfig)
	zone := outputs[render.NamedZoneDir+"/example.net.zone"]
	if !strings.Contains(zone, "IN SOA") || !strings.Contains(zone, "IN NS") {
		t.Fatalf("empty primary zone was not rendered with SOA and NS records: %s", zone)
	}

	if _, err := exec.LookPath("named-checkzone"); err != nil {
		t.Skip("named-checkzone not installed")
	}

	path := filepath.Join(t.TempDir(), "example.net.zone")
	if err := os.WriteFile(path, []byte(zone), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := exec.Command("named-checkzone", "example.net", path).CombinedOutput()
	if err != nil {
		t.Fatalf("named-checkzone rejected metadata-only zone: %v\n%s", err, output)
	}
}

func TestRenderDDNSPrimaryWithoutRecords(t *testing.T) {
	yaml := dnsEmptyPrimaryConfig + `dhcp:
  enabled: true
  subnets4:
    - subnet: 10.0.0.0/24
  ddns:
    enabled: true
    domain: hosts.example.net
    reverse: false
    key: AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
`
	outputs := renderByDest(t, yaml)
	shadow := outputs[render.NamedStaticDir+"/example.net.zone"]
	if !strings.Contains(shadow, "IN SOA") || !strings.Contains(shadow, "IN NS") {
		t.Fatalf("record-free DDNS primary must retain its static metadata: %s", shadow)
	}
}
