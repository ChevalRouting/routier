//go:build linux

package motd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/iproute"
	vnl "github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
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
	RxBps float64
	TxBps float64
}

func Write(user, pass string) {
	content := Render(routierVersion(), ifaceAddrs(), routeSummary(), bandwidth5m(), user, pass)
	_ = os.WriteFile("/etc/motd", []byte(content), 0644)
}

func Render(version string, ifaces []string, routes RouteSummary, bw *Bandwidth, user, pass string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n%s  %s\n\n", banner, version)

	if len(ifaces) > 0 {
		fmt.Fprintf(&b, "  Network:\n")
		for _, line := range ifaces {
			fmt.Fprintf(&b, "    %s\n", line)
		}

		fmt.Fprintln(&b)
	}

	if routes.Total > 0 {
		fmt.Fprintf(&b, "  Routes:  %d total   bgp %d   ospf %d   static %d\n\n",
			routes.Total, routes.BGP, routes.OSPF, routes.Static)
	}

	if bw != nil {
		fmt.Fprintf(&b, "  Traffic (5m avg):  rx %s   tx %s\n\n", humanBits(bw.RxBps), humanBits(bw.TxBps))
	}

	if user != "" && pass != "" {
		fmt.Fprintf(&b, "  Initial credentials (change via web UI at https://<ip>:8443):\n")
		fmt.Fprintf(&b, "    username: %s\n", user)
		fmt.Fprintf(&b, "    password: %s\n", pass)
		fmt.Fprintln(&b)
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

func bandwidth5m() *Bandwidth {
	db, err := webdb.InitDB(dbPath)
	if err != nil {
		return nil
	}

	defer db.Close()

	window := int64(5 * 60)
	cutoff := time.Now().Unix() - window
	points := webdb.IfaceTotals(db, cutoff, webdb.HistoryBucket(window, 60))
	if len(points) == 0 {
		return nil
	}

	var rx, tx float64
	var n float64
	for _, p := range points {
		if p.RxBps != nil {
			rx += *p.RxBps
		}

		if p.TxBps != nil {
			tx += *p.TxBps
		}

		n++
	}

	if n == 0 {
		return nil
	}

	return &Bandwidth{RxBps: rx / n, TxBps: tx / n}
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
	links, err := vnl.LinkList()
	if err != nil {
		return nil
	}

	var lines []string
	for _, link := range links {
		name := link.Attrs().Name
		if name == "lo" || link.Attrs().Flags&unix.IFF_LOOPBACK != 0 {
			continue
		}

		switch link.Type() {
		case "veth", "bridge", "tun", "wireguard":
			continue
		}

		addrs, _ := vnl.AddrList(link, vnl.FAMILY_ALL)
		var cidrs []string
		for _, a := range addrs {
			if a.IP.IsLinkLocalUnicast() {
				continue
			}

			cidrs = append(cidrs, a.IPNet.String())
		}

		if len(cidrs) > 0 {
			lines = append(lines, fmt.Sprintf("%-12s  %s", name, strings.Join(cidrs, "  ")))
		} else {
			lines = append(lines, fmt.Sprintf("%-12s  (no address)", name))
		}
	}

	return lines
}
