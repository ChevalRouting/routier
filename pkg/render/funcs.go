package render

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
)

func ifaceAddresses(cfg *config.Config, name string) []string {
	if i, ok := cfg.Interfaces[name]; ok {
		return i.Addresses
	}

	if t, ok := cfg.Tunnels[name]; ok {
		return t.Addresses
	}

	if w, ok := cfg.Wireguard[name]; ok {
		return w.Addresses
	}

	for _, iface := range cfg.Interfaces {
		if v, ok := iface.VLANs[name]; ok {
			return v.Addresses
		}
	}

	return nil
}

func findVLANDevice(cfg *config.Config, name string) string {
	for _, iface := range cfg.Interfaces {
		if v, ok := iface.VLANs[name]; ok {
			return v.Device
		}
	}

	return ""
}

func neighborDefaultAF(addr string) string {
	ip := net.ParseIP(addr)
	if ip != nil && ip.To4() == nil {
		return "ipv6-unicast"
	}

	return "ipv4-unicast"
}

func isStaticAddr(addr string) bool {
	switch addr {
	case "dhcp", "dhcp4", "dhcp6", "slaac":
		return false
	}

	return true
}

func sanitizeNftName(s string) string {
	return strings.NewReplacer(":", "_", ".", "_", "-", "_").Replace(s)
}

func templateFuncs(data TemplateData) template.FuncMap {
	cfg := data.Config
	return template.FuncMap{
		"iface": func(name string) string {
			if i, ok := cfg.Interfaces[name]; ok {
				return i.Device
			}

			if dev := findVLANDevice(cfg, name); dev != "" {
				return dev
			}

			return name
		},
		"vlan": func(ifaceName, vlanName string) string {
			if i, ok := cfg.Interfaces[ifaceName]; ok {
				if v, ok := i.VLANs[vlanName]; ok {
					return v.Device
				}
			}

			return ""
		},
		"addr": func(ifaceName string) string {
			addrs := ifaceAddresses(cfg, ifaceName)
			if len(addrs) > 0 {
				return addrs[0]
			}

			return ""
		},
		"addr4": func(ifaceName string) string {
			addrs := ifaceAddresses(cfg, ifaceName)
			for _, a := range addrs {
				h, _, err := net.ParseCIDR(a)
				if err != nil || h.To4() == nil {
					continue
				}

				return h.String()
			}

			return ""
		},
		"addr6": func(ifaceName string) string {
			addrs := ifaceAddresses(cfg, ifaceName)
			for _, a := range addrs {
				h, _, err := net.ParseCIDR(a)
				if err != nil || h.To4() != nil {
					continue
				}

				return h.String()
			}

			return ""
		},
		"network4": func(ifaceName string) string {
			addrs := ifaceAddresses(cfg, ifaceName)
			for _, a := range addrs {
				h, n, err := net.ParseCIDR(a)
				if err != nil || h.To4() == nil {
					continue
				}

				return n.String()
			}

			return ""
		},
		"network6": func(ifaceName string) string {
			addrs := ifaceAddresses(cfg, ifaceName)
			for _, a := range addrs {
				h, n, err := net.ParseCIDR(a)
				if err != nil || h.To4() != nil {
					continue
				}

				return n.String()
			}

			return ""
		},
		"gateway": func(ifaceName string) string {
			if cfg.Routing == nil {
				return ""
			}

			dev := ""
			if i, ok := cfg.Interfaces[ifaceName]; ok {
				dev = i.Device
			}

			for _, r := range cfg.Routing.Static {
				if r.Destination != "0.0.0.0/0" || r.Via == "" {
					continue
				}

				if r.Dev != "" && r.Dev != dev && r.Dev != ifaceName {
					continue
				}

				if !strings.Contains(r.Via, ":") {
					return r.Via
				}
			}

			return ""
		},
		"gateway6": func(ifaceName string) string {
			if cfg.Routing == nil {
				return ""
			}

			dev := ""
			if i, ok := cfg.Interfaces[ifaceName]; ok {
				dev = i.Device
			}

			for _, r := range cfg.Routing.Static {
				if r.Destination != "::/0" || r.Via == "" {
					continue
				}

				if r.Dev != "" && r.Dev != dev && r.Dev != ifaceName {
					continue
				}

				if strings.Contains(r.Via, ":") {
					return r.Via
				}
			}

			return ""
		},
		"ip": func(cidr string) string {
			h, _, err := net.ParseCIDR(cidr)
			if err != nil {
				return cidr
			}

			return h.String()
		},
		"subnet": func(cidr string) string {
			_, n, err := net.ParseCIDR(cidr)
			if err != nil {
				return cidr
			}

			return n.String()
		},
		"join":     strings.Join,
		"contains": strings.Contains,
		"extraLines": func(lines []string, indent string) string {
			var b strings.Builder
			for _, l := range lines {
				if l = strings.TrimSpace(l); l != "" {
					b.WriteString("\n" + indent + l)
				}
			}

			return b.String()
		},
		"quote": func(s string) string {
			return fmt.Sprintf("%q", s)
		},
		"default": func(def, val string) string {
			if val == "" {
				return def
			}

			return val
		},
		"allDevices": func() []string {
			var devs []string
			for _, i := range cfg.Interfaces {
				devs = append(devs, i.Device)

				for _, v := range i.VLANs {
					devs = append(devs, v.Device)
				}
			}

			return devs
		},
		"afName": func(s string) string {
			return strings.ReplaceAll(s, "-", " ")
		},
		"bgpAFs": func(bgp *config.BGP) []string {
			seen := map[string]struct{}{}
			for af := range bgp.AddressFamilies {
				seen[af] = struct{}{}
			}

			for _, n := range bgp.Neighbors {
				if len(n.AddressFamilies) > 0 {
					for af := range n.AddressFamilies {
						seen[af] = struct{}{}
					}
				} else {
					seen[neighborDefaultAF(n.Address)] = struct{}{}
				}
			}

			afs := make([]string, 0, len(seen))
			for af := range seen {
				afs = append(afs, af)
			}

			sort.Strings(afs)
			return afs
		},
		"bgpNeighborInAF": func(n config.BGPNeighbor, af string) *config.BGPNeighborAF {
			if len(n.AddressFamilies) > 0 {
				return n.AddressFamilies[af]
			}

			if neighborDefaultAF(n.Address) == af {
				return &config.BGPNeighborAF{}
			}

			return nil
		},
		"nftDefines": func() string {
			var b strings.Builder

			nftSet := func(vals []string) string {
				if len(vals) == 1 {
					return vals[0]
				}

				return "{ " + strings.Join(vals, ", ") + " }"
			}

			var collectAddrs func(addresses []string)

			emitAddrDefines := func(prefix string, addresses []string) {
				var v4h, v4n, v6h, v6n []string
				for _, addr := range addresses {
					if !isStaticAddr(addr) {
						continue
					}

					h, n, err := net.ParseCIDR(addr)
					if err != nil {
						ip := net.ParseIP(strings.TrimSpace(addr))
						if ip == nil {
							continue
						}

						if ip.To4() != nil {
							v4h = append(v4h, ip.String())
						} else {
							v6h = append(v6h, ip.String())
						}

						continue
					}

					if h.To4() != nil {
						v4h = append(v4h, h.String())
						v4n = append(v4n, n.String())
					} else {
						v6h = append(v6h, h.String())
						v6n = append(v6n, n.String())
					}
				}

				if len(v4h) > 0 {
					fmt.Fprintf(&b, "define %s_address = %s\n", prefix, nftSet(v4h))
				}

				if len(v4n) > 0 {
					fmt.Fprintf(&b, "define %s_network = %s\n", prefix, nftSet(v4n))
				}

				if len(v6h) > 0 {
					fmt.Fprintf(&b, "define %s_address6 = %s\n", prefix, nftSet(v6h))
				}

				if len(v6n) > 0 {
					fmt.Fprintf(&b, "define %s_network6 = %s\n", prefix, nftSet(v6n))
				}
			}

			writeAddrDefines := func(prefix string, addresses []string) {
				collectAddrs(addresses)
				emitAddrDefines(prefix, addresses)
			}

			var ifaceDevs, tunnelDevs, wgDevs []string
			var meV4, meV6 []string
			vrfMembers := map[string][]string{}

			collectAddrs = func(addresses []string) {
				for _, addr := range addresses {
					if !isStaticAddr(addr) {
						continue
					}

					h, _, err := net.ParseCIDR(addr)
					if err != nil {
						continue
					}

					if h.To4() != nil {
						meV4 = append(meV4, h.String())
					} else {
						meV6 = append(meV6, h.String())
					}
				}
			}

			ifaceNames := make([]string, 0, len(cfg.Interfaces))
			for n := range cfg.Interfaces {
				ifaceNames = append(ifaceNames, n)
			}

			sort.Strings(ifaceNames)

			vrrpByID := map[int][]string{}
			var vrrpIDs []int

			for _, name := range ifaceNames {
				iface := cfg.Interfaces[name]
				sanitized := sanitizeNftName(name)
				dev := iface.Device
				if dev == "" {
					dev = iface.Select
				}

				fmt.Fprintf(&b, "define %s_interfaces = \"%s\"\n", sanitized, dev)
				if dev != "" {
					ifaceDevs = append(ifaceDevs, "\""+dev+"\"")
					if iface.VRF != "" {
						vrfMembers[iface.VRF] = append(vrfMembers[iface.VRF], "\""+dev+"\"")
					}
				}

				writeAddrDefines(sanitized, iface.Addresses)

				var vips []string
				for _, v := range iface.VRRP {
					vips = append(vips, v.VIPs...)
					if _, seen := vrrpByID[v.ID]; !seen {
						vrrpIDs = append(vrrpIDs, v.ID)
					}

					vrrpByID[v.ID] = append(vrrpByID[v.ID], v.VIPs...)
				}

				if len(vips) > 0 {
					emitVRRPDefines(&b, nftSet, emitAddrDefines, sanitized+"_vrrp", vips)
				}

				if cfg.Routing != nil {
					for _, r := range cfg.Routing.Static {
						if r.Via == "" || r.Dev != iface.Device {
							continue
						}

						if r.Destination != "0.0.0.0/0" && r.Destination != "::/0" {
							continue
						}

						if strings.Contains(r.Via, ":") {
							fmt.Fprintf(&b, "define %s_gateway6 = %s\n", sanitized, r.Via)
						} else {
							fmt.Fprintf(&b, "define %s_gateway = %s\n", sanitized, r.Via)
						}
					}
				}

				vlanNames := make([]string, 0, len(iface.VLANs))
				for n := range iface.VLANs {
					vlanNames = append(vlanNames, n)
				}

				sort.Strings(vlanNames)
				for _, vname := range vlanNames {
					vlan := iface.VLANs[vname]
					vsanitized := sanitizeNftName(vname)
					fmt.Fprintf(&b, "define %s_interfaces = \"%s\"\n", vsanitized, vlan.Device)

					ifaceDevs = append(ifaceDevs, "\""+vlan.Device+"\"")
					writeAddrDefines(vsanitized, vlan.Addresses)
				}
			}

			sort.Ints(vrrpIDs)
			for _, id := range vrrpIDs {
				emitVRRPDefines(&b, nftSet, emitAddrDefines, "vrrp_"+strconv.Itoa(id), vrrpByID[id])
			}

			tunnelNames := make([]string, 0, len(cfg.Tunnels))
			for n := range cfg.Tunnels {
				tunnelNames = append(tunnelNames, n)
			}

			sort.Strings(tunnelNames)

			for _, name := range tunnelNames {
				tunnel := cfg.Tunnels[name]
				sanitized := sanitizeNftName(name)
				fmt.Fprintf(&b, "define %s_interfaces = \"%s\"\n", sanitized, name)
				tunnelDevs = append(tunnelDevs, "\""+name+"\"")
				writeAddrDefines(sanitized, tunnel.Addresses)
			}

			wgNames := make([]string, 0, len(cfg.Wireguard))
			for n := range cfg.Wireguard {
				wgNames = append(wgNames, n)
			}

			sort.Strings(wgNames)

			for _, name := range wgNames {
				wg := cfg.Wireguard[name]
				sanitized := sanitizeNftName(name)
				fmt.Fprintf(&b, "define %s_interfaces = \"%s\"\n", sanitized, name)
				wgDevs = append(wgDevs, "\""+name+"\"")
				writeAddrDefines(sanitized, wg.Addresses)
			}

			vrfNames := make([]string, 0, len(cfg.VRFs))
			for n := range cfg.VRFs {
				vrfNames = append(vrfNames, n)
			}

			sort.Strings(vrfNames)

			var vrfDevs []string
			for _, name := range vrfNames {
				sanitized := sanitizeNftName(name)
				fmt.Fprintf(&b, "define vrf_%s_interfaces = \"%s\"\n", sanitized, name)
				if members := vrfMembers[name]; len(members) > 0 {
					fmt.Fprintf(&b, "define vrf_%s_members = %s\n", sanitized, nftSet(members))
				}

				vrfDevs = append(vrfDevs, "\""+name+"\"")
			}

			if len(ifaceDevs) > 0 {
				fmt.Fprintf(&b, "define interfaces = %s\n", nftSet(ifaceDevs))
			}

			if len(tunnelDevs) > 0 {
				fmt.Fprintf(&b, "define tunnels = %s\n", nftSet(tunnelDevs))
			}

			if len(wgDevs) > 0 {
				fmt.Fprintf(&b, "define wireguard = %s\n", nftSet(wgDevs))
			}

			if len(vrfDevs) > 0 {
				fmt.Fprintf(&b, "define vrfs = %s\n", nftSet(vrfDevs))
			}

			if cfg.DNS != nil && len(cfg.DNS.Nameservers) > 0 {
				fmt.Fprintf(&b, "define dns_nameservers = %s\n", nftSet(cfg.DNS.Nameservers))
			}

			if len(meV4) > 0 {
				fmt.Fprintf(&b, "define me = %s\n", nftSet(meV4))
			}

			if len(meV6) > 0 {
				fmt.Fprintf(&b, "define me6 = %s\n", nftSet(meV6))
			}

			emitIPSet := func(prefix string, ips []string) {
				var v4, v6 []string
				for _, ip := range ips {
					p := net.ParseIP(strings.TrimSpace(ip))
					if p == nil {
						continue
					}

					if p.To4() != nil {
						v4 = append(v4, p.String())
					} else {
						v6 = append(v6, p.String())
					}
				}

				if len(v4) > 0 {
					fmt.Fprintf(&b, "define %s = %s\n", prefix, nftSet(v4))
				}

				if len(v6) > 0 {
					fmt.Fprintf(&b, "define %s6 = %s\n", prefix, nftSet(v6))
				}
			}

			if cfg.Routing != nil {
				rt := cfg.Routing
				if rt.BGP != nil {
					neighbors := make([]string, 0, len(rt.BGP.Neighbors))
					for _, n := range rt.BGP.Neighbors {
						neighbors = append(neighbors, n.Address)
						emitIPSet("bgp_neighbor_"+sanitizeNftName(n.Address), []string{n.Address})

						if n.Description != "" {
							emitIPSet("bgp_neighbor_"+sanitizeNftName(n.Description), []string{n.Address})
						}
					}

					emitIPSet("bgp_neighbors", neighbors)
				}

				if rt.OSPF != nil && len(rt.OSPF.Interfaces) > 0 {
					ospfIfs := make([]string, 0, len(rt.OSPF.Interfaces))

					for name := range rt.OSPF.Interfaces {
						ospfIfs = append(ospfIfs, name)
					}

					sort.Strings(ospfIfs)
					fmt.Fprintf(&b, "define ospf_interfaces = %s\n", nftSet(ospfIfs))
				}

				if rt.Anycast != nil {
					var allIPs, allEndpoints []string
					for _, svc := range rt.Anycast.Services {
						san := sanitizeNftName(svc.Name)
						emitIPSet("anycast_"+san, svc.AnycastIPs)

						endpoints := make([]string, 0, len(svc.Endpoints))
						for _, e := range svc.Endpoints {
							endpoints = append(endpoints, e.IP)
						}

						emitIPSet("anycast_"+san+"_endpoints", endpoints)
						allIPs = append(allIPs, svc.AnycastIPs...)
						allEndpoints = append(allEndpoints, endpoints...)
					}

					emitIPSet("anycast_ips", allIPs)
					emitIPSet("anycast_endpoints", allEndpoints)
				}
			}

			friendNames := make([]string, 0, len(data.Friends))
			for n := range data.Friends {
				friendNames = append(friendNames, n)
			}

			sort.Strings(friendNames)
			for _, fname := range friendNames {
				fv := data.Friends[fname]
				prefix := "friends_" + sanitizeNftName(fname)

				if len(fv.Exports) > 0 {
					exportNames := make([]string, 0, len(fv.Exports))
					for n := range fv.Exports {
						exportNames = append(exportNames, n)
					}

					sort.Strings(exportNames)
					for _, en := range exportNames {
						fmt.Fprintf(&b, "define %s_%s = %s\n", prefix, en, fv.Exports[en])
					}

					continue
				}

				ifaceNames := make([]string, 0, len(fv.Interfaces))
				for n := range fv.Interfaces {
					ifaceNames = append(ifaceNames, n)
				}

				sort.Strings(ifaceNames)
				for _, iname := range ifaceNames {
					emitAddrDefines(prefix+"_"+sanitizeNftName(iname), fv.Interfaces[iname])
				}
			}

			return b.String()
		},
		"renderStr": func(s string) (string, error) {
			funcs := templateFuncs(data)
			t, err := template.New("render-str").Funcs(funcs).Parse(s)
			if err != nil {
				return "", fmt.Errorf("renderStr: %w", err)
			}

			var buf bytes.Buffer
			if err := t.Execute(&buf, data); err != nil {
				return "", fmt.Errorf("renderStr: %w", err)
			}

			return buf.String(), nil
		},
		"frrInterfaces": func() []string {
			seen := map[string]struct{}{}
			if cfg.Routing != nil {
				if cfg.Routing.OSPF != nil {
					for name := range cfg.Routing.OSPF.Interfaces {
						seen[name] = struct{}{}
					}
				}

				if cfg.Routing.OSPF6 != nil {
					for name := range cfg.Routing.OSPF6.Interfaces {
						seen[name] = struct{}{}
					}
				}

				if cfg.Routing.PBR != nil {
					for name := range cfg.Routing.PBR.Policies {
						seen[name] = struct{}{}
					}
				}
			}

			names := make([]string, 0, len(seen))
			for name := range seen {
				names = append(names, name)
			}

			sort.Strings(names)
			return names
		},
		"ospfIface": func(name string) *config.OSPFInterface {
			if cfg.Routing == nil || cfg.Routing.OSPF == nil {
				return nil
			}

			return cfg.Routing.OSPF.Interfaces[name]
		},
		"ospf6Iface": func(name string) *config.OSPF6Interface {
			if cfg.Routing == nil || cfg.Routing.OSPF6 == nil {
				return nil
			}

			return cfg.Routing.OSPF6.Interfaces[name]
		},
		"pbrPolicy": func(name string) string {
			if cfg.Routing == nil || cfg.Routing.PBR == nil {
				return ""
			}

			return cfg.Routing.PBR.Policies[name]
		},
		"bgpNeighborByDesc": func(desc string) string {
			if cfg.Routing == nil {
				return ""
			}

			if cfg.Routing.BGP != nil {
				for _, n := range cfg.Routing.BGP.Neighbors {
					if n.Description == desc {
						return n.Address
					}
				}
			}

			for _, vrf := range cfg.Routing.VRFs {
				if vrf.BGP == nil {
					continue
				}

				for _, n := range vrf.BGP.Neighbors {
					if n.Description == desc {
						return n.Address
					}
				}
			}

			return ""
		},
		"hasOSPF6": func() bool {
			if cfg.Routing == nil {
				return false
			}

			if cfg.Routing.OSPF6 != nil {
				return true
			}

			for _, vr := range cfg.Routing.VRFs {
				if vr.OSPF6 != nil {
					return true
				}
			}

			return false
		},
		"bfdProfiles": func() []config.BFDProfile {
			if cfg.Routing == nil || cfg.Routing.BFD == nil {
				return nil
			}

			return cfg.Routing.BFD.Profiles
		},
		"needBFD": func() bool {
			if cfg.Routing == nil {
				return false
			}

			if cfg.Routing.BFD != nil && len(cfg.Routing.BFD.Profiles) > 0 {
				return true
			}

			bgpHasBFD := func(bgp *config.BGP) bool {
				if bgp == nil {
					return false
				}

				for _, n := range bgp.Neighbors {
					if n.BFD {
						return true
					}
				}

				return false
			}

			if bgpHasBFD(cfg.Routing.BGP) {
				return true
			}

			for _, vr := range cfg.Routing.VRFs {
				if bgpHasBFD(vr.BGP) {
					return true
				}
			}

			return false
		},
		"nftRoutierTable": func() (string, error) {
			return renderRoutierTable(cfg)
		},
		"nftUserDefines": func() string {
			if cfg.Nftables == nil {
				return ""
			}

			return strings.TrimRight(cfg.Nftables.Defines, "\n")
		},
		"nftIncludes": func() ([]string, error) {
			if cfg.Nftables == nil {
				return nil, nil
			}

			funcs := templateFuncs(data)
			var out []string
			for _, p := range cfg.Nftables.Include {
				path := p
				if !filepath.IsAbs(path) {
					path = filepath.Join(cfg.BaseDir, path)
				}

				content, rerr := os.ReadFile(path)
				if rerr != nil {
					log.Warn().Err(rerr).Str("file", path).Msg("nftables include: skipping unreadable file")
					continue
				}

				t, terr := template.New("nft-include").Funcs(funcs).Parse(string(content))
				if terr != nil {
					return nil, fmt.Errorf("nftables include %s: %w", path, terr)
				}

				var buf bytes.Buffer
				if terr := t.Execute(&buf, data); terr != nil {
					return nil, fmt.Errorf("nftables include %s: %w", path, terr)
				}

				out = append(out, buf.String())
			}

			return out, nil
		},
	}
}

func emitVRRPDefines(b *strings.Builder, nftSet func([]string) string, emitAddr func(string, []string), prefix string, vips []string) {
	if len(vips) == 0 {
		return
	}

	emitAddr(prefix, vips)

	var hosts []string
	for _, addr := range vips {
		h, _, err := net.ParseCIDR(addr)
		if err != nil {
			if ip := net.ParseIP(strings.TrimSpace(addr)); ip != nil {
				hosts = append(hosts, ip.String())
			}

			continue
		}

		hosts = append(hosts, h.String())
	}

	if len(hosts) > 0 {
		fmt.Fprintf(b, "define %s_addresses = %s\n", prefix, nftSet(hosts))
	}
}

type ownedChain struct {
	name      string
	header    string
	defPolicy string
	filter    bool
}

var ownedChains = []ownedChain{
	{name: "input", header: "type filter hook input priority filter", defPolicy: "drop", filter: true},
	{name: "forward", header: "type filter hook forward priority filter", defPolicy: "drop", filter: true},
	{name: "output", header: "type filter hook output priority filter", defPolicy: "accept", filter: true},
	{name: "prerouting", header: "type nat hook prerouting priority dstnat", defPolicy: "accept"},
	{name: "postrouting", header: "type nat hook postrouting priority srcnat", defPolicy: "accept"},
}

func renderRoutierTable(cfg *config.Config) (string, error) {
	managed := routierManagedRules(cfg)

	var b strings.Builder
	b.WriteString("table inet routier {\n")

	for _, oc := range ownedChains {
		userLines, err := nftChainUserLines(cfg, oc.name)
		if err != nil {
			return "", err
		}

		if !oc.filter && !nftChainConfigured(cfg, oc.name) {
			continue
		}

		fmt.Fprintf(&b, "\tchain %s {\n", oc.name)
		fmt.Fprintf(&b, "\t\t%s; policy %s;\n", oc.header, nftChainPolicy(cfg, oc.name, oc.defPolicy))

		if auto := managed[oc.name]; len(auto) > 0 {
			b.WriteString("\n\t\t# --- routier auto rules ---\n")
			for _, l := range auto {
				fmt.Fprintf(&b, "\t\t%s\n", l)
			}
		}

		if len(userLines) > 0 {
			if len(managed[oc.name]) > 0 {
				b.WriteString("\n\t\t# --- user rules ---\n")
			}

			for _, l := range userLines {
				if strings.TrimSpace(l) == "" {
					b.WriteString("\n")
				} else {
					fmt.Fprintf(&b, "\t\t%s\n", l)
				}
			}
		}

		b.WriteString("\t}\n")
	}

	b.WriteString("}\n")
	return b.String(), nil
}

func nftChainConfigured(cfg *config.Config, name string) bool {
	return cfg.Nftables != nil && cfg.Nftables.Chains[name] != nil
}

func nftChainPolicy(cfg *config.Config, name, def string) string {
	if cfg.Nftables != nil {
		if ch := cfg.Nftables.Chains[name]; ch != nil && ch.Policy != "" {
			return ch.Policy
		}
	}

	return def
}

func nftChainUserLines(cfg *config.Config, name string) ([]string, error) {
	if cfg.Nftables == nil {
		return nil, nil
	}

	ch := cfg.Nftables.Chains[name]
	if ch == nil {
		return nil, nil
	}

	var lines []string
	if block := strings.TrimRight(ch.Rules, "\n"); block != "" {
		lines = append(lines, strings.Split(block, "\n")...)
	}

	for _, f := range ch.Files {
		p := f
		if !filepath.IsAbs(p) {
			p = filepath.Join(cfg.BaseDir, p)
		}

		content, rerr := os.ReadFile(p)
		if rerr != nil {
			log.Warn().Err(rerr).Str("file", p).Msg("nftables chain file: skipping unreadable file")
			continue
		}

		lines = append(lines, splitRuleLines(string(content))...)
	}

	lines = append(lines, managedRuleLines(ch.Managed)...)

	return lines, nil
}

func routierManagedRules(cfg *config.Config) map[string][]string {
	out := map[string][]string{}
	add := func(chain, rule string) { out[chain] = append(out[chain], rule) }

	var wgPorts []int
	seenPort := map[int]bool{}
	for _, wg := range cfg.Wireguard {
		if wg.AllowInbound && wg.ListenPort > 0 && !seenPort[wg.ListenPort] {
			seenPort[wg.ListenPort] = true
			wgPorts = append(wgPorts, wg.ListenPort)
		}
	}

	if len(wgPorts) > 0 {
		sort.Ints(wgPorts)
		ports := make([]string, len(wgPorts))
		for i, p := range wgPorts {
			ports[i] = strconv.Itoa(p)
		}

		set := ports[0]
		if len(ports) > 1 {
			set = "{ " + strings.Join(ports, ", ") + " }"
		}

		add("input", fmt.Sprintf("udp dport %s accept comment \"routier: wireguard\"", set))
	}

	if cfg.Conntrackd != nil && cfg.Conntrackd.AllowInbound && cfg.Conntrackd.Interface != "" {
		port := cfg.Conntrackd.Port
		if port == 0 {
			port = 3780
		}

		add("input", fmt.Sprintf("iifname %q udp dport %d accept comment \"routier: conntrackd\"", cfg.Conntrackd.Interface, port))
	}

	seenDev := map[string]bool{}
	ifaceNames := make([]string, 0, len(cfg.Interfaces))
	for name := range cfg.Interfaces {
		ifaceNames = append(ifaceNames, name)
	}

	sort.Strings(ifaceNames)
	for _, name := range ifaceNames {
		iface := cfg.Interfaces[name]

		for _, v := range iface.VRRP {
			if !v.AllowInbound {
				continue
			}

			dev := iface.Device
			if v.Interface != "" {
				dev = v.Interface
			}

			if dev == "" || seenDev[dev] {
				continue
			}

			seenDev[dev] = true
			add("input", fmt.Sprintf("iifname %q meta l4proto 112 accept comment \"routier: vrrp\"", dev))
		}
	}

	if r := cfg.Routing; r != nil {
		if r.BGP != nil {
			for _, ifn := range r.BGP.AllowInbound {
				out["input"] = append(out["input"], scopedAllow(cfg, ifn, "tcp dport 179", "bgp")...)
			}
		}

		if r.OSPF != nil {
			for _, ifn := range r.OSPF.AllowInbound {
				out["input"] = append(out["input"], scopedAllow(cfg, ifn, "meta l4proto 89", "ospf", "ip")...)
			}
		}

		if r.OSPF6 != nil {
			for _, ifn := range r.OSPF6.AllowInbound {
				out["input"] = append(out["input"], scopedAllow(cfg, ifn, "meta l4proto 89", "ospf6", "ip6")...)
			}
		}
	}

	return out
}

func ifaceStaticFamilies(iface *config.Interface) (v4, v6 bool) {
	for _, addr := range iface.Addresses {
		if !isStaticAddr(addr) {
			continue
		}

		h, _, err := net.ParseCIDR(addr)
		if err != nil {
			continue
		}

		if h.To4() != nil {
			v4 = true
		} else {
			v6 = true
		}
	}

	return
}

func scopedAllow(cfg *config.Config, ifaceName, match, comment string, fams ...string) []string {
	iface := cfg.Interfaces[ifaceName]
	if iface == nil {
		return nil
	}

	san := sanitizeNftName(ifaceName)
	v4, v6 := ifaceStaticFamilies(iface)

	want := func(f string) bool {
		if len(fams) == 0 {
			return true
		}

		return slices.Contains(fams, f)
	}

	var out []string
	if v4 && want("ip") {
		out = append(out, fmt.Sprintf("iifname $%s_interfaces ip saddr $%s_network %s accept comment %q", san, san, match, "routier: "+comment))
	}

	if v6 && want("ip6") {
		out = append(out, fmt.Sprintf("iifname $%s_interfaces ip6 saddr $%s_network6 %s accept comment %q", san, san, match, "routier: "+comment))
	}

	if len(out) == 0 {
		out = append(out, fmt.Sprintf("iifname $%s_interfaces %s accept comment %q", san, match, "routier: "+comment))
	}

	return out
}

func splitRuleLines(s string) []string {
	var lines []string

	for _, raw := range strings.Split(s, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		lines = append(lines, line)
	}

	return lines
}

func managedRuleLines(rules []config.ManagedRule) []string {
	var lines []string

	for _, rule := range rules {
		if rule.Disabled {
			continue
		}

		line := strings.Join(buildRuleParts(&rule), " ")

		comment := rule.Comment
		if comment == "" && rule.Tag != "" {
			comment = "routier:" + rule.Tag
		}

		if comment != "" {
			line = fmt.Sprintf("%s comment %q", line, comment)
		}

		lines = append(lines, line)
	}

	return lines
}

func buildRuleParts(rule *config.ManagedRule) []string {
	var parts []string

	m := rule.Match
	if m != nil {
		if m.CTState != "" {
			parts = append(parts, "ct state", m.CTState)
		}

		hasPort := m.SPort != "" || m.DPort != ""
		if m.Protocol != "" && !hasPort {
			parts = append(parts, "meta l4proto", m.Protocol)
		}

		if m.IIF != "" {
			parts = append(parts, "iifname", formatNftSet(m.IIF))
		}

		if m.OIF != "" {
			parts = append(parts, "oifname", formatNftSet(m.OIF))
		}

		af := m.AddrFamily
		if af == "" && (m.SAddr != "" || m.DAddr != "") {
			candidate := m.SAddr
			if candidate == "" {
				candidate = m.DAddr
			}

			if strings.Contains(candidate, ":") {
				af = "ip6"
			} else {
				af = "ip"
			}
		}

		if m.SAddr != "" {
			parts = append(parts, af+" saddr", formatNftSet(m.SAddr))
		}

		if m.DAddr != "" {
			parts = append(parts, af+" daddr", formatNftSet(m.DAddr))
		}

		if m.SPort != "" {
			proto := m.Protocol
			if proto == "" {
				proto = "tcp"
			}

			parts = append(parts, proto+" sport", formatNftSet(m.SPort))
		}

		if m.DPort != "" {
			proto := m.Protocol
			if proto == "" {
				proto = "tcp"
			}

			parts = append(parts, proto+" dport", formatNftSet(m.DPort))
		}
	}

	switch rule.Action {
	case "dnat", "snat":
		if rule.ActionTo == "" {
			parts = append(parts, rule.Action)
		} else if fam := natTargetFamily(rule.ActionTo); fam != "" {
			parts = append(parts, rule.Action, fam, "to", rule.ActionTo)
		} else {
			parts = append(parts, rule.Action+" to "+rule.ActionTo)
		}
	default:
		parts = append(parts, rule.Action)
	}

	return parts
}

func natTargetFamily(target string) string {
	host := target
	if strings.HasPrefix(host, "[") {
		if i := strings.Index(host, "]"); i > 0 {
			host = host[1:i]
		}
	} else if strings.Count(host, ":") == 1 {
		host = host[:strings.LastIndex(host, ":")]
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return ""
	}

	if ip.To4() != nil {
		return "ip"
	}

	return "ip6"
}

func formatNftSet(v string) string {
	if strings.HasPrefix(v, "$") {
		return v
	}

	if strings.Contains(v, ",") {
		return "{ " + v + " }"
	}

	return v
}

type NftVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func NftVars(cfg *config.Config, opts ...Option) []NftVar {
	data := TemplateData{Config: cfg}
	for _, o := range opts {
		o(&data)
	}

	funcs := templateFuncs(data)
	definesFunc := funcs["nftDefines"].(func() string)
	raw := definesFunc()

	vars := []NftVar{}
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "define ") {
			continue
		}

		rest := strings.TrimPrefix(line, "define ")
		if idx := strings.Index(rest, " ="); idx > 0 {
			vars = append(vars, NftVar{
				Name:  rest[:idx],
				Value: strings.TrimSpace(rest[idx+2:]),
			})
		}
	}

	sort.Slice(vars, func(i, j int) bool { return vars[i].Name < vars[j].Name })

	return vars
}

func FriendNftVars(cfg *config.Config, name string, opts ...Option) []NftVar {
	prefix := "friends_" + sanitizeNftName(name) + "_"
	out := []NftVar{}

	for _, v := range NftVars(cfg, opts...) {
		if strings.HasPrefix(v.Name, prefix) {
			out = append(out, v)
		}
	}

	return out
}
