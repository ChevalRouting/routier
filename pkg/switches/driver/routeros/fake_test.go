package routeros

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ChevalRouting/routier/pkg/switches/driver"
)

const (
	fakeUser     = "cibi"
	fakePassword = "secret"
)

var fixtures = map[string]string{
	"/rest/system/identity":    `{"name":"crs310-lab"}`,
	"/rest/system/resource":    `{"board-name":"CRS310-8G+2S+","version":"7.16 (stable)","architecture-name":"arm64","uptime":"1d2h"}`,
	"/rest/system/routerboard": `{"routerboard":"true","model":"CRS310-8G+2S+","serial-number":"HJ0123456","current-firmware":"7.16"}`,
	"/rest/interface": `[
{".id":"*1","name":"ether1","type":"ether","running":"true","disabled":"false","mac-address":"48:A9:8A:00:00:01","comment":"desk"},
{".id":"*2","name":"ether2","type":"ether","running":"true","disabled":"false","mac-address":"48:A9:8A:00:00:02"},
{".id":"*3","name":"ether3","type":"ether","running":"false","disabled":"false","mac-address":"48:A9:8A:00:00:03"},
{".id":"*7","name":"ether7","type":"ether","running":"true","disabled":"false","mac-address":"48:A9:8A:00:00:07","slave":"true"},
{".id":"*8","name":"ether8","type":"ether","running":"true","disabled":"false","mac-address":"48:A9:8A:00:00:08","slave":"true"},
{".id":"*9","name":"sfp-sfpplus1","type":"ether","running":"true","disabled":"false","mac-address":"48:A9:8A:00:00:09"},
{".id":"*A","name":"sfp-sfpplus2","type":"ether","running":"false","disabled":"true","mac-address":"48:A9:8A:00:00:0A"},
{".id":"*B","name":"bond1","type":"bond","running":"true","disabled":"false","mac-address":"48:A9:8A:00:00:07"},
{".id":"*C","name":"bridge","type":"bridge","running":"true","disabled":"false","mac-address":"48:A9:8A:00:00:01"},
{".id":"*D","name":"mgmt","type":"vlan","running":"true","disabled":"false","mac-address":"48:A9:8A:00:00:01"}
]`,
	"/rest/interface/bridge": `[{".id":"*C","name":"bridge","vlan-filtering":"true","pvid":"1","protocol-mode":"rstp"}]`,
	"/rest/interface/bridge/port": `[
{".id":"*0","interface":"ether1","bridge":"bridge","pvid":"10","frame-types":"admit-only-untagged-and-priority-tagged","dynamic":"false","disabled":"false"},
{".id":"*1","interface":"ether2","bridge":"bridge","pvid":"1","frame-types":"admit-all","dynamic":"false","disabled":"false"},
{".id":"*2","interface":"ether3","bridge":"bridge","pvid":"20","frame-types":"admit-only-untagged-and-priority-tagged","dynamic":"false","disabled":"false"},
{".id":"*3","interface":"bond1","bridge":"bridge","pvid":"1","frame-types":"admit-only-vlan-tagged","dynamic":"false","disabled":"false"},
{".id":"*4","interface":"sfp-sfpplus1","bridge":"bridge","pvid":"1","frame-types":"admit-only-vlan-tagged","dynamic":"false","disabled":"false"}
]`,
	"/rest/interface/bridge/vlan": `[
{".id":"*1","bridge":"bridge","vlan-ids":"10","tagged":"bridge,ether2,sfp-sfpplus1","untagged":"","current-tagged":"bridge,ether2,sfp-sfpplus1","current-untagged":"ether1","dynamic":"false","disabled":"false"},
{".id":"*2","bridge":"bridge","vlan-ids":"20","tagged":"ether2,sfp-sfpplus1,bond1","untagged":"ether3","current-tagged":"ether2,sfp-sfpplus1,bond1","current-untagged":"ether3","dynamic":"false","disabled":"false"},
{".id":"*3","bridge":"bridge","vlan-ids":"100-102","tagged":"sfp-sfpplus1","untagged":"","current-tagged":"sfp-sfpplus1","current-untagged":"","dynamic":"false","disabled":"false"},
{".id":"*4","bridge":"bridge","vlan-ids":"1","tagged":"","untagged":"","current-tagged":"","current-untagged":"bridge,ether2","dynamic":"true","disabled":"false"}
]`,
	"/rest/interface/bonding": `[{".id":"*B","name":"bond1","slaves":"ether7,ether8","mode":"802.3ad","transmit-hash-policy":"layer-2-and-3"}]`,
}

type recordedCall struct {
	Method string
	Path   string
	Body   map[string]any
}

type fakeRouterOS struct {
	mu    sync.Mutex
	calls []recordedCall
}

func (f *fakeRouterOS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, password, ok := r.BasicAuth()
	if !ok || user != fakeUser || password != fakePassword {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":401,"message":"Unauthorized"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost && r.URL.Path == "/rest/interface/ethernet/monitor" {
		_, _ = w.Write([]byte(`[{"name":"ether1","status":"link-ok","rate":"1Gbps"},{"name":"sfp-sfpplus1","status":"link-ok","rate":"10Gbps"}]`))
		return
	}

	if r.Method != http.MethodGet {
		f.record(r)
		_, _ = w.Write([]byte(`{".id":"*99"}`))
		return
	}

	body, ok := fixtures[r.URL.Path]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":404,"message":"Not Found"}`))
		return
	}

	_, _ = w.Write([]byte(body))
}

func (f *fakeRouterOS) record(r *http.Request) {
	data, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(data, &body)

	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, recordedCall{Method: r.Method, Path: strings.TrimPrefix(r.URL.Path, "/rest"), Body: body})
}

func newFakeDriver(t *testing.T, password string) (*Driver, *fakeRouterOS) {
	t.Helper()

	fake := &fakeRouterOS{}
	server := httptest.NewTLSServer(fake)
	t.Cleanup(server.Close)

	target := driver.Target{
		Host:          strings.TrimPrefix(server.URL, "https://"),
		TLSSkipVerify: true,
		Credentials:   map[string]string{"username": fakeUser, "password": password},
	}
	created, err := New(target)
	if err != nil {
		t.Fatal(err)
	}

	return created.(*Driver), fake
}

func (f *fakeRouterOS) recorded() []recordedCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]recordedCall(nil), f.calls...)
}
