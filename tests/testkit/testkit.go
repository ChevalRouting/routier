package testkit

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/ChevalRouting/routier/pkg/server/api"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/golang-jwt/jwt/v5"
)

const MinimalConfig = `version: v3.0.0
hostname: rtr1
interfaces:
  lan:
    select: name=eth0
    addresses: ["192.168.1.1/24"]
`

const FullConfig = `version: v3.0.0
hostname: core
interfaces:
  wan:
    select: name=eth0
    addresses: ["198.51.100.2/24"]
  lan:
    select: name=eth1
    addresses: ["10.0.0.1/24"]
routing:
  bgp:
    asn: 65010
    router_id: 10.0.0.1
    neighbors:
      - address: 198.51.100.1
        remote_asn: 65020
  static:
    - destination: 172.16.0.0/12
      via: 10.0.0.2
sysctl:
  net.ipv4.ip_forward: "1"
ha:
  vrrp:
    - id: 10
      interface: lan
      vips: ["10.0.0.254/24"]
      priority: 150
  conntrackd:
    interface: lan
    address: 10.0.0.1
    peer_ips: ["10.0.0.3"]
ssh:
  port: 2222
  permit_root_login: "no"
wireguard:
  wg0:
    private_key: dGVzdA==
    listen_port: 51820
    addresses: ["10.10.0.1/24"]
    peers:
      - public_key: cGVlcg==
        allowed_ips: ["10.10.0.2/32"]
dns:
  nameservers: ["127.0.0.1"]
  search: ["example.net"]
  server:
    enabled: true
    listen: ["127.0.0.1", "iface(lan)", "vips(lan)"]
    allow_from: ["10.0.0.0/24"]
    allow_inbound: ["lan"]
    upstreams: ["1.1.1.1", "9.9.9.9@853"]
    log_queries: true
    threads: 2
    forward:
      - domain: corp.example.net
        servers: ["10.0.0.53"]
    cache:
      size: 64m
      rrset_size: 32m
      min_ttl: 60
      max_ttl: 3600
      prefetch: true
    zones:
      - name: example.net
        ttl: 300
        nameservers: ["ns1.example.net."]
        soa:
          primary: ns1.example.net.
          email: hostmaster@example.net
          refresh: 3600
          retry: 600
          expire: 604800
          minimum: 300
        records:
          - { name: "@",   type: A,     value: 10.0.0.1 }
          - { name: ns1,   type: A,     value: 10.0.0.1 }
          - { name: nas,   type: AAAA,  value: "2001:db8::10" }
          - { name: files, type: CNAME, value: nas.example.net. }
          - { name: "@",   type: MX,    value: mail.example.net., priority: 10 }
          - { name: "@",   type: TXT,   value: "managed by routier" }
      - name: 0.0.10.in-addr.arpa
        nameservers: ["ns1.example.net."]
        soa: { primary: ns1.example.net., email: hostmaster@example.net }
        records:
          - { name: "1", type: PTR, value: gw.example.net. }
`

type Node struct {
	URL    string
	Cfg    string
	JWT    string
	Server *httptest.Server
}

func WriteConfig(t *testing.T, dir, contents string) string {
	t.Helper()

	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return path
}

func LoadCfg(t *testing.T, yaml string) *config.Config {
	t.Helper()

	cfg, err := config.Load(WriteConfig(t, t.TempDir(), yaml))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	return cfg
}

func MintJWT(t *testing.T, secret []byte) string {
	t.Helper()

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "admin",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	s, err := tok.SignedString(secret)
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	return s
}

func NewNode(t *testing.T, configContents string) *Node {
	t.Helper()

	dir := t.TempDir()
	cfgPath := WriteConfig(t, dir, configContents)
	dbPath := filepath.Join(dir, "web.db")
	secret := []byte("test-secret-" + filepath.Base(dir))

	srv, err := api.New(t.Context(), cfgPath, dbPath, secret, false)
	if err != nil {
		t.Fatalf("api.New: %v", err)
	}

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	if u, uerr := url.Parse(ts.URL); uerr == nil {
		if port, perr := strconv.Atoi(u.Port()); perr == nil {
			srv.SetAdvertise(port, false)
		}
	}

	return &Node{URL: ts.URL, Cfg: cfgPath, JWT: MintJWT(t, secret), Server: ts}
}

func (n *Node) Do(t *testing.T, method, path string, body any) (int, []byte) {
	t.Helper()

	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}

		rdr = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, n.URL+path, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+n.JWT)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}

	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}

func DecodeData[T any](t *testing.T, raw []byte) T {
	t.Helper()

	var wrapped struct {
		Result T `json:"result"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		t.Fatalf("decode response %s: %v", string(raw), err)
	}

	return wrapped.Result
}

func BuildCLI(t *testing.T) string {
	t.Helper()

	bin := filepath.Join(t.TempDir(), "routier")
	out, err := exec.Command("go", "build", "-o", bin, "github.com/ChevalRouting/routier/cmd/routier").CombinedOutput()
	if err != nil {
		t.Fatalf("build cli: %v\n%s", err, out)
	}

	return bin
}

func RunCLI(t *testing.T, bin string, args ...string) (string, error) {
	t.Helper()

	out, err := exec.Command(bin, args...).CombinedOutput()
	return string(out), err
}
