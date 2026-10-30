package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
)

func writeFriendConfig(t *testing.T, path, token string) {
	t.Helper()

	data := "version: " + config.CurrentVersion + "\nfriends:\n  - name: edge\n    url: https://edge.example\n    token: " + token + "\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestFriendTokenCacheReloadsAfterTTL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	writeFriendConfig(t, path, "first")

	cache := &friendTokenCache{}
	now := time.Now()
	if f := cache.lookup(path, "first", now); f == nil || f.Name != "edge" {
		t.Fatalf("lookup(first) = %v, want friend edge", f)
	}

	writeFriendConfig(t, path, "second")
	if f := cache.lookup(path, "second", now.Add(time.Second)); f != nil {
		t.Fatal("a cached snapshot should not see the new token yet")
	}

	if f := cache.lookup(path, "second", now.Add(friendTokenTTL)); f == nil {
		t.Fatal("the cache should reload after the ttl")
	}

	if f := cache.lookup(path, "first", now.Add(friendTokenTTL)); f != nil {
		t.Fatal("a removed token must stop authenticating after a reload")
	}
}
