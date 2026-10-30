package auth

import (
	"path/filepath"
	"testing"
)

func TestLoadAdminConfigWithoutConfig(t *testing.T) {
	cfg, err := loadAdminConfig(filepath.Join(t.TempDir(), "config.yml"))
	if err != nil {
		t.Fatal(err)
	}

	if !isUIAdmin(cfg, builtinAdmin) {
		t.Fatal("built-in admin rejected without a config")
	}

	if isUIAdmin(cfg, "guest") {
		t.Fatal("unknown user accepted without a config")
	}
}
