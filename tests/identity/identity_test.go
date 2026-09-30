package identitytest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/auth/identity"
)

func TestIdentityPersistAndVerify(t *testing.T) {
	path := filepath.Join(t.TempDir(), "id")

	id, err := identity.LoadOrCreate(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	again, err := identity.LoadOrCreate(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	if id.Fingerprint() != again.Fingerprint() {
		t.Fatal("identity not persisted across loads")
	}

	if !strings.HasPrefix(id.Fingerprint(), "SHA256:") {
		t.Fatalf("unexpected fingerprint format: %q", id.Fingerprint())
	}

	msg := []byte("challenge")
	sig := id.Sign(msg)

	pub, err := identity.ParsePublicKey(id.PublicKeyBase64())
	if err != nil {
		t.Fatalf("parse pubkey: %v", err)
	}

	if !identity.Verify(pub, msg, sig) {
		t.Fatal("signature did not verify")
	}

	if identity.Verify(pub, []byte("tampered"), sig) {
		t.Fatal("signature verified against wrong message")
	}

	if identity.Fingerprint(pub) != id.Fingerprint() {
		t.Fatal("Fingerprint(pub) mismatch")
	}

	if !identity.FingerprintMatch(id.Fingerprint(), again.Fingerprint()) {
		t.Fatal("FingerprintMatch should be true for equal fingerprints")
	}
}
