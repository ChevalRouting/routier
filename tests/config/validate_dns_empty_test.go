package configtest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/tests/testkit"
)

func TestValidateDNSPrimaryWithoutRecords(t *testing.T) {
	for _, records := range []string{"", "        records: []\n"} {
		yaml := dnsServerBase + "    zones:\n      - name: example.net\n        nameservers: [ns.example.org.]\n        soa: {email: hostmaster@example.net}\n" + records
		if _, err := config.LoadAndValidate(testkit.WriteConfig(t, t.TempDir(), yaml), false); err != nil {
			t.Fatal(err)
		}
	}
}

func TestValidateDNSEmptyPrimaryStillRequiresGlue(t *testing.T) {
	yaml := dnsServerBase + "    zones:\n      - name: example.net\n        nameservers: [ns.example.net.]\n        soa: {email: hostmaster@example.net}\n"
	_, err := config.LoadAndValidate(testkit.WriteConfig(t, t.TempDir(), yaml), false)
	if err == nil || !strings.Contains(err.Error(), "has no A or AAAA record") {
		t.Fatalf("expected missing glue error, got %v", err)
	}
}
