package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/auth/identity"
	"github.com/ChevalRouting/routier/pkg/types"
)

func TestConfigDecryptsSealedResponse(t *testing.T) {
	sender, err := identity.LoadOrCreate(filepath.Join(t.TempDir(), "sender"))
	if err != nil {
		t.Fatal(err)
	}

	recipient, err := identity.LoadOrCreate(filepath.Join(t.TempDir(), "recipient"))
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Hostname: "peer", Version: "v3.0.0"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var payload any = cfg
		env, _ := json.Marshal(types.Response[any]{Result: &payload})

		sealed, serr := sender.Seal(recipient.X25519PublicBase64(), env)
		if serr != nil {
			http.Error(w, serr.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", types.SealedContentType)
		w.Write(sealed)
	}))
	defer srv.Close()

	c := New(&Config{URL: srv.URL})
	c.SetSealOpener(recipient, sender.PublicKeyBase64())

	got, err := c.Config(context.Background())
	if err != nil {
		t.Fatalf("Config: %v", err)
	}

	if got.Hostname != "peer" {
		t.Fatalf("decrypted config mismatch: %q", got.Hostname)
	}
}

func TestConfigDecryptsLargeSealedResponse(t *testing.T) {
	sender, _ := identity.LoadOrCreate(filepath.Join(t.TempDir(), "sender"))
	recipient, _ := identity.LoadOrCreate(filepath.Join(t.TempDir(), "recipient"))

	big := strings.Repeat("x", 2<<20)
	cfg := &config.Config{Hostname: big, Version: "v3.0.0"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var payload any = cfg
		env, _ := json.Marshal(types.Response[any]{Result: &payload})
		sealed, _ := sender.Seal(recipient.X25519PublicBase64(), env)
		w.Header().Set("Content-Type", types.SealedContentType)
		w.Write(sealed)
	}))
	defer srv.Close()

	c := New(&Config{URL: srv.URL})
	c.SetSealOpener(recipient, sender.PublicKeyBase64())

	got, err := c.Config(context.Background())
	if err != nil {
		t.Fatalf("Config: %v", err)
	}

	if got.Hostname != big {
		t.Fatalf("decrypted large config mismatch (len %d)", len(got.Hostname))
	}
}

func TestConfigRejectsSealedResponseWithoutOpener(t *testing.T) {
	sender, _ := identity.LoadOrCreate(filepath.Join(t.TempDir(), "sender"))
	recipient, _ := identity.LoadOrCreate(filepath.Join(t.TempDir(), "recipient"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var payload any = &config.Config{Hostname: "peer"}
		env, _ := json.Marshal(types.Response[any]{Result: &payload})
		sealed, _ := sender.Seal(recipient.X25519PublicBase64(), env)
		w.Header().Set("Content-Type", types.SealedContentType)
		w.Write(sealed)
	}))
	defer srv.Close()

	c := New(&Config{URL: srv.URL})
	if _, err := c.Config(context.Background()); err == nil {
		t.Fatal("expected error when a sealed response arrives without a decryption key")
	}
}
