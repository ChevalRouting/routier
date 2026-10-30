package configtest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/tests/testkit"
)

func TestValidateDNSTransfers(t *testing.T) {
	cases := map[string]dnsCase{
		"source addresses":        {yaml: "primaries: [192.0.2.1]\n        transfer_source: 198.51.100.11\n        transfer_source_v6: fd00:42::1"},
		"automatic sources":       {yaml: "primaries: [192.0.2.1]\n        transfer_source: '*'\n        transfer_source_v6: '*'"},
		"IPv6 in IPv4 source":     {yaml: "primaries: [192.0.2.1]\n        transfer_source: fd00:42::1", wantErr: "transfer_source:"},
		"IPv4 in IPv6 source":     {yaml: "primaries: [192.0.2.1]\n        transfer_source_v6: 198.51.100.11", wantErr: "transfer_source_v6:"},
		"mapped IPv4 source":      {yaml: "primaries: [192.0.2.1]\n        transfer_source: ::ffff:198.51.100.11", wantErr: "transfer_source:"},
		"invalid source":          {yaml: "primaries: [192.0.2.1]\n        transfer_source: invalid", wantErr: "transfer_source:"},
		"primary transfer source": {yaml: "transfer_source: 198.51.100.11", wantErr: "require a secondary zone"},
		"primary notify sources":  {yaml: "notify_source: 198.51.100.11\n        notify_source_v6: fd00:42::1"},
		"bad notify source":       {yaml: "notify_source_v6: 198.51.100.11", wantErr: "notify_source_v6:"},
		"primary":                 {yaml: "secondaries: [192.0.2.2]\n        transfer_key: {name: transfer.example.net, secret: AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=}"},
		"bad address":             {yaml: "secondaries: [invalid]", wantErr: "secondaries[0]"},
		"bad secret":              {yaml: "transfer_key: {name: transfer.example.net, secret: invalid}", wantErr: "transfer_key.secret"},
		"bad key name":            {yaml: "transfer_key: {name: 'bad\"key', secret: AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=}", wantErr: "transfer_key.name"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) { validateDNSTransferCase(t, tc) })
	}
}

func validateDNSTransferCase(t *testing.T, tc dnsCase) {
	records := "        records: [{name: ns, type: A, value: 192.0.2.1}]\n"
	if strings.Contains(tc.yaml, "primaries:") {
		records = ""
	}

	yaml := dnsServerBase + "    zones:\n      - name: example.net\n        nameservers: [ns.example.net]\n        soa: {email: hostmaster@example.net}\n" + records + "        " + tc.yaml + "\n"
	_, err := config.LoadAndValidate(testkit.WriteConfig(t, t.TempDir(), yaml), false)
	if tc.wantErr == "" {
		if err != nil {
			t.Fatal(err)
		}

		return
	}

	if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
		t.Fatalf("wanted %q, got %v", tc.wantErr, err)
	}
}
