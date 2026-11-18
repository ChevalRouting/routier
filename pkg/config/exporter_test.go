package config

import (
	"slices"
	"testing"
)

func TestExporterAddresses(t *testing.T) {
	if got := (&ExporterConfig{}).Addresses(); !slices.Equal(got, []string{"127.0.0.1:9469", "[::1]:9469"}) {
		t.Fatalf("default addresses = %v", got)
	}

	if got := (&ExporterConfig{Listen: []string{"0.0.0.0"}, Port: 9100}).Addresses(); !slices.Equal(got, []string{"0.0.0.0:9100"}) {
		t.Fatalf("addresses = %v", got)
	}
}
