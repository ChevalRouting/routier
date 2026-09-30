package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ChevalRouting/routier/pkg/auth/identity"
	"github.com/ChevalRouting/routier/pkg/types"
)

func pollServer(t *testing.T, sender, recipient *identity.Identity, hash string) *httptest.Server {
	t.Helper()

	key, err := sender.SharedKey(recipient.X25519PublicBase64())
	if err != nil {
		t.Fatalf("SharedKey: %v", err)
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/friends/poll" {
			t.Errorf("poll requested %q, want /api/friends/poll (no double slash)", r.URL.Path)
		}

		poll := types.FriendPoll{
			Hello:      types.FriendsHello{Hostname: "peer"},
			Interfaces: []types.FriendInterface{{Name: "eth0", Addresses: []string{"10.0.0.1/24"}}},
			Exports:    []types.FriendVar{{Name: "x", Value: "y"}},
			ConfigHash: hash,
		}

		if r.URL.Query().Get("have") != hash {
			poll.Config = json.RawMessage(`{"hostname":"peer","version":"v3.0.0"}`)
		}

		env, _ := json.Marshal(types.Response[types.FriendPoll]{Result: &poll})
		sealed, serr := identity.SealShared(key, env)
		if serr != nil {
			http.Error(w, serr.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", types.SharedSealedContentType)
		w.Write(sealed)
	}))
}

func TestPollDecryptsAndGatesConfig(t *testing.T) {
	sender, _ := identity.LoadOrCreate(filepath.Join(t.TempDir(), "sender"))
	recipient, _ := identity.LoadOrCreate(filepath.Join(t.TempDir(), "recipient"))

	const hash = "abc123"
	srv := pollServer(t, sender, recipient, hash)
	defer srv.Close()

	c := New(&Config{URL: srv.URL})
	key, _ := recipient.SharedKey(sender.X25519PublicBase64())
	c.SetSharedKey(key)

	p, err := c.Poll(context.Background(), "")
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}

	if p.Hello.Hostname != "peer" || len(p.Interfaces) != 1 || len(p.Exports) != 1 {
		t.Fatalf("unexpected poll payload: %+v", p)
	}

	if p.ConfigHash != hash || len(p.Config) == 0 {
		t.Fatalf("first poll should carry the full config, got hash=%q len=%d", p.ConfigHash, len(p.Config))
	}

	p2, err := c.Poll(context.Background(), hash)
	if err != nil {
		t.Fatalf("Poll (have hash): %v", err)
	}

	if len(p2.Config) != 0 {
		t.Fatalf("config should be omitted when the hash matches, got %d bytes", len(p2.Config))
	}
}

func TestPollRejectsSharedResponseWithoutKey(t *testing.T) {
	sender, _ := identity.LoadOrCreate(filepath.Join(t.TempDir(), "sender"))
	recipient, _ := identity.LoadOrCreate(filepath.Join(t.TempDir(), "recipient"))

	srv := pollServer(t, sender, recipient, "h")
	defer srv.Close()

	c := New(&Config{URL: srv.URL})
	if _, err := c.Poll(context.Background(), ""); err == nil {
		t.Fatal("expected error when a shared-sealed response arrives without a shared key")
	}
}
