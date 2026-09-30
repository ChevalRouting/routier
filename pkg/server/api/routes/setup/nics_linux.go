//go:build linux

package setup

import (
	"github.com/ChevalRouting/routier/pkg/net/netlink"
	"github.com/ChevalRouting/routier/pkg/types"
)

func systemNics() ([]types.SystemNic, error) {
	nics, err := netlink.SystemNics()
	if err != nil {
		return nil, err
	}

	out := make([]types.SystemNic, 0, len(nics))
	for _, n := range nics {
		out = append(out, types.SystemNic{
			Name:      n.Name,
			MAC:       n.MAC,
			Operstate: n.Operstate,
			Addrs:     n.Addrs,
			Physical:  n.Physical,
			Driver:    n.Driver,
			Speed:     n.Speed,
			Duplex:    n.Duplex,
			Carrier:   n.Carrier,
			PCIVendor: n.PCIVendor,
			PCIDevice: n.PCIDevice,
		})
	}

	return out, nil
}
