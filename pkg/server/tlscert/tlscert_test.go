package tlscert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/ChevalRouting/certs"
	"github.com/ChevalRouting/routier/pkg/config"
)

func testPaths(t *testing.T) Paths {
	t.Helper()
	dir := t.TempDir()
	return Paths{Dir: dir, CertFile: filepath.Join(dir, "tls.crt"), KeyFile: filepath.Join(dir, "tls.key")}
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = listener.Close() }()
	return listener.Addr().(*net.TCPAddr).Port
}

func issue(t *testing.T, name string) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: name},
		DNSNames:     []string{name},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(60 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	private, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: private})
}

func servedName(t *testing.T, p *Provider, serverName string) string {
	t.Helper()
	cert, err := p.TLSConfig().GetCertificate(&tls.ClientHelloInfo{ServerName: serverName})
	if err != nil {
		t.Fatal(err)
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}

	return leaf.Subject.CommonName
}

func TestSettingsMapToTheCertsModule(t *testing.T) {
	paths := Paths{Dir: "/var/lib/routier"}
	files := paths.settings(config.WebTLS{CertFile: "router.crt", KeyFile: "router.key"})
	if files.CertFile != "/var/lib/routier/tls/router.crt" || files.KeyFile != "/var/lib/routier/tls/router.key" || files.ACME != nil {
		t.Fatalf("file settings = %+v", files)
	}

	acme := paths.settings(config.WebTLS{ACME: &config.WebACME{
		Directory: "https://ca.internal/acme/directory", Email: "ops@example.net", Domains: []string{"rt1.example.net"},
		HTTPPort: 8081, CAPEM: "pem", EABKID: "kid", EABKey: "c2VjcmV0",
	}})
	want := certs.ACME{
		Directory: "https://ca.internal/acme/directory", Email: "ops@example.net", Domains: []string{"rt1.example.net"},
		ChallengeAddr: ":8081", CAFile: "/var/lib/routier/acme-ca.pem", EABKID: "kid", EABKey: "c2VjcmV0",
	}
	if acme.ACME == nil || !reflect.DeepEqual(*acme.ACME, want) || acme.CertFile != "" {
		t.Fatalf("acme settings = %+v, want %+v", acme.ACME, want)
	}

	defaults := paths.settings(config.WebTLS{ACME: &config.WebACME{Domains: []string{"rt1.example.net"}}})
	if defaults.ACME.ChallengeAddr != "" || defaults.ACME.CAFile != "" || defaults.ACME.Directory != "" {
		t.Fatalf("defaults should be left to the certs module: %+v", defaults.ACME)
	}
}

func TestACMECAIsWrittenFromTheInlineBundle(t *testing.T) {
	paths := testPaths(t)
	web := config.WebTLS{ACME: &config.WebACME{Domains: []string{"rt1.example.net"}, CAPEM: "bundle\n"}}
	if err := paths.writeACMECA(web); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(paths.settings(web).ACME.CAFile)
	if err != nil || string(data) != "bundle\n" {
		t.Fatalf("ca bundle = %q, %v", data, err)
	}

	if err := paths.writeACMECA(config.WebTLS{}); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSettingsFromAppliedConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "last-applied.yml")
	port := strconv.Itoa(freePort(t))
	data := "version: " + config.CurrentVersion + "\nhostname: rt1\nweb:\n  tls:\n    acme:\n      directory: https://127.0.0.1:1/directory\n      domains: [rt1.example.net]\n      http_port: " + port + "\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	settings, err := LoadSettings(path)
	if err != nil {
		t.Fatal(err)
	}

	if settings.ACME == nil || settings.ACME.Domains[0] != "rt1.example.net" || strconv.Itoa(settings.ACME.HTTPPort) != port {
		t.Fatalf("settings = %+v", settings)
	}

	p, err := New(testPaths(t), config.WebTLS{})
	if err != nil {
		t.Fatal(err)
	}

	p.reload(path)
	if p.Status().Mode != "acme" {
		t.Fatalf("mode after reload = %q", p.Status().Mode)
	}
}

func TestStoreFileAcceptsOnlyPEMFiles(t *testing.T) {
	paths := testPaths(t)
	p, err := New(paths, config.WebTLS{})
	if err != nil {
		t.Fatal(err)
	}

	cert, key := issue(t, "custom.example")
	if err := p.StoreFile("custom.crt", cert); err != nil {
		t.Fatal(err)
	}

	if err := p.StoreFile("custom.key", key); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(paths.File("custom.key"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("key mode = %v, %v", info.Mode().Perm(), err)
	}

	for name, data := range map[string][]byte{"../escape.crt": cert, "notes.txt": []byte("hello")} {
		if err := p.StoreFile(name, data); !errors.Is(err, ErrInvalidFile) {
			t.Fatalf("StoreFile(%q) = %v, want ErrInvalidFile", name, err)
		}
	}

	if files := p.Status().Files; len(files) != 2 || files[0] != "custom.crt" || files[1] != "custom.key" {
		t.Fatalf("files = %q", files)
	}

	if content, err := p.ReadCertificate("custom.crt"); err != nil || content != string(cert) {
		t.Fatalf("ReadCertificate(custom.crt) = %q, %v", content, err)
	}

	if _, err := p.ReadCertificate("custom.key"); !errors.Is(err, ErrInvalidFile) {
		t.Fatalf("ReadCertificate(custom.key) = %v, want ErrInvalidFile", err)
	}

	served, err := New(paths, config.WebTLS{CertFile: "custom.crt", KeyFile: "custom.key"})
	if err != nil {
		t.Fatal(err)
	}

	if got := servedName(t, served, ""); got != "custom.example" || served.Status().Mode != "file" {
		t.Fatalf("served %q in mode %q", got, served.Status().Mode)
	}
}
