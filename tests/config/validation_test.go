package configtest

import (
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestValidationSetAndNewErrors(t *testing.T) {
	cfg := &config.Config{Hostname: "r1"}
	before := config.ValidationSet(cfg)
	if len(before) != 0 {
		t.Fatalf("base config should be valid, got %v", before)
	}

	cfg.Interfaces = map[string]*config.Interface{"eth0": {}}
	if added := config.NewValidationErrors(before, cfg); len(added) == 0 {
		t.Fatal("expected a new validation error for interface missing select")
	}

	cfg.Interfaces = map[string]*config.Interface{"eth0": {Select: "name=eth0"}}
	if added := config.NewValidationErrors(before, cfg); len(added) != 0 {
		t.Fatalf("valid interface should not error: %v", added)
	}
}

func TestNewValidationErrorsIgnoresPreExisting(t *testing.T) {
	cfg := &config.Config{
		Hostname:   "r1",
		Interfaces: map[string]*config.Interface{"bad": {}},
	}
	before := config.ValidationSet(cfg)
	if len(before) == 0 {
		t.Fatal("expected a pre-existing validation error")
	}

	cfg.Hostname = "r2"
	if added := config.NewValidationErrors(before, cfg); len(added) != 0 {
		t.Fatalf("unrelated change blocked by pre-existing error: %v", added)
	}
}
