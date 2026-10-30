package setup

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestNameservers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resolv.conf")
	body := "search lan\nnameserver 192.0.2.53\n# nameserver 198.51.100.1\nnameserver 2001:db8::53\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	got := nameservers(path)
	want := []string{"192.0.2.53", "2001:db8::53"}
	if !slices.Equal(got, want) {
		t.Fatalf("nameservers = %v, want %v", got, want)
	}
}

func TestNameserversMissingFile(t *testing.T) {
	if got := nameservers(filepath.Join(t.TempDir(), "absent")); len(got) != 0 {
		t.Fatalf("nameservers = %v, want none", got)
	}
}
