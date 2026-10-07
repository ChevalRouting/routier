package systemtest

import (
	"errors"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/daemon/bind"
)

const statsDump = `+++ Statistics Dump +++ (1787178606)
++ Incoming Requests ++
                   7 QUERY
++ Incoming Queries ++
                   5 A
++ Name Server Statistics ++
                   7 IPv4 requests received
                   7 responses sent
                   4 queries resulted in successful answer
                   2 queries resulted in NXDOMAIN
                   3 queries caused recursion
++ Cache Statistics ++
[View: default]
                  12 cache hits
                   4 cache misses
--- Statistics Dump --- (1787178606)
`

const statusOut = `version: BIND 9.18.49 (Extended Support Version) <id:cd4a53b>
running on gw: Linux aarch64
boot time: Thu, 20 Aug 2026 09:26:53 GMT
last configured: Thu, 20 Aug 2026 09:27:10 GMT
configuration file: /etc/bind/named.conf
number of zones: 3 (1 automatic)
recursive clients: 0/900/1000
server is up and running
`

func TestBindStatsParse(t *testing.T) {
	stats := bind.ParseStats(statsDump)
	if len(stats) == 0 {
		t.Fatal("parsed no statistics")
	}

	want := map[string]float64{
		"QUERY":                                 7,
		"IPv4 requests received":                7,
		"queries resulted in successful answer": 4,
		"queries resulted in NXDOMAIN":          2,
		"cache hits":                            12,
		"cache misses":                          4,
	}

	for name, value := range want {
		if got := bind.StatValue(stats, name); got != value {
			t.Errorf("%s = %v, want %v", name, got, value)
		}
	}

	for _, s := range stats {
		if s.Name == "QUERY" && s.Section != "Incoming Requests" {
			t.Errorf("QUERY section = %q, want %q", s.Section, "Incoming Requests")
		}
	}
}

func TestBindKeyStatisticsFiltersAndOrders(t *testing.T) {
	key := bind.KeyStatistics(bind.ParseStats(statsDump))
	if len(key) == 0 {
		t.Fatal("no key statistics")
	}

	if key[0].Name != "QUERY" {
		t.Errorf("first key stat = %q, want QUERY", key[0].Name)
	}

	for _, s := range key {
		if s.Name == "A" {
			t.Error("non-key statistic leaked into the key set")
		}
	}
}

func TestBindStatusParse(t *testing.T) {
	restore := bind.SetRunner(func(name string, args ...string) ([]byte, error) {
		return testBindStatusParseCallback(t, name, args...)
	})
	defer restore()

	status, err := bind.New("").Status()
	if err != nil {
		t.Fatalf("status: %v", err)
	}

	if !status.Running {
		t.Error("expected running")
	}

	if !strings.HasPrefix(status.Version, "BIND 9.18") {
		t.Errorf("version = %q", status.Version)
	}

	if status.Zones != 3 {
		t.Errorf("zones = %d, want 3", status.Zones)
	}
}

func TestBindRndcArgvIncludesKeyAndControlChannel(t *testing.T) {
	var got []string

	restore := bind.SetRunner(func(_ string, args ...string) ([]byte, error) {
		got = args

		return []byte("ok"), nil
	})
	defer restore()

	if err := bind.New("").ReloadZone("home.arpa"); err != nil {
		t.Fatalf("reload zone: %v", err)
	}

	joined := strings.Join(got, " ")
	for _, want := range []string{"-s 127.0.0.1", "-p 953", "-k " + bind.RndcKey, "reload home.arpa"} {
		if !strings.Contains(joined, want) {
			t.Errorf("rndc argv missing %q, got: %s", want, joined)
		}
	}
}

func TestBindControlErrorSurfacesOutput(t *testing.T) {
	restore := bind.SetRunner(func(_ string, _ ...string) ([]byte, error) {
		return []byte("rndc: connect failed: connection refused"), errors.New("exit status 1")
	})
	defer restore()

	err := bind.New("").Reconfig()
	if err == nil {
		t.Fatal("expected an error")
	}

	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("error lost the rndc output: %v", err)
	}
}

func TestBindDigParse(t *testing.T) {
	digOut := `
; <<>> DiG 9.18.49 <<>> @127.0.0.1 nas.home.arpa A
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 4242
;; flags: qr aa rd ra; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1

;; ANSWER SECTION:
nas.home.arpa.		300	IN	A	10.0.0.10

;; Query time: 3 msec
;; SERVER: 127.0.0.1#53(127.0.0.1) (UDP)
`

	restore := bind.SetRunner(func(name string, unusedArg3 ...string) ([]byte, error) {
		return testBindDigParseCallback(t, digOut, name, unusedArg3...)
	})
	defer restore()

	result, err := bind.New("").Query("nas.home.arpa", "A")
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if result.RCode != "NOERROR" {
		t.Errorf("rcode = %q", result.RCode)
	}

	if len(result.Answers) != 1 || result.Answers[0].Data != "10.0.0.10" {
		t.Fatalf("answers = %+v", result.Answers)
	}

	if result.QueryMS != 3 {
		t.Errorf("query time = %d, want 3", result.QueryMS)
	}
}

func TestBindZoneViewMarksUnanswered(t *testing.T) {
	restore := bind.SetRunner(func(_ string, _ ...string) ([]byte, error) {
		return []byte(";; ->>HEADER<<- opcode: QUERY, status: SERVFAIL, id: 1\n"), nil
	})
	defer restore()

	views := bind.New("").Zones([]bind.ZoneInput{{Name: "home.arpa", Serial: 7}})
	if len(views) != 1 {
		t.Fatalf("views = %v", views)
	}

	if views[0].Answered {
		t.Error("a SERVFAIL must not count as answered")
	}

	if views[0].Serial != 7 {
		t.Errorf("declared serial lost: %d", views[0].Serial)
	}
}

func testBindStatusParseCallback(t *testing.T, name string, args ...string) ([]byte, error) {
	if name != "rndc" {
		t.Errorf("expected rndc, got %q", name)
	}

	return []byte(statusOut), nil
}

func testBindDigParseCallback(t *testing.T, digOut string, name string, _ ...string) ([]byte, error) {
	if name != "dig" {
		t.Errorf("expected dig, got %q", name)
	}

	return []byte(digOut), nil
}
