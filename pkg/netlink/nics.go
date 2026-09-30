//go:build linux

package netlink

import (
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	vnl "github.com/vishvananda/netlink"
)

type Nic struct {
	Name      string
	MAC       string
	Operstate string
	Physical  bool
	Addrs     []string
	Driver    string
	Speed     int
	Duplex    string
	Carrier   bool
	PCIVendor string
	PCIDevice string
}

func SystemNics() ([]Nic, error) {
	links, err := vnl.LinkList()
	if err != nil {
		return nil, err
	}

	nics := []Nic{}
	for _, l := range links {
		if !isBaseNic(l) {
			continue
		}

		a := l.Attrs()
		nic := Nic{
			Name:      a.Name,
			MAC:       a.HardwareAddr.String(),
			Operstate: a.OperState.String(),
			Physical:  isPhysical(l),
			Addrs:     nicAddrs(l),
		}

		if nic.Physical {
			readNicHardware(&nic)
		}

		nics = append(nics, nic)
	}

	sort.Slice(nics, func(i, j int) bool { return nics[i].Name < nics[j].Name })
	return nics, nil
}

func readNicHardware(nic *Nic) {
	base := filepath.Join("/sys/class/net", nic.Name)

	if link, err := os.Readlink(filepath.Join(base, "device", "driver")); err == nil {
		nic.Driver = filepath.Base(link)
	}

	if b, err := os.ReadFile(filepath.Join(base, "speed")); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil && n > 0 {
			nic.Speed = n
		}
	}

	if b, err := os.ReadFile(filepath.Join(base, "duplex")); err == nil {
		nic.Duplex = strings.TrimSpace(string(b))
	}

	if b, err := os.ReadFile(filepath.Join(base, "carrier")); err == nil {
		nic.Carrier = strings.TrimSpace(string(b)) == "1"
	}

	if b, err := os.ReadFile(filepath.Join(base, "device", "vendor")); err == nil {
		nic.PCIVendor = strings.TrimSpace(string(b))
	}

	if b, err := os.ReadFile(filepath.Join(base, "device", "device")); err == nil {
		nic.PCIDevice = strings.TrimSpace(string(b))
	}
}

func isBaseNic(l vnl.Link) bool {
	if l.Attrs().Flags&net.FlagLoopback != 0 {
		return false
	}

	switch l.(type) {
	case *vnl.Device, *vnl.Macvlan, *vnl.Macvtap, *vnl.Bond:
		return true
	}

	return false
}

func isPhysical(l vnl.Link) bool {
	_, ok := l.(*vnl.Device)
	return ok
}

func nicAddrs(l vnl.Link) []string {
	addrs, _ := vnl.AddrList(l, vnl.FAMILY_ALL)
	var out []string
	for _, a := range addrs {
		if a.IP.IsLinkLocalUnicast() || a.IP.IsLinkLocalMulticast() {
			continue
		}

		out = append(out, a.IPNet.String())
	}

	return out
}
