package friendstest

import (
	"encoding/base64"
	"github.com/ChevalRouting/routier/tests/harness"
	"net/http"
	"testing"

	"github.com/ChevalRouting/routier/pkg/identity"
	"github.com/ChevalRouting/routier/pkg/types"
)

const nodeAConfig = `version: v3.0.0
hostname: node-a
interfaces:
  wan:
    select: name=eth1
    addresses: ["198.51.100.1/24"]
`

func nodeBConfig(token string) string {
	return `version: v3.0.0
hostname: node-b
interfaces:
  lan:
    select: name=eth0
    addresses: ["10.0.0.2/24"]
friends:
  - name: node-a
    url: http://127.0.0.1:1
    token: ` + token + `
    enabled: true
`
}

func TestFriendsHelloChallenge(t *testing.T) {
	a := harness.NewNode(t, nodeAConfig)

	status, raw := a.Do(t, http.MethodGet, "/api/friends/hello?challenge=nonce-123", nil)
	if status != http.StatusOK {
		t.Fatalf("hello status %d: %s", status, raw)
	}

	hello := harness.DecodeData[types.FriendsHello](t, raw)
	if hello.IdentityFingerprint == "" || hello.Signature == "" {
		t.Fatalf("hello missing identity/signature: %+v", hello)
	}

	pub, err := identity.ParsePublicKey(hello.IdentityPublicKey)
	if err != nil {
		t.Fatalf("parse pubkey: %v", err)
	}

	sig, err := base64.StdEncoding.DecodeString(hello.Signature)
	if err != nil {
		t.Fatalf("decode sig: %v", err)
	}

	if !identity.Verify(pub, []byte("nonce-123"), sig) {
		t.Fatal("challenge signature did not verify against advertised identity")
	}

	if identity.Fingerprint(pub) != hello.IdentityFingerprint {
		t.Fatal("advertised fingerprint does not match public key")
	}
}

func TestFriendsPreviewAddPair(t *testing.T) {
	const token = "shared-secret-token"

	a := harness.NewNode(t, nodeAConfig)
	b := harness.NewNode(t, nodeBConfig(token))

	_, helloRaw := a.Do(t, http.MethodGet, "/api/friends/hello", nil)
	aFingerprint := harness.DecodeData[types.FriendsHello](t, helloRaw).IdentityFingerprint

	status, raw := a.Do(t, http.MethodPost, "/api/friends/preview", types.FriendPreviewRequest{
		URL: b.URL, Token: token, TLSSkipVerify: true,
	})
	if status != http.StatusOK {
		t.Fatalf("preview status %d: %s", status, raw)
	}

	preview := harness.DecodeData[types.FriendPreview](t, raw)
	if !preview.Reachable || preview.Hostname != "node-b" || preview.IdentityFingerprint == "" {
		t.Fatalf("unexpected preview: %+v", preview)
	}

	if preview.IsSelf || preview.AlreadyExists {
		t.Fatalf("preview misclassified: %+v", preview)
	}

	status, raw = a.Do(t, http.MethodPost, "/api/friends", types.AddFriendRequest{
		Name: "node-b", URL: b.URL, Token: token, TLSSkipVerify: true,
	})
	if status != http.StatusOK {
		t.Fatalf("add status %d: %s", status, raw)
	}

	info := harness.DecodeData[types.FriendInfo](t, raw)
	if info.Fingerprint != preview.IdentityFingerprint {
		t.Fatalf("added friend fingerprint mismatch: %+v", info)
	}

	if info.Paired {
		t.Fatal("add must not pair; pairing is a separate human-validated step")
	}

	status, raw = a.Do(t, http.MethodPost, "/api/friends/node-b/pair", types.VerifyFriendRequest{
		Fingerprint: preview.IdentityFingerprint,
	})
	if status != http.StatusOK {
		t.Fatalf("pair status %d: %s", status, raw)
	}

	if !harness.DecodeData[types.FriendInfo](t, raw).Paired {
		t.Fatal("expected friend to be paired after verification")
	}

	_, raw = a.Do(t, http.MethodGet, "/api/friends", nil)
	aFriends := harness.DecodeData[[]types.FriendInfo](t, raw)
	if len(aFriends) != 1 || aFriends[0].Name != "node-b" || aFriends[0].Fingerprint == "" {
		t.Fatalf("A friends list wrong: %+v", aFriends)
	}

	_, raw = b.Do(t, http.MethodGet, "/api/friends", nil)
	bFriends := harness.DecodeData[[]types.FriendInfo](t, raw)
	if len(bFriends) != 1 || bFriends[0].Name != "node-a" {
		t.Fatalf("B friends list wrong: %+v", bFriends)
	}

	if bFriends[0].Fingerprint != aFingerprint {
		t.Fatalf("B did not pin A's fingerprint: got %q want %q", bFriends[0].Fingerprint, aFingerprint)
	}

	if bFriends[0].URL != a.URL {
		t.Fatalf("pairing did not update B's callback URL: got %q want %q", bFriends[0].URL, a.URL)
	}

	status, raw = b.Do(t, http.MethodGet, "/api/friends/node-a/config", nil)
	if status != http.StatusOK {
		t.Fatalf("B could not fetch A's config after pairing: %d %s", status, raw)
	}

	remoteCfg := harness.DecodeData[map[string]any](t, raw)
	if remoteCfg["hostname"] != "node-a" {
		t.Fatalf("unexpected remote config from A: %+v", remoteCfg)
	}

	status, raw = a.Do(t, http.MethodGet, "/api/friends/node-b/interfaces", nil)
	if status != http.StatusOK {
		t.Fatalf("remote interfaces status %d: %s", status, raw)
	}

	ifaces := harness.DecodeData[[]types.FriendInterface](t, raw)
	found := false
	for _, i := range ifaces {
		if i.Name == "lan" {
			found = true
		}
	}

	if !found {
		t.Fatalf("expected B's lan interface, got %+v", ifaces)
	}
}

func TestFriendDeleteCascade(t *testing.T) {
	c := harness.NewNode(t, `version: v3.0.0
hostname: node-c
friends:
  - name: peer
    url: http://127.0.0.1:1
    token: tok
    enabled: true
    identity:
      fingerprint: SHA256:abc
wireguard:
  wg-fr-peer:
    friend: peer
    private_key: dGVzdA==
    addresses: ["169.254.0.0/31"]
    peers:
      - public_key: cGVlcg==
        allowed_ips: ["169.254.0.1/32"]
`)

	status, raw := c.Do(t, http.MethodDelete, "/api/friends/peer", nil)
	if status != http.StatusOK {
		t.Fatalf("delete preview status %d: %s", status, raw)
	}

	preview := harness.DecodeData[types.FriendDeleteResult](t, raw)
	if preview.Deleted {
		t.Fatal("expected preview not to delete")
	}

	if len(preview.Sections) != 1 || preview.Sections[0].Section != "wireguard" || preview.Sections[0].Key != "wg-fr-peer" {
		t.Fatalf("unexpected cascade preview: %+v", preview.Sections)
	}

	status, raw = c.Do(t, http.MethodDelete, "/api/friends/peer?confirm=true", nil)
	if status != http.StatusOK {
		t.Fatalf("delete confirm status %d: %s", status, raw)
	}

	if !harness.DecodeData[types.FriendDeleteResult](t, raw).Deleted {
		t.Fatal("expected confirmed delete")
	}

	_, raw = c.Do(t, http.MethodGet, "/api/friends", nil)
	if friends := harness.DecodeData[[]types.FriendInfo](t, raw); len(friends) != 0 {
		t.Fatalf("friend not removed: %+v", friends)
	}

	_, raw = c.Do(t, http.MethodGet, "/api/config/wireguard", nil)
	if wg := harness.DecodeData[map[string]any](t, raw); len(wg) != 0 {
		t.Fatalf("tagged wireguard not removed: %+v", wg)
	}
}
