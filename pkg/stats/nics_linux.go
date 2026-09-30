//go:build linux

package stats

import "github.com/ChevalRouting/routier/pkg/net/netlink"

func PhysicalIfaces() map[string]bool {
	nics, err := netlink.SystemNics()
	if err != nil {
		return map[string]bool{}
	}

	out := map[string]bool{}
	for _, n := range nics {
		if n.Physical {
			out[n.Name] = true
		}
	}

	return out
}
