package switches

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	swengine "github.com/ChevalRouting/routier/pkg/switches"
	"github.com/ChevalRouting/routier/pkg/switches/model"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
)

func TestCandidatesGroupMatchAndSkip(t *testing.T) {
	catalog, err := model.Load("")
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Switches: map[string]*config.Switch{"crs309": {Host: "10.0.0.3"}}}
	neighbors := []types.LLDPNeighbor{
		{LocalIface: "eth1", Protocol: "CDP", ChassisID: "48:a9:8a:00:00:01", ChassisName: "crs310"},
		{LocalIface: "eth1", Protocol: "LLDP", ChassisID: "48:a9:8a:00:00:01", ChassisName: "crs310", SysDescr: "MikroTik RouterOS 7.16 (stable) CRS310-8G+2S+", MgmtIP: "10.0.0.2", PortID: "sfp-sfpplus1", Capabilities: "Bridge"},
		{LocalIface: "eth2", Protocol: "LLDP", ChassisID: "48:a9:8a:00:00:02", ChassisName: "crs309", MgmtIP: "10.0.0.3"},
		{LocalIface: "eth3", Protocol: "LLDP", ChassisID: "aa:bb", ChassisName: "edge-2", SysDescr: "Routier"},
	}

	got := candidates(neighbors, cfg, catalog)
	if len(got) != 1 {
		t.Fatalf("expected one candidate, got %+v", got)
	}

	c := got[0]
	if c.Name != "crs310" || c.MgmtIP != "10.0.0.2" || c.ModelID != "mikrotik-crs310-8g-2s" || c.LocalIface != "eth1" || len(c.Protocols) != 2 {
		t.Fatalf("unexpected candidate %+v", c)
	}
}

func testApp(t *testing.T) *appctx.App {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "web.db")
	db, err := webdb.InitDB(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = db.Close() })

	creds, err := swengine.OpenCredentials(db, dbPath)
	if err != nil {
		t.Fatal(err)
	}

	catalog, err := model.Load("")
	if err != nil {
		t.Fatal(err)
	}

	return &appctx.App{DB: db, Validator: appctx.NewValidator(), Switches: swengine.NewEngine(creds, catalog, swengine.DefaultDrivers)}
}

func serve(t *testing.T, app *appctx.App, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	r := chi.NewRouter()
	r.Use(app.Inject())
	Routes(r)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestCredentialsEndpoints(t *testing.T) {
	app := testApp(t)
	if rec := serve(t, app, http.MethodPut, "/api/switches/crs310/credentials", `{"values":{"username":"cibi"}}`); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("missing password accepted: %d %s", rec.Code, rec.Body)
	}

	if rec := serve(t, app, http.MethodPut, "/api/switches/crs310/credentials", `{"values":{"username":"cibi","password":"s3cret"}}`); rec.Code != http.StatusOK {
		t.Fatalf("credentials refused: %d %s", rec.Code, rec.Body)
	}

	stored, err := app.Switches.Credentials().Get(context.Background(), "crs310")
	if err != nil || stored["password"] != "s3cret" {
		t.Fatalf("credentials not stored: %v %v", stored, err)
	}

	if rec := serve(t, app, http.MethodDelete, "/api/switches/crs310/credentials", ""); rec.Code != http.StatusOK {
		t.Fatalf("delete failed: %d %s", rec.Code, rec.Body)
	}

	if rec := serve(t, app, http.MethodDelete, "/api/switches/crs310/credentials", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("second delete: %d %s", rec.Code, rec.Body)
	}
}

func TestModelsEndpoint(t *testing.T) {
	rec := serve(t, testApp(t), http.MethodGet, "/api/switch-models", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "mikrotik-crs309-1g-8s") {
		t.Fatalf("unexpected models response: %d %s", rec.Code, rec.Body)
	}
}
