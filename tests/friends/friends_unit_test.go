package friendstest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/auth/identity"
	"github.com/ChevalRouting/routier/pkg/types"
)

func TestFriendsAddRemoveIndex(t *testing.T) {
	cfg := &config.Config{}
	if err := friends.Add(cfg, &config.Friend{Name: "peer-a", URL: "https://198.51.100.1:8080", Token: "t"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	if err := friends.Add(cfg, &config.Friend{Name: "peer-a"}); err == nil {
		t.Fatal("expected duplicate name to error")
	}

	if friends.Index(cfg, "peer-a") != 0 {
		t.Fatalf("index = %d", friends.Index(cfg, "peer-a"))
	}

	if friends.Get(cfg, "missing") != nil {
		t.Fatal("expected nil for missing friend")
	}

	if _, err := friends.Remove(cfg, "peer-a"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	if len(cfg.Friends) != 0 {
		t.Fatalf("friends not removed: %d", len(cfg.Friends))
	}

	if _, err := friends.Remove(cfg, "peer-a"); err == nil {
		t.Fatal("expected remove of missing friend to error")
	}
}

func TestFriendsPinIdentityTOFU(t *testing.T) {
	f := &config.Friend{Name: "peer-a"}
	if err := friends.PinIdentity(f, "SHA256:aaa", "edpub", "xpub"); err != nil {
		t.Fatalf("first pin: %v", err)
	}

	if f.Identity.Fingerprint != "SHA256:aaa" || f.Identity.PublicKey != "edpub" || f.Identity.X25519PublicKey != "xpub" {
		t.Fatalf("not pinned: %+v", f.Identity)
	}

	if err := friends.PinIdentity(f, "SHA256:aaa", "edpub", "xpub"); err != nil {
		t.Fatalf("matching pin: %v", err)
	}

	if err := friends.PinIdentity(f, "SHA256:bbb", "edpub", "xpub"); err == nil {
		t.Fatal("expected mismatch to error")
	}

	if err := friends.PinIdentity(f, "", "", ""); err == nil {
		t.Fatal("expected empty fingerprint to error")
	}
}

func TestFriendsTaggedSectionsAndRemove(t *testing.T) {
	cfg := &config.Config{Wireguard: map[string]*config.Wireguard{
		"wg-fr-peer-b": {Friend: "peer-b"},
		"wg-fr-peer-c": {Friend: "peer-c"},
		"wg-manual":    {},
	}}

	tagged := friends.TaggedSections(cfg, "peer-b")
	if len(tagged) != 1 || tagged[0].Section != "wireguard" || tagged[0].Key != "wg-fr-peer-b" {
		t.Fatalf("tagged = %+v", tagged)
	}

	friends.RemoveTagged(cfg, "peer-b")
	if _, ok := cfg.Wireguard["wg-fr-peer-b"]; ok {
		t.Fatal("tagged interface not removed")
	}

	if _, ok := cfg.Wireguard["wg-fr-peer-c"]; !ok {
		t.Fatal("removed unrelated friend's interface")
	}

	if _, ok := cfg.Wireguard["wg-manual"]; !ok {
		t.Fatal("removed untagged interface")
	}
}

func unitHelloServer(t *testing.T, id *identity.Identity, hostname, token string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var data any
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/friends/hello":
			data = types.FriendsHello{
				Hostname:            hostname,
				IdentityFingerprint: id.Fingerprint(),
				IdentityPublicKey:   id.PublicKeyBase64(),
				Version:             "test",
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}

		_ = json.NewEncoder(w).Encode(types.Response[any]{AppCode: 200, Result: &data})
	}))
}

func mustIdentity(t *testing.T) *identity.Identity {
	t.Helper()
	id, err := identity.LoadOrCreate(t.TempDir() + "/id")
	if err != nil {
		t.Fatalf("identity: %v", err)
	}

	return id
}

func TestFriendsClientHelloAndPreview(t *testing.T) {
	remote := mustIdentity(t)
	srv := unitHelloServer(t, remote, "peer-a", "shared-token")
	defer srv.Close()

	client := friends.NewClient(srv.URL, "shared-token", false)
	hello, err := client.Hello(context.Background())
	if err != nil {
		t.Fatalf("hello: %v", err)
	}

	if hello.Hostname != "peer-a" || hello.IdentityFingerprint != remote.Fingerprint() {
		t.Fatalf("unexpected hello: %+v", hello)
	}

	cfg := &config.Config{}
	if p := friends.Preview(context.Background(), client, cfg, "SHA256:self"); !p.Reachable || p.IsSelf || p.AlreadyExists {
		t.Fatalf("unexpected preview: %+v", p)
	}

	if sp := friends.Preview(context.Background(), client, cfg, remote.Fingerprint()); !sp.IsSelf {
		t.Fatalf("expected self detection: %+v", sp)
	}

	cfg.Friends = append(cfg.Friends, &config.Friend{Name: "dup", Identity: config.FriendIdentity{Fingerprint: remote.Fingerprint()}})
	if dp := friends.Preview(context.Background(), client, cfg, ""); !dp.AlreadyExists {
		t.Fatalf("expected already-exists: %+v", dp)
	}
}

func TestFriendsClientBadToken(t *testing.T) {
	remote := mustIdentity(t)
	srv := unitHelloServer(t, remote, "peer-a", "shared-token")
	defer srv.Close()

	client := friends.NewClient(srv.URL, "wrong", false)
	if _, err := client.Hello(context.Background()); err == nil {
		t.Fatal("expected unauthorized error")
	}
}
