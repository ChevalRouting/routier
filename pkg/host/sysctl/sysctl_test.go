package sysctl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestList(t *testing.T) {
	root := t.TempDir()
	write(t, root, "net/ipv4/ip_forward", "1\n", 0o644)
	write(t, root, "net/ipv4/tcp_rmem", "4096\t131072\t6291456\n", 0o644)
	write(t, root, "net/ipv4/route/flush", "", 0o200)

	params, err := List(root)
	if err != nil {
		t.Fatal(err)
	}

	want := []Parameter{
		{Key: "net.ipv4.ip_forward", Value: "1"},
		{Key: "net.ipv4.tcp_rmem", Value: "4096 131072 6291456"},
	}
	if len(params) != len(want) {
		t.Fatalf("got %v, want %v", params, want)
	}

	for i := range want {
		if params[i] != want[i] {
			t.Fatalf("got %v, want %v", params[i], want[i])
		}
	}
}

func write(t *testing.T, root, rel, data string, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(data), mode); err != nil {
		t.Fatal(err)
	}
}
