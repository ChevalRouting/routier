package cloud

import (
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseMappingConfig(t *testing.T) {
	data := []byte(`#cloud-config
hostname: edge
routier:
  password: s3cret-pass
  config:
    hostname: edge
    interfaces:
      wan:
        select: name=eth0
        addresses: [dhcp]
`)

	p, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	if p.Password != "s3cret-pass" {
		t.Fatalf("password = %q", p.Password)
	}

	var cfg map[string]any
	if err := yaml.Unmarshal(p.Config, &cfg); err != nil {
		t.Fatal(err)
	}

	if cfg["hostname"] != "edge" || cfg["interfaces"] == nil {
		t.Fatalf("config = %#v", cfg)
	}
}

func TestParseStringConfig(t *testing.T) {
	data := []byte("routier:\n  config: |\n    hostname: core\n")
	p, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	if strings.TrimSpace(string(p.Config)) != "hostname: core" {
		t.Fatalf("config = %q", p.Config)
	}
}

func TestParseWithoutRoutierSection(t *testing.T) {
	p, err := Parse([]byte("#cloud-config\npackages: [htop]\n"))
	if err != nil {
		t.Fatal(err)
	}

	if p.Config != nil || p.Password != "" {
		t.Fatalf("provision = %#v", p)
	}
}

func TestParseRejectsListConfig(t *testing.T) {
	if _, err := Parse([]byte("routier:\n  config: [a, b]\n")); err == nil {
		t.Fatal("list config accepted")
	}
}

func TestLoadMissingFile(t *testing.T) {
	p, err := Load(filepath.Join(t.TempDir(), "cloud-config.txt"))
	if err != nil || p.Config != nil {
		t.Fatalf("provision = %#v, err = %v", p, err)
	}
}

func TestProvisionedStamp(t *testing.T) {
	stamp := filepath.Join(t.TempDir(), "state", "cloud-instance")
	if Provisioned(stamp, "i-123") {
		t.Fatal("fresh instance reported provisioned")
	}

	if err := MarkProvisioned(stamp, "i-123"); err != nil {
		t.Fatal(err)
	}

	if !Provisioned(stamp, "i-123") {
		t.Fatal("marked instance not provisioned")
	}

	if Provisioned(stamp, "i-456") {
		t.Fatal("new instance id treated as provisioned")
	}

	if Provisioned(stamp, "") {
		t.Fatal("empty instance id treated as provisioned")
	}
}

func TestParseCibiJoin(t *testing.T) {
	p, err := Parse([]byte("#cloud-config\nroutier:\n  cibi:\n    server: https://cibi.example\n    code: ABCD-EFGH\n    port: 8443\n"))
	if err != nil {
		t.Fatal(err)
	}

	if p.Cibi == nil || p.Cibi.Server != "https://cibi.example" || p.Cibi.Code != "ABCD-EFGH" || p.Cibi.Port != 8443 {
		t.Fatalf("cibi = %+v", p.Cibi)
	}
}
