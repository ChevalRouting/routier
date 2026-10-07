package apitest

import (
	"github.com/ChevalRouting/routier/tests/testkit"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/types"
)

func TestAPIRequiresAuth(t *testing.T) {
	n := testkit.NewNode(t, testkit.MinimalConfig)

	resp, err := http.Get(n.URL + "/api/config")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	defer func(action func() error) { _ = action() }(resp.Body.Close)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", resp.StatusCode)
	}
}

func TestAPIGetConfigAndSection(t *testing.T) {
	n := testkit.NewNode(t, testkit.MinimalConfig)

	status, raw := n.Do(t, http.MethodGet, "/api/config", nil)
	if status != http.StatusOK {
		t.Fatalf("get config %d: %s", status, raw)
	}

	cfg := testkit.DecodeData[map[string]any](t, raw)
	if cfg["hostname"] != "rtr1" {
		t.Fatalf("config hostname = %v", cfg["hostname"])
	}

	status, raw = n.Do(t, http.MethodGet, "/api/config/interfaces", nil)
	if status != http.StatusOK {
		t.Fatalf("get section %d: %s", status, raw)
	}

	if ifaces := testkit.DecodeData[map[string]any](t, raw); len(ifaces) != 1 {
		t.Fatalf("expected 1 interface, got %+v", ifaces)
	}

	if status, _ := n.Do(t, http.MethodGet, "/api/config/bogus", nil); status != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown section, got %d", status)
	}
}

func TestAPIStagingDiffAndDiscard(t *testing.T) {
	n := testkit.NewNode(t, testkit.MinimalConfig)

	status, raw := n.Do(t, http.MethodPut, "/api/config/dns", map[string]any{
		"nameservers": []string{"1.1.1.1"},
	})
	if status != http.StatusOK {
		t.Fatalf("put dns %d: %s", status, raw)
	}

	_, raw = n.Do(t, http.MethodGet, "/api/config/diff", nil)
	diff := testkit.DecodeData[[]types.DiffLine](t, raw)
	found := false
	for _, l := range diff {
		if l.Type == "add" && strings.Contains(l.Text, "1.1.1.1") {
			found = true
		}
	}

	if !found {
		t.Fatalf("diff did not show staged dns change: %+v", diff)
	}

	if status, _ := n.Do(t, http.MethodDelete, "/api/config/staging", nil); status != http.StatusOK {
		t.Fatalf("discard staging failed: %d", status)
	}

	_, raw = n.Do(t, http.MethodGet, "/api/config/diff", nil)
	for _, l := range testkit.DecodeData[[]types.DiffLine](t, raw) {
		if l.Type != "same" {
			t.Fatalf("diff without staging should not show changes: %+v", l)
		}
	}
}

func TestAPIWireguardKeygen(t *testing.T) {
	n := testkit.NewNode(t, testkit.MinimalConfig)

	status, raw := n.Do(t, http.MethodPost, "/api/wireguard/keygen", nil)
	if status != http.StatusOK {
		t.Fatalf("keygen %d: %s", status, raw)
	}

	kp := testkit.DecodeData[types.KeygenResponse](t, raw)
	if kp.PrivateKey == "" || kp.PublicKey == "" {
		t.Fatalf("empty keypair: %+v", kp)
	}

	req, _ := http.NewRequest(http.MethodPost, n.URL+"/api/wireguard/pubkey", strings.NewReader(kp.PrivateKey))
	req.Header.Set("Authorization", "Bearer "+n.JWT)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("pubkey: %v", err)
	}

	defer func(action func() error) { _ = action() }(resp.Body.Close)
	body, _ := io.ReadAll(resp.Body)

	if pk := testkit.DecodeData[types.PubKeyResponse](t, body); pk.PublicKey != kp.PublicKey {
		t.Fatalf("derived pubkey %q != keygen pubkey %q", pk.PublicKey, kp.PublicKey)
	}
}

func TestAPINftablesVarsAndSnapshots(t *testing.T) {
	n := testkit.NewNode(t, testkit.MinimalConfig)

	status, raw := n.Do(t, http.MethodGet, "/api/config/nftables/vars", nil)
	if status != http.StatusOK {
		t.Fatalf("nft vars %d: %s", status, raw)
	}

	vars := testkit.DecodeData[[]map[string]any](t, raw)
	hasLan := false
	for _, v := range vars {
		if name, _ := v["name"].(string); strings.HasPrefix(name, "lan_") {
			hasLan = true
		}
	}

	if !hasLan {
		t.Fatalf("expected lan_* nft vars, got %+v", vars)
	}

	if status, _ := n.Do(t, http.MethodGet, "/api/snapshots", nil); status != http.StatusOK {
		t.Fatalf("snapshots endpoint failed: %d", status)
	}
}
