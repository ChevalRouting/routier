package persistencetest

import (
	"database/sql"
	"testing"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	_ "modernc.org/sqlite"
)

func routesDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(`CREATE TABLE kernel_routes (dst TEXT, gateway TEXT, dev TEXT, protocol TEXT, metric INTEGER, family TEXT)`); err != nil {
		t.Fatal(err)
	}

	rows := []struct {
		dst, gw, dev, proto, fam string
		metric                   int
	}{
		{"10.0.0.0/24", "", "eth0", "kernel", "ipv4", 0},
		{"0.0.0.0/0", "192.168.1.1", "eth0", "static", "ipv4", 0},
		{"192.168.1.0/24", "", "eth0", "bgp", "ipv4", 100},
		{"::/0", "fe80::1", "eth0", "bgp", "ipv6", 0},
		{"2001:db8::/32", "", "eth1", "ospf6", "ipv6", 0},
	}
	for _, r := range rows {
		if _, err := db.Exec(`INSERT INTO kernel_routes VALUES (?,?,?,?,?,?)`, r.dst, r.gw, r.dev, r.proto, r.metric, r.fam); err != nil {
			t.Fatal(err)
		}
	}

	return db
}

func TestQueryKernelRoutesPagination(t *testing.T) {
	db := routesDB(t)

	page, total, err := webdb.QueryKernelRoutes(db, webdb.Filter{}, 0, 2)
	if err != nil {
		t.Fatal(err)
	}

	if total != 5 {
		t.Errorf("total=%d want 5", total)
	}

	if len(page) != 2 {
		t.Errorf("page len=%d want 2 (limit)", len(page))
	}

	if page2, _, _ := webdb.QueryKernelRoutes(db, webdb.Filter{}, 4, 2); len(page2) != 1 {
		t.Errorf("last page len=%d want 1", len(page2))
	}
}

func TestQueryKernelRoutesFilters(t *testing.T) {
	db := routesDB(t)

	if _, total, _ := webdb.QueryKernelRoutes(db, webdb.Filter{Protocols: []string{"bgp"}}, 0, 100); total != 2 {
		t.Errorf("bgp total=%d want 2", total)
	}

	if _, total, _ := webdb.QueryKernelRoutes(db, webdb.Filter{Family: "ipv6"}, 0, 100); total != 2 {
		t.Errorf("ipv6 total=%d want 2", total)
	}

	if _, total, _ := webdb.QueryKernelRoutes(db, webdb.Filter{Query: "192.168"}, 0, 100); total != 2 {
		t.Errorf("q total=%d want 2", total)
	}
}

func TestQueryKernelRoutesDefaultOnly(t *testing.T) {
	db := routesDB(t)

	page, total, err := webdb.QueryKernelRoutes(db, webdb.Filter{DefaultOnly: true}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}

	if total != 2 || len(page) != 2 {
		t.Errorf("defaults total=%d len=%d want 2/2", total, len(page))
	}
}
