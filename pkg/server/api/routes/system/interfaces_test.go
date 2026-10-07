package system

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestReadInterfaceLink(t *testing.T) {
	for _, tc := range []readInterfaceLinkCase{
		{"eth0", `[{"ifname":"eth0","flags":["UP","LOWER_UP"],"mtu":1500,"stats64":{"rx":{"bytes":9007199254740993}}}]`},
		{"vlan10", `[{"ifname":"vlan10","link":"eth0","linkinfo":{"info_kind":"vlan","info_data":{"id":10,"protocol":"802.1Q"}}}]`},
		{"bond0", `[{"ifname":"bond0","linkinfo":{"info_kind":"bond","info_data":{"mode":"802.3ad","miimon":100}}}]`},
		{"br0", `[{"ifname":"br0","linkinfo":{"info_kind":"bridge","info_data":{"stp_state":1}}}]`},
	} {
		t.Run(tc.name, func(t *testing.T) { testReadInterfaceLinkCallback(&tc, t) })
	}
}

func TestReadInterfaceLinkFailures(t *testing.T) {
	for _, payload := range []string{`[]`, `invalid`, `[{"ifname":"other"}]`} {
		if _, err := readInterfaceLink(context.Background(), "eth0", func(context.Context, ...string) ([]byte, error) { return []byte(payload), nil }); err == nil {
			t.Fatalf("accepted %q", payload)
		}
	}

	want := errors.New("command failed")
	if _, err := readInterfaceLink(context.Background(), "eth0", func(context.Context, ...string) ([]byte, error) { return nil, want }); !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
}

func TestInterfaceStatusRejectsInvalidName(t *testing.T) {
	for _, name := range []string{"", "..", "../../etc/passwd", "eth0\n", "abcdefghijklmnop"} {
		r := httptest.NewRequest("GET", "/", nil)
		route := chi.NewRouteContext()
		route.URLParams.Add("name", name)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, route))
		w := httptest.NewRecorder()
		InterfaceStatus(w, r)
		if w.Code != 400 {
			t.Fatalf("name %q: status %d", name, w.Code)
		}
	}
}

func TestInterfaceStatusResponse(t *testing.T) {
	interfaces, err := net.Interfaces()
	if err != nil || len(interfaces) == 0 {
		t.Skip("no interfaces available")
	}

	name := interfaces[0].Name
	payload, err := json.Marshal([]map[string]any{{"ifname": name, "mtu": 1500, "linkinfo": map[string]any{"info_kind": "vlan", "info_data": map[string]any{"id": 42}}}})
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ip"), []byte("#!/bin/sh\nprintf '%s' '"+string(payload)+"'\n"), 0700); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", dir)
	router := chi.NewRouter()
	Routes(router)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/system/interfaces/"+name, nil))
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}

	var response interfaceStatusResponseCase
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if response.Result.Link["ifname"] != name || response.Result.Link["linkinfo"] == nil {
		t.Fatalf("missing link details: %s", w.Body.String())
	}
}

type readInterfaceLinkCase struct{ name, payload string }

type interfaceStatusResponseCase struct {
	Result interfaceStatus `json:"result"`
}

func testReadInterfaceLinkCallback(tc *readInterfaceLinkCase, t *testing.T) {
	link, err := readInterfaceLink(context.Background(), (*tc).name, func(unusedArg2 context.Context, args ...string) ([]byte, error) {
		return testReadInterfaceLinkCallbackCallback(tc, t, unusedArg2, args...)
	})
	if err != nil {
		t.Fatal(err)
	}

	if link["ifname"] != (*tc).name {
		t.Fatalf("wrong link: %v", link)
	}

	if (*tc).name == "eth0" && link["stats64"].(map[string]any)["rx"].(map[string]any)["bytes"].(interface{ String() string }).String() != "9007199254740993" {
		t.Fatal("counter precision lost")
	}

	if (*tc).name != "eth0" && link["linkinfo"] == nil {
		t.Fatal("type details lost")
	}
}

func testReadInterfaceLinkCallbackCallback(tc *readInterfaceLinkCase, t *testing.T, _ context.Context, args ...string) ([]byte, error) {
	if !reflect.DeepEqual(args, []string{"-j", "-d", "-s", "link", "show", "dev", (*tc).name}) {
		t.Fatalf("arguments: %v", args)
	}

	return []byte((*tc).payload), nil
}
