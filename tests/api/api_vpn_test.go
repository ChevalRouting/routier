package apitest

import (
	"net/http"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/tests/testkit"
)

func tunnelTestConfig() map[string]*config.Interface {
	return map[string]*config.Interface{
		"gre-office": {Type: "tunnel", Mode: "gre", Local: "192.0.2.1", Remote: "192.0.2.2"},
	}
}

func TestVPNSectionEditorsPreserveOtherProtocols(t *testing.T) {
	n := testkit.NewNode(t, testkit.MinimalConfig)
	status, raw := n.Do(t, http.MethodPut, "/api/config/interfaces", tunnelTestConfig())
	if status != http.StatusOK {
		t.Fatalf("put VPN %d: %s", status, raw)
	}

	wireguard := map[string]*config.Wireguard{"wg0": {PrivateKey: "test-key"}}
	status, raw = n.Do(t, http.MethodPut, "/api/config/vpn.wireguard", wireguard)
	if status != http.StatusOK {
		t.Fatalf("put WireGuard %d: %s", status, raw)
	}

	status, raw = n.Do(t, http.MethodGet, "/api/config/vpn", nil)
	if status != http.StatusOK {
		t.Fatalf("get VPN %d: %s", status, raw)
	}

	vpn := testkit.DecodeData[config.VPNConfig](t, raw)
	if vpn.Wireguard["wg0"] == nil {
		t.Fatalf("section edit lost another VPN protocol: %+v", vpn)
	}

	status, raw = n.Do(t, http.MethodGet, "/api/config/tunnels", nil)
	if status != http.StatusOK || testkit.DecodeData[map[string]*config.Tunnel](t, raw)["gre-office"] == nil {
		t.Fatalf("legacy alias must read interface tunnels %d: %s", status, raw)
	}
}

func TestSessionVPNRoutes(t *testing.T) {
	n := testkit.NewNode(t, testkit.MinimalConfig)
	status, raw := n.Do(t, http.MethodPost, "/api/v1/sessions", nil)
	if status != http.StatusOK {
		t.Fatalf("create session %d: %s", status, raw)
	}

	session := testkit.DecodeData[v1SessionInfo](t, raw)
	prefix := "/api/v1/sessions/" + session.ID
	status, raw = n.Do(t, http.MethodPut, prefix+"/interfaces", tunnelTestConfig())
	if status != http.StatusOK {
		t.Fatalf("put session VPN %d: %s", status, raw)
	}

	for _, path := range []string{"/vpn/tunnels/gre-office", "/tunnels/gre-office"} {
		status, raw = n.Do(t, http.MethodGet, prefix+path, nil)
		if status != http.StatusOK || testkit.DecodeData[config.Tunnel](t, raw).Mode != "gre" {
			t.Fatalf("VPN route %s %d: %s", path, status, raw)
		}
	}
}

func TestLegacyTunnelSessionEditsCanonicalInterfaces(t *testing.T) {
	n := testkit.NewNode(t, testkit.MinimalConfig)
	status, raw := n.Do(t, http.MethodPost, "/api/v1/sessions", nil)
	if status != http.StatusOK {
		t.Fatalf("create session %d: %s", status, raw)
	}

	session := testkit.DecodeData[v1SessionInfo](t, raw)
	prefix := "/api/v1/sessions/" + session.ID
	tunnel := &config.Tunnel{Mode: "gre", Local: "192.0.2.1", Remote: "192.0.2.2"}
	status, raw = n.Do(t, http.MethodPut, prefix+"/tunnels/gre-office", tunnel)
	if status != http.StatusOK {
		t.Fatalf("create legacy tunnel %d: %s", status, raw)
	}

	status, raw = n.Do(t, http.MethodGet, prefix+"/interfaces/gre-office", nil)
	iface := testkit.DecodeData[config.Interface](t, raw)
	if status != http.StatusOK || iface.Type != "tunnel" || iface.Remote != tunnel.Remote {
		t.Fatalf("legacy tunnel did not persist as interface %d: %s", status, raw)
	}

	tunnel.Remote = "192.0.2.3"
	status, raw = n.Do(t, http.MethodPut, prefix+"/tunnels/gre-office", tunnel)
	if status != http.StatusOK {
		t.Fatalf("update legacy tunnel %d: %s", status, raw)
	}

	status, raw = n.Do(t, http.MethodGet, prefix+"/interfaces/gre-office", nil)
	if status != http.StatusOK || testkit.DecodeData[config.Interface](t, raw).Remote != tunnel.Remote {
		t.Fatalf("legacy update did not persist %d: %s", status, raw)
	}

	status, raw = n.Do(t, http.MethodDelete, prefix+"/tunnels/gre-office", nil)
	if status != http.StatusOK {
		t.Fatalf("delete legacy tunnel %d: %s", status, raw)
	}

	status, raw = n.Do(t, http.MethodGet, prefix+"/interfaces/gre-office", nil)
	if status != http.StatusNotFound {
		t.Fatalf("legacy delete left the interface %d: %s", status, raw)
	}
}
