//go:build linux

package motd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/net/iproute"
	"github.com/ChevalRouting/routier/pkg/net/netlink"
	"github.com/ChevalRouting/routier/pkg/telemetry/stats"
)

const dbPath = "/var/lib/routier/web.db"

const banner = ` .-----.__________________     ___             __  _
 | .-. |__|__|__|__|__|__|    / _ \___  __ __ / /_(_)__ ____
 | '-' |                 |   / , _/ _ \/ // // __/ / -_) __/
 '-(o)-'-----------(o)---'  /_/|_|\___/\_,_/ \__/_/\__/_/
`

type RouteSummary struct {
	BGP    int
	OSPF   int
	Static int
	Total  int
}

type Bandwidth struct {
	RxBytesPS float64
	TxBytesPS float64
}

func Write(ctx context.Context, user, pass string) {
	content := Render(routierVersion(), ifaceAddrs(), routeSummary(), bandwidth5m(ctx), user, pass)
	_ = os.WriteFile("/etc/motd", []byte(content), 0644)
}

func WriteIssue(version string) {
	_ = os.WriteFile("/etc/issue", []byte(RenderIssue(version)), 0644)
}

func RenderIssue(version string) string {
	if version == "" || version == "dev" {
		version = routierVersion()
	}

	return fmt.Sprintf("\n%s  %s\n\n", banner, version)
}

func Render(version string, ifaces []string, routes RouteSummary, bw *Bandwidth, user, pass string) string {
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "\n%s  %s\n\n", banner, version)

	if len(ifaces) > 0 {
		_, _ = fmt.Fprintf(&b, "  Network:\n")
		for _, line := range ifaces {
			_, _ = fmt.Fprintf(&b, "    %s\n", line)
		}

		_, _ = fmt.Fprintln(&b)
	}

	if routes.Total > 0 {
		_, _ = fmt.Fprintf(&b, "  Routes:  %d total   bgp %d   ospf %d   static %d\n\n",
			routes.Total, routes.BGP, routes.OSPF, routes.Static)
	}

	if bw != nil {
		_, _ = fmt.Fprintf(&b, "  Traffic (5m avg):  rx %s   tx %s\n\n", humanBits(bw.RxBytesPS), humanBits(bw.TxBytesPS))
	}

	if user != "" && pass != "" {
		_, _ = fmt.Fprintf(&b, "  Initial credentials (change via web UI at https://<ip>:8443):\n")
		_, _ = fmt.Fprintf(&b, "    username: %s\n", user)
		_, _ = fmt.Fprintf(&b, "    password: %s\n", pass)
		_, _ = fmt.Fprintln(&b)
	}

	return b.String()
}

func routeSummary() RouteSummary {
	var s RouteSummary
	for _, r := range iproute.ShowRoutes() {
		s.Total++

		switch r.Protocol {
		case "bgp":
			s.BGP++
		case "ospf", "ospf6":
			s.OSPF++
		case "static":
			s.Static++
		}
	}

	return s
}

func bandwidth5m(ctx context.Context) *Bandwidth {
	db, err := webdb.InitDB(ctx, dbPath)
	if err != nil {
		return nil
	}

	defer func(action func() error) { _ = action() }(db.Close)

	window := int64(5 * 60)
	cutoff := time.Now().Unix() - window
	points := webdb.IfaceTotals(ctx, db, cutoff, webdb.HistoryBucket(window, 60), stats.PhysicalIfaces())
	if len(points) == 0 {
		return nil
	}

	var rx, tx float64
	var n float64
	for _, p := range points {
		if p.RxBytesPS != nil {
			rx += *p.RxBytesPS
		}

		if p.TxBytesPS != nil {
			tx += *p.TxBytesPS
		}

		n++
	}

	if n == 0 {
		return nil
	}

	return &Bandwidth{RxBytesPS: rx / n, TxBytesPS: tx / n}
}

func humanBits(bytesPerSec float64) string {
	bits := bytesPerSec * 8
	units := []string{"bps", "Kbps", "Mbps", "Gbps", "Tbps"}
	i := 0
	for bits >= 1000 && i < len(units)-1 {
		bits /= 1000
		i++
	}

	return fmt.Sprintf("%.1f %s", bits, units[i])
}

func routierVersion() string {
	out, err := exec.Command("apk", "info", "routier").Output()
	if err != nil {
		return "dev"
	}

	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return "dev"
	}

	v := strings.TrimPrefix(fields[0], "routier-")
	if i := strings.LastIndex(v, "-r"); i >= 0 {
		v = v[:i]
	}

	if v == "" {
		return "dev"
	}

	return v
}

func ifaceAddrs() []string {
	nics, err := netlink.SystemNics()
	if err != nil {
		return nil
	}

	var lines []string
	for _, n := range nics {
		if len(n.Addrs) > 0 {
			lines = append(lines, fmt.Sprintf("%-12s  %s", n.Name, strings.Join(n.Addrs, "  ")))
		} else {
			lines = append(lines, fmt.Sprintf("%-12s  (no address)", n.Name))
		}
	}

	return lines
}
