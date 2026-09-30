package apitest

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ChevalRouting/routier/tests/testkit"
)

const dnsAPIConfig = `version: v3.0.0
hostname: rtr1
interfaces:
  lan:
    select: name=eth0
    addresses: ["10.0.0.1/24"]
dns:
  nameservers: [127.0.0.1]
  search: [example.net]
  server:
    enabled: true
    listen: [iface(lan)]
    allow_from: ["10.0.0.0/24"]
    allow_inbound: [lan]
    upstreams: [1.1.1.1]
    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { primary: ns1.home.arpa., email: hostmaster@home.arpa }
        records:
          - { name: "@",   type: A, value: 10.0.0.1 }
          - { name: ns1,   type: A, value: 10.0.0.1 }
`

func TestAPIDNSRequiresAuth(t *testing.T) {
	n := testkit.NewNode(t, dnsAPIConfig)

	resp, err := http.Get(n.URL + "/api/dns")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", resp.StatusCode)
	}
}

func TestAPIDNSStatsWithoutDaemon(t *testing.T) {
	n := testkit.NewNode(t, dnsAPIConfig)

	status, raw := n.Do(t, http.MethodGet, "/api/dns/stats", nil)
	if status != http.StatusOK {
		t.Fatalf("stats must return 200 even with no daemon, got %d: %s", status, raw)
	}

	body := testkit.DecodeData[map[string]any](t, raw)

	services, ok := body["services"].([]any)
	if !ok || len(services) == 0 {
		t.Fatalf("expected a services list, got %v", body)
	}

	first, ok := services[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected services shape: %v", services)
	}

	if first["service"] != "named" {
		t.Fatalf("expected the named service state, got %v", first)
	}

	if first["running"] != false {
		t.Fatalf("expected running=false with no daemon, got %v", first)
	}
}

func TestAPIDNSZonesFallBackToDeclared(t *testing.T) {
	n := testkit.NewNode(t, dnsAPIConfig)

	status, raw := n.Do(t, http.MethodGet, "/api/dns/zones", nil)
	if status != http.StatusOK {
		t.Fatalf("zones %d: %s", status, raw)
	}

	zones := testkit.DecodeData[[]map[string]any](t, raw)
	if len(zones) != 1 {
		t.Fatalf("expected one zone, got %v", zones)
	}

	if zones[0]["name"] != "home.arpa" {
		t.Fatalf("unexpected zone: %v", zones[0])
	}

	if zones[0]["answered"] != false {
		t.Fatalf("no daemon means the zone cannot be answered: %v", zones[0])
	}

	records, ok := zones[0]["records"].([]any)
	if !ok || len(records) != 2 {
		t.Fatalf("expected the declared records, got %v", zones[0]["records"])
	}
}

func TestAPIDNSUnknownZoneIs404(t *testing.T) {
	n := testkit.NewNode(t, dnsAPIConfig)

	status, _ := n.Do(t, http.MethodGet, "/api/dns/zones/nope.example", nil)
	if status != http.StatusNotFound {
		t.Fatalf("expected 404 for an unconfigured zone, got %d", status)
	}
}

func TestAPIDNSDisabledIs404(t *testing.T) {
	n := testkit.NewNode(t, testkit.MinimalConfig)

	status, _ := n.Do(t, http.MethodGet, "/api/dns", nil)
	if status != http.StatusNotFound {
		t.Fatalf("expected 404 with no dns server configured, got %d", status)
	}
}

func TestAPIConfigDNSSectionExposesServer(t *testing.T) {
	n := testkit.NewNode(t, dnsAPIConfig)

	status, raw := n.Do(t, http.MethodGet, "/api/config/dns", nil)
	if status != http.StatusOK {
		t.Fatalf("get dns section %d: %s", status, raw)
	}

	body := testkit.DecodeData[map[string]any](t, raw)
	if _, ok := body["server"]; !ok {
		t.Fatalf("dns section must expose the server subtree: %v", body)
	}

	status, raw = n.Do(t, http.MethodGet, "/api/config/dns_server", nil)
	if status != http.StatusOK {
		t.Fatalf("get dns_server section %d: %s", status, raw)
	}

	server := testkit.DecodeData[map[string]any](t, raw)
	if server["enabled"] != true {
		t.Fatalf("expected the enabled server: %v", server)
	}
}

func TestAPIPutDNSServerKeepsResolvConf(t *testing.T) {
	n := testkit.NewNode(t, dnsAPIConfig)

	body := `{"enabled":true,"mode":"forwarder","listen":["iface(lan)"],` +
		`"allow_from":["10.0.0.0/24"],"upstreams":["9.9.9.9"]}`

	status, raw := n.Do(t, http.MethodPut, "/api/config/dns_server", json.RawMessage(body))
	if status != http.StatusOK {
		t.Fatalf("put dns_server %d: %s", status, raw)
	}

	status, raw = n.Do(t, http.MethodGet, "/api/config/dns", nil)
	if status != http.StatusOK {
		t.Fatalf("get dns %d: %s", status, raw)
	}

	dns := testkit.DecodeData[map[string]any](t, raw)

	servers, ok := dns["nameservers"].([]any)
	if !ok || len(servers) != 1 || servers[0] != "127.0.0.1" {
		t.Fatalf("nameservers must survive a dns_server put, got %v", dns["nameservers"])
	}

	search, ok := dns["search"].([]any)
	if !ok || len(search) != 1 || search[0] != "example.net" {
		t.Fatalf("search must survive a dns_server put, got %v", dns["search"])
	}

	server, ok := dns["server"].(map[string]any)
	if !ok || server["mode"] != "forwarder" {
		t.Fatalf("server subtree was not replaced: %v", dns["server"])
	}
}

func TestAPIPutDNSKeepsServerWhenOmitted(t *testing.T) {
	n := testkit.NewNode(t, dnsAPIConfig)

	status, raw := n.Do(t, http.MethodPut, "/api/config/dns",
		json.RawMessage(`{"nameservers":["8.8.8.8"],"search":["other.net"]}`))
	if status != http.StatusOK {
		t.Fatalf("put dns %d: %s", status, raw)
	}

	status, raw = n.Do(t, http.MethodGet, "/api/config/dns", nil)
	if status != http.StatusOK {
		t.Fatalf("get dns %d: %s", status, raw)
	}

	dns := testkit.DecodeData[map[string]any](t, raw)
	if _, ok := dns["server"].(map[string]any); !ok {
		t.Fatalf("a dns put without a server key must not wipe dns.server, got %v", dns)
	}
}

func TestAPIPutDNSServerRejectsInvalidZone(t *testing.T) {
	n := testkit.NewNode(t, dnsAPIConfig)

	body := `{"enabled":true,"mode":"authoritative","listen":["iface(lan)"],` +
		`"allow_from":["10.0.0.0/24"],"zones":[{"name":"bad.example",` +
		`"records":[{"name":"@","type":"A","value":"not-an-ip"}]}]}`

	status, _ := n.Do(t, http.MethodPut, "/api/config/dns_server", json.RawMessage(body))
	if status == http.StatusOK {
		t.Fatal("expected an invalid zone to be rejected by validation")
	}
}

func TestAPINftVarsIncludeDNS(t *testing.T) {
	n := testkit.NewNode(t, dnsAPIConfig)

	status, raw := n.Do(t, http.MethodGet, "/api/config/nftables/vars", nil)
	if status != http.StatusOK {
		t.Fatalf("nft vars %d: %s", status, raw)
	}

	vars := testkit.DecodeData[[]map[string]any](t, raw)

	seen := map[string]bool{}
	for _, v := range vars {
		if name, ok := v["name"].(string); ok {
			seen[name] = true
		}
	}

	for _, want := range []string{"dns_allow_from", "dns_listen"} {
		if !seen[want] {
			t.Errorf("expected the %s nft variable", want)
		}
	}
}
