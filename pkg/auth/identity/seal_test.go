package identity

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func newIdentity(t *testing.T) *Identity {
	t.Helper()

	id, err := LoadOrCreate(filepath.Join(t.TempDir(), "id"))
	if err != nil {
		t.Fatalf("LoadOrCreate: %v", err)
	}

	return id
}

func TestSealOpenRoundTrip(t *testing.T) {
	alice := newIdentity(t)
	bob := newIdentity(t)

	msg := []byte("wireguard private key + password hash")

	env, err := alice.Seal(bob.X25519PublicBase64(), msg)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	if bytes.Contains(env, msg) {
		t.Fatal("plaintext leaked into the sealed envelope")
	}

	got, err := bob.Open(env, alice.PublicKeyBase64())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if !bytes.Equal(got, msg) {
		t.Fatalf("round-trip mismatch: %q", got)
	}
}

func TestOpenRejectsTamper(t *testing.T) {
	alice := newIdentity(t)
	bob := newIdentity(t)

	env, _ := alice.Seal(bob.X25519PublicBase64(), []byte("secret"))

	var e sealedEnvelope
	if err := json.Unmarshal(env, &e); err != nil {
		t.Fatal(err)
	}

	e.Cipher = "AAAA" + e.Cipher[4:]
	tampered, _ := json.Marshal(e)

	if _, err := bob.Open(tampered, alice.PublicKeyBase64()); err != ErrSealAuth {
		t.Fatalf("want ErrSealAuth on tamper, got %v", err)
	}
}

func TestOpenRejectsWrongSender(t *testing.T) {
	alice := newIdentity(t)
	bob := newIdentity(t)
	mallory := newIdentity(t)

	env, _ := alice.Seal(bob.X25519PublicBase64(), []byte("secret"))

	if _, err := bob.Open(env, mallory.PublicKeyBase64()); err != ErrSealAuth {
		t.Fatalf("want ErrSealAuth for wrong sender key, got %v", err)
	}
}

func TestOpenRejectsWrongRecipient(t *testing.T) {
	alice := newIdentity(t)
	bob := newIdentity(t)
	charlie := newIdentity(t)

	env, _ := alice.Seal(bob.X25519PublicBase64(), []byte("secret"))

	if _, err := charlie.Open(env, alice.PublicKeyBase64()); err != ErrSealDecrypt {
		t.Fatalf("want ErrSealDecrypt for wrong recipient, got %v", err)
	}
}
