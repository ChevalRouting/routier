package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func testVault(t *testing.T) *Vault {
	t.Helper()

	v, err := New(bytes.Repeat([]byte{7}, KeySize))
	if err != nil {
		t.Fatal(err)
	}

	return v
}

func TestSealOpenRoundTrip(t *testing.T) {
	v := testVault(t)
	sealed, err := v.Seal([]byte("rtr_api_secret"), []byte("routier-1"))
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Contains(sealed, []byte("rtr_api_secret")) {
		t.Fatal("sealed value contains the plaintext")
	}

	plain, err := v.Open(sealed, []byte("routier-1"))
	if err != nil {
		t.Fatal(err)
	}

	if string(plain) != "rtr_api_secret" {
		t.Fatalf("unexpected plaintext %q", plain)
	}
}

func TestOpenRejectsWrongAAD(t *testing.T) {
	v := testVault(t)
	sealed, err := v.Seal([]byte("rtr_api_secret"), []byte("routier-1"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := v.Open(sealed, []byte("routier-2")); err == nil {
		t.Fatal("opened a value sealed for another routier")
	}
}

func TestOpenRejectsWrongKey(t *testing.T) {
	sealed, err := testVault(t).Seal([]byte("rtr_api_secret"), nil)
	if err != nil {
		t.Fatal(err)
	}

	other, err := New(bytes.Repeat([]byte{9}, KeySize))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := other.Open(sealed, nil); err == nil {
		t.Fatal("opened a value with the wrong key")
	}
}

func TestOpenRejectsShortInput(t *testing.T) {
	if _, err := testVault(t).Open([]byte{1, 2, 3}, nil); err != ErrSealedTooShort {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestLoadOrCreatePersistsKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys", "vault.key")
	first, err := LoadOrCreate(path)
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Fatalf("vault key mode is %v", info.Mode().Perm())
	}

	sealed, err := first.Seal([]byte("rtr_api_secret"), nil)
	if err != nil {
		t.Fatal(err)
	}

	second, err := LoadOrCreate(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := second.Open(sealed, nil); err != nil {
		t.Fatalf("reloaded key cannot open: %v", err)
	}
}
