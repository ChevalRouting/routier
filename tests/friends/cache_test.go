package friendstest

import (
	"net/http"
	"testing"

	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/ChevalRouting/routier/tests/testkit"
)

func TestFriendInterfacesCacheFallback(t *testing.T) {
	const token = "cache-token"

	b := testkit.NewNode(t, `version: v3.0.0
hostname: node-b
interfaces:
  lan:
    select: name=eth0
    addresses: ["10.0.0.2/24"]
friends:
  - name: node-a
    url: http://127.0.0.1:1
    token: `+token+`
    enabled: true
`)

	a := testkit.NewNode(t, `version: v3.0.0
hostname: node-a
friends:
  - name: node-b
    url: `+b.URL+`
    token: `+token+`
    tls_skip_verify: true
    enabled: true
`)

	status, raw := a.Do(t, http.MethodGet, "/api/friends/node-b/interfaces", nil)
	if status != http.StatusOK {
		t.Fatalf("live interfaces %d: %s", status, raw)
	}

	if !hasIface(testkit.DecodeData[[]types.FriendInterface](t, raw), "lan") {
		t.Fatalf("live interfaces missing lan: %s", raw)
	}

	b.Server.Close()

	status, raw = a.Do(t, http.MethodGet, "/api/friends/node-b/interfaces", nil)
	if status != http.StatusOK {
		t.Fatalf("cached interfaces %d: %s", status, raw)
	}

	if !hasIface(testkit.DecodeData[[]types.FriendInterface](t, raw), "lan") {
		t.Fatalf("cache fallback missing lan: %s", raw)
	}
}

func hasIface(ifaces []types.FriendInterface, name string) bool {
	for _, i := range ifaces {
		if i.Name == name {
			return true
		}
	}

	return false
}
