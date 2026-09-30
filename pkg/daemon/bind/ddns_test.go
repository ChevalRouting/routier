package bind

import (
	"os"
	"strings"
	"testing"
)

const axfrSample = `; <<>> DiG 9.18 <<>> @127.0.0.1 -t AXFR home.arpa
home.arpa.        3600 IN SOA ns.home.arpa. hostmaster.home.arpa. 5 3600 600 604800 300
home.arpa.        3600 IN NS  ns.home.arpa.
laptop.home.arpa. 3600 IN A   192.168.1.50
laptop.home.arpa. 3600 IN DHCID AAIBY2/AuCccgoJbsaxcQc9TUapptP69lOjxfNuVAA2kjEA=
phone.home.arpa.  3600 IN AAAA 2001:db8::5
home.arpa.        3600 IN SOA ns.home.arpa. hostmaster.home.arpa. 5 3600 600 604800 300
;; Query time: 1 msec
`

func testClient() *Client {
	return &Client{
		Resolver:      "127.0.0.1",
		TSIGName:      "routier-ddns",
		TSIGAlgorithm: "hmac-sha256",
		TSIGSecret:    "c2VjcmV0",
	}
}

func TestDynamicRecordsFiltersInfraRecords(t *testing.T) {
	restore := SetRunner(func(name string, args ...string) ([]byte, error) {
		if name != "dig" {
			t.Fatalf("unexpected command %q", name)
		}
		return []byte(axfrSample), nil
	})
	defer restore()

	records, err := testClient().DynamicRecords("home.arpa")
	if err != nil {
		t.Fatalf("DynamicRecords: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("got %d records, want 2: %+v", len(records), records)
	}

	want := map[string]string{
		"laptop.home.arpa.": "192.168.1.50",
		"phone.home.arpa.":  "2001:db8::5",
	}
	for _, rec := range records {
		if want[rec.Name] != rec.Value {
			t.Errorf("record %s = %q, want %q", rec.Name, rec.Value, want[rec.Name])
		}
	}
}

func TestDynamicRecordsTransferFailed(t *testing.T) {
	restore := SetRunner(func(name string, args ...string) ([]byte, error) {
		return []byte("; Transfer failed.\n"), nil
	})
	defer restore()

	if _, err := testClient().DynamicRecords("home.arpa"); err == nil {
		t.Fatal("expected error on transfer failure")
	}
}

func TestDeleteRecordScriptForAddress(t *testing.T) {
	var script string
	restore := SetRunner(func(name string, args ...string) ([]byte, error) {
		if name != "nsupdate" {
			t.Fatalf("unexpected command %q", name)
		}

		data, err := os.ReadFile(args[len(args)-1])
		if err != nil {
			t.Fatalf("read script: %v", err)
		}
		script = string(data)

		return nil, nil
	})
	defer restore()

	if err := testClient().DeleteRecord("home.arpa", "laptop.home.arpa.", "a", "192.168.1.50"); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}

	for _, want := range []string{
		"zone home.arpa\n",
		"update delete laptop.home.arpa. A 192.168.1.50\n",
		"update delete laptop.home.arpa. DHCID\n",
		"send\n",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q\n%s", want, script)
		}
	}
}

func TestDeleteRecordScriptForPTR(t *testing.T) {
	var script string
	restore := SetRunner(func(name string, args ...string) ([]byte, error) {
		data, _ := os.ReadFile(args[len(args)-1])
		script = string(data)
		return nil, nil
	})
	defer restore()

	if err := testClient().DeleteRecord("1.168.192.in-addr.arpa", "50.1.168.192.in-addr.arpa.", "PTR", "laptop.home.arpa."); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}

	if strings.Contains(script, "DHCID") {
		t.Errorf("PTR delete should not touch DHCID:\n%s", script)
	}
}

func TestDeleteRecordRequiresKey(t *testing.T) {
	c := &Client{Resolver: "127.0.0.1"}
	if err := c.DeleteRecord("home.arpa", "laptop.home.arpa.", "A", "192.168.1.50"); err == nil {
		t.Fatal("expected error without a TSIG key")
	}
}
