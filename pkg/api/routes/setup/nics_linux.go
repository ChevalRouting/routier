//go:build linux

package setup

import (
	vnl "github.com/vishvananda/netlink"
)

func nicAddrs(name string) []string {
	link, err := vnl.LinkByName(name)
	if err != nil {
		return nil
	}

	addrs, _ := vnl.AddrList(link, vnl.FAMILY_ALL)
	var out []string
	for _, a := range addrs {
		if a.IP.IsLinkLocalUnicast() || a.IP.IsLinkLocalMulticast() {
			continue
		}

		out = append(out, a.IPNet.String())
	}

	return out
}
