package live

import (
	"crypto/tls"
	"path/filepath"
	"strings"
	"testing"
)

func TestPassword(t *testing.T) {
	p, err := Password(16)
	if err != nil {
		t.Fatal(err)
	}

	if len(p) != 16 || strings.Trim(p, passwordAlphabet) != "" {
		t.Fatalf("password = %q", p)
	}
}

func TestBanner(t *testing.T) {
	text := Banner("HEADER\n", []string{"https://192.0.2.10:8080"}, "s3cretPass")
	for _, want := range []string{"HEADER", "https://192.0.2.10:8080", "password s3cretPass", "setup-routier"} {
		if !strings.Contains(text, want) {
			t.Errorf("banner misses %q:\n%s", want, text)
		}
	}

	if strings.Contains(Banner("", nil, ""), "password") {
		t.Error("banner without a seed password mentions one")
	}
}

func TestEnsureCert(t *testing.T) {
	dir := t.TempDir()
	cert, key := filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key")
	created, err := EnsureCert(cert, key, "routier")
	if err != nil || !created {
		t.Fatalf("created = %v, err = %v", created, err)
	}

	if created, err := EnsureCert(cert, key, "routier"); err != nil || created {
		t.Fatalf("second call created = %v, err = %v", created, err)
	}

	if _, err := tls.LoadX509KeyPair(cert, key); err != nil {
		t.Fatalf("generated pair does not load: %v", err)
	}
}
