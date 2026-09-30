package friendstest

import (
	"github.com/ChevalRouting/routier/tests/testkit"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func friendStub(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/friends/hello", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"result":{"hostname":"peer","identity_fingerprint":"SHA256:stubfingerprint","identity_public_key":"c3R1Yg==","version":"test"}}`))
	})
	mux.HandleFunc("/api/friends/interfaces", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"result":[{"name":"wan","addresses":["198.51.100.9"]}]}`))
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func TestCLIFriendsParity(t *testing.T) {
	bin := testkit.BuildCLI(t)
	stub := friendStub(t)

	dir := t.TempDir()
	cfg := testkit.WriteConfig(t, dir, "version: v3.0.0\nhostname: cli-node\n")
	id := filepath.Join(dir, "identity")

	common := []string{"friends", "--config", cfg, "--identity", id}

	if out, err := testkit.RunCLI(t, bin, append(append([]string{}, common...),
		"add", "--name", "peer", "--url", stub.URL, "--token", "tok", "--tls-skip")...); err != nil {
		t.Fatalf("friends add: %v\n%s", err, out)
	}

	out, err := testkit.RunCLI(t, bin, append(append([]string{}, common...), "list")...)
	if err != nil || !strings.Contains(out, "peer") {
		t.Fatalf("friends list = %q, err %v", out, err)
	}

	if out, err := testkit.RunCLI(t, bin, append(append([]string{}, common...),
		"preview", "--url", stub.URL, "--token", "tok", "--tls-skip")...); err != nil || !strings.Contains(out, "SHA256:stubfingerprint") {
		t.Fatalf("friends preview = %q, err %v", out, err)
	}

	if out, err := testkit.RunCLI(t, bin, append(append([]string{}, common...), "interfaces", "peer")...); err != nil || !strings.Contains(out, "wan") {
		t.Fatalf("friends interfaces = %q, err %v", out, err)
	}

	if out, err := testkit.RunCLI(t, bin, append(append([]string{}, common...), "update", "peer", "--enabled=false")...); err != nil {
		t.Fatalf("friends update: %v\n%s", err, out)
	}

	if out, err := testkit.RunCLI(t, bin, append(append([]string{}, common...), "remove", "peer", "--yes")...); err != nil {
		t.Fatalf("friends remove: %v\n%s", err, out)
	}

	out, err = testkit.RunCLI(t, bin, append(append([]string{}, common...), "list")...)
	if err != nil || strings.Contains(out, "peer") {
		t.Fatalf("friend not removed: %q, err %v", out, err)
	}
}
