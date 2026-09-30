package identity

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestSharedKeySymmetry(t *testing.T) {
	alice := newIdentity(t)
	bob := newIdentity(t)

	ka, err := alice.SharedKey(bob.X25519PublicBase64())
	if err != nil {
		t.Fatalf("alice SharedKey: %v", err)
	}

	kb, err := bob.SharedKey(alice.X25519PublicBase64())
	if err != nil {
		t.Fatalf("bob SharedKey: %v", err)
	}

	if ka != kb {
		t.Fatal("shared keys differ between the two peers")
	}

	mallory := newIdentity(t)
	km, _ := alice.SharedKey(mallory.X25519PublicBase64())
	if km == ka {
		t.Fatal("unrelated peer derived the same shared key")
	}
}

func TestSealSharedRoundTrip(t *testing.T) {
	alice := newIdentity(t)
	bob := newIdentity(t)

	key, _ := alice.SharedKey(bob.X25519PublicBase64())
	msg := []byte("interfaces + exports + config")

	env, err := SealShared(key, msg)
	if err != nil {
		t.Fatalf("SealShared: %v", err)
	}

	if bytes.Contains(env, msg) {
		t.Fatal("plaintext leaked into the shared-sealed envelope")
	}

	got, err := OpenShared(key, env)
	if err != nil {
		t.Fatalf("OpenShared: %v", err)
	}

	if !bytes.Equal(got, msg) {
		t.Fatalf("round-trip mismatch: %q", got)
	}
}

func TestOpenSharedWrongKey(t *testing.T) {
	alice := newIdentity(t)
	bob := newIdentity(t)
	charlie := newIdentity(t)

	key, _ := alice.SharedKey(bob.X25519PublicBase64())
	other, _ := alice.SharedKey(charlie.X25519PublicBase64())

	env, _ := SealShared(key, []byte("secret"))
	if _, err := OpenShared(other, env); err != ErrSealDecrypt {
		t.Fatalf("want ErrSealDecrypt for wrong key, got %v", err)
	}
}

func TestOpenSharedRejectsStale(t *testing.T) {
	alice := newIdentity(t)
	bob := newIdentity(t)

	key, _ := alice.SharedKey(bob.X25519PublicBase64())
	env, _ := SealShared(key, []byte("secret"))

	var e sharedEnvelope
	if err := json.Unmarshal(env, &e); err != nil {
		t.Fatal(err)
	}

	e.TS = time.Now().Add(-10 * time.Minute).Unix()
	stale, _ := json.Marshal(e)

	if _, err := OpenShared(key, stale); err != ErrSealStale {
		t.Fatalf("want ErrSealStale, got %v", err)
	}
}
