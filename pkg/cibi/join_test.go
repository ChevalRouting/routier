package cibi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/auth/identity"
	webdb "github.com/ChevalRouting/routier/pkg/db"
)

func testRoutier(t *testing.T) (*webdb.DB, *identity.Identity) {
	t.Helper()
	dir := t.TempDir()
	db, err := webdb.InitDB(context.Background(), filepath.Join(dir, "web.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = db.Close() })
	self, err := identity.LoadOrCreate(filepath.Join(dir, identity.KeyFilename))
	if err != nil {
		t.Fatal(err)
	}

	return db, self
}

type cibiServer struct {
	status int
	body   string
	seen   *joinRequest
}

func (c cibiServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/enroll" || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	_ = json.NewDecoder(r.Body).Decode(c.seen)
	w.WriteHeader(c.status)
	_, _ = w.Write([]byte(c.body))
}

func fakeCibi(t *testing.T, status int, body string, seen *joinRequest) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(cibiServer{status: status, body: body, seen: seen})
	t.Cleanup(server.Close)
	return server
}

func TestJoinSendsAKeyCibiCanUse(t *testing.T) {
	db, self := testRoutier(t)
	var seen joinRequest
	server := fakeCibi(t, http.StatusOK, `{"result":{"routier_id":"r1","name":"edge","networks":["Office"],"linked":"hub","applying":true}}`, &seen)
	result, err := Join{Server: server.URL + "/", Code: "ABCD-EFGH", Port: 8443}.Run(context.Background(), db, self)
	if err != nil {
		t.Fatal(err)
	}

	if result.Name != "edge" || result.Summary() != "joined cibi as edge; networks Office; linked to hub; cibi is applying the configuration" {
		t.Fatalf("result %+v", result)
	}

	if seen.Code != "ABCD-EFGH" || seen.Port != 8443 || seen.Fingerprint != self.Fingerprint() {
		t.Fatalf("request %+v", seen)
	}

	key, err := webdb.AuthenticateAPIKey(context.Background(), db, seen.Token)
	if err != nil || !strings.HasPrefix(key.Name, KeyName+" ") {
		t.Fatalf("the token sent to cibi does not authenticate: %+v %v", key, err)
	}

	first := seen.Token
	if _, err := (Join{Server: server.URL, Code: "IJKL"}).Run(context.Background(), db, self); err != nil {
		t.Fatal(err)
	}

	if _, err := webdb.AuthenticateAPIKey(context.Background(), db, first); err == nil {
		t.Fatal("joining again must revoke the key of the previous join")
	}
}

func TestRefusedJoinRevokesTheKey(t *testing.T) {
	db, self := testRoutier(t)
	var seen joinRequest
	server := fakeCibi(t, http.StatusForbidden, `{"code":403,"error":"this code is unknown, used or expired"}`, &seen)
	if _, err := (Join{Server: server.URL, Code: "used"}).Run(context.Background(), db, self); !errors.Is(err, ErrRefused) {
		t.Fatalf("refused code: %v", err)
	}

	if _, err := webdb.AuthenticateAPIKey(context.Background(), db, seen.Token); err == nil {
		t.Fatal("the key of a refused join still authenticates")
	}

	if err := (Join{Server: "cibi.example", Code: "x"}).Validate(); err == nil {
		t.Fatal("an address without a scheme was accepted")
	}
}

func TestPendingRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), PendingFilename)
	if _, found, err := LoadPending(path); found || err != nil {
		t.Fatalf("missing file: %v %v", found, err)
	}

	want := Join{Server: "https://cibi.example", Code: "ABCD", URL: "https://203.0.113.9:8080"}
	if err := SavePending(path, want); err != nil {
		t.Fatal(err)
	}

	got, found, err := LoadPending(path)
	if err != nil || !found || got != want {
		t.Fatalf("pending %+v %v %v", got, found, err)
	}
}
