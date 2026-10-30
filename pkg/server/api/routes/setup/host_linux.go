//go:build linux

package setup

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/ChevalRouting/routier/pkg/types"
	vnl "github.com/vishvananda/netlink"
)

var dhcpPidfiles = []string{"/run/dhcpcd/*.pid", "/run/udhcpc.*.pid", "/var/run/udhcpc.*.pid"}

const (
	dhcpRouteProtocol = 16
	infiniteLifetime  = 0xffffffff
)

func currentHost() (types.SetupHost, error) {
	host := types.SetupHost{DhcpIfaces: dhcpIfaces()}

	routes, err := vnl.RouteList(nil, vnl.FAMILY_V4)
	if err != nil {
		return host, err
	}

	for _, r := range routes {
		if !isDefaultRoute(r) {
			continue
		}

		link, err := vnl.LinkByIndex(r.LinkIndex)
		if err != nil {
			continue
		}

		host.DefaultIface = link.Attrs().Name
		if r.Protocol == dhcpRouteProtocol || hasLeasedAddress(link) {
			host.DhcpIfaces = appendUnique(host.DhcpIfaces, host.DefaultIface)
		}

		if r.Gw != nil {
			host.Gateway4 = r.Gw.String()
		}

		break
	}

	return host, nil
}

func isDefaultRoute(r vnl.Route) bool {
	if r.Dst == nil {
		return true
	}

	ones, _ := r.Dst.Mask.Size()
	return ones == 0
}

func hasLeasedAddress(link vnl.Link) bool {
	addrs, err := vnl.AddrList(link, vnl.FAMILY_V4)
	if err != nil {
		return false
	}

	for _, a := range addrs {
		if a.IP.IsGlobalUnicast() && a.ValidLft > 0 && a.ValidLft != infiniteLifetime {
			return true
		}
	}

	return false
}

func appendUnique(names []string, name string) []string {
	if name == "" || slices.Contains(names, name) {
		return names
	}

	return append(names, name)
}

func dhcpIfaces() []string {
	names := []string{}
	for _, pattern := range dhcpPidfiles {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			names = appendUnique(names, pidfileIface(filepath.Base(m)))
		}
	}

	return names
}

func pidfileIface(base string) string {
	base = strings.TrimSuffix(base, ".pid")
	if after, ok := strings.CutPrefix(base, "udhcpc."); ok {
		return after
	}

	if strings.HasSuffix(base, "-6") {
		return ""
	}

	return strings.TrimSuffix(base, "-4")
}
