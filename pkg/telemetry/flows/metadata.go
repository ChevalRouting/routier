package flows

import (
	"net"
	"net/netip"
	"sort"

	"github.com/ChevalRouting/routier/pkg/config"
)

type MetadataLink struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Speed       uint64 `json:"speed"`
}

type MetadataInterface struct {
	IfIndex int `json:"ifindex"`
	MetadataLink
}

type MetadataExporter struct {
	ExporterSubnet string              `json:"exportersubnet"`
	Name           string              `json:"name"`
	Default        MetadataLink        `json:"default"`
	Interfaces     []MetadataInterface `json:"interfaces"`
}

type Metadata struct {
	Exporters []MetadataExporter `json:"exporters"`
}

type NetInterface struct {
	Index int
	Name  string
}

func ExporterAddresses(s Settings) []netip.Addr {
	if s.Source.IsValid() {
		return []netip.Addr{s.Source}
	}

	seen := map[netip.Addr]bool{}
	var out []netip.Addr
	for _, addr := range s.Collectors {
		conn, err := net.DialUDP("udp", nil, net.UDPAddrFromAddrPort(addr))
		if err != nil {
			continue
		}

		local := conn.LocalAddr().(*net.UDPAddr).AddrPort().Addr().Unmap()
		_ = conn.Close()
		if !seen[local] {
			seen[local] = true
			out = append(out, local)
		}
	}

	return out
}

func configNames(cfg *config.Config) map[string]string {
	names := map[string]string{}
	for name, iface := range cfg.Interfaces {
		if iface.Device != "" {
			names[iface.Device] = name
		}
	}

	return names
}

func BuildMetadata(cfg *config.Config, exporters []netip.Addr, ifaces []NetInterface, speed func(string) uint64) Metadata {
	names := configNames(cfg)
	interfaces := make([]MetadataInterface, 0, len(ifaces))
	for _, iface := range ifaces {
		interfaces = append(interfaces, MetadataInterface{
			IfIndex:      iface.Index,
			MetadataLink: MetadataLink{Name: iface.Name, Description: names[iface.Name], Speed: speed(iface.Name) / 1_000_000},
		})
	}

	sort.Slice(interfaces, func(i, j int) bool { return interfaces[i].IfIndex < interfaces[j].IfIndex })

	name := cfg.Hostname
	if name == "" {
		name = "routier"
	}

	out := Metadata{Exporters: []MetadataExporter{}}
	for _, addr := range exporters {
		out.Exporters = append(out.Exporters, MetadataExporter{
			ExporterSubnet: netip.PrefixFrom(addr, addr.BitLen()).String(),
			Name:           name,
			Default:        MetadataLink{Name: "unknown"},
			Interfaces:     interfaces,
		})
	}

	return out
}

func SystemInterfaces() []NetInterface {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	out := make([]NetInterface, 0, len(ifaces))
	for _, iface := range ifaces {
		out = append(out, NetInterface{Index: iface.Index, Name: iface.Name})
	}

	return out
}
