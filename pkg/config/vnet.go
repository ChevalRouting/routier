package config

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"net"
	"net/netip"
	"slices"
)

func VNetLinkName(name, role string) string {
	return fmt.Sprintf("rt%s%x", role, sha256.Sum256([]byte(name)))[:15]
}

func VNetGatewayMAC(vn *VNet) string {
	if vn.Gateway != nil && vn.Gateway.MAC != "" {
		return vn.Gateway.MAC
	}

	return fmt.Sprintf("02:cb:%02x:%02x:%02x:01", byte(vn.VNI>>16), byte(vn.VNI>>8), byte(vn.VNI))
}

func validateVNetGateway(name string, gw *VNetGateway, add addfunc) {
	if len(gw.IPs) == 0 {
		add("interfaces.%s.vnet.gateway: at least one ip is required", name)
	}

	for _, ip := range gw.IPs {
		if addr, err := netip.ParseAddr(ip); err != nil || addr.IsUnspecified() || addr.IsMulticast() {
			add("interfaces.%s.vnet.gateway: %q is not a host address", name, ip)
		}
	}

	if gw.MAC == "" {
		return
	}

	if mac, err := net.ParseMAC(gw.MAC); err != nil || len(mac) != 6 || mac[0]&1 == 1 {
		add("interfaces.%s.vnet.gateway: mac %q must be a unicast ethernet address", name, gw.MAC)
	}
}

func InterfaceVXLAN(iface *Interface) *VXLAN {
	if iface.VXLAN == nil || iface.Type != "vxlan-bridge" {
		return iface.VXLAN
	}

	vx := *iface.VXLAN
	learning := false
	vx.External = true
	vx.Learning = &learning
	return &vx
}

func validateVNets(cfg *Config, add addfunc) {
	vlans := map[string]map[int]string{}
	vnis := map[int]string{}
	generated := map[string]string{}
	for name, iface := range cfg.Interfaces {
		roles := []string{}
		if iface.Type == "vxlan-bridge" {
			roles = []string{"x"}
			if iface.MTU == 0 {
				add("interfaces.%s: vxlan-bridge requires an explicit overlay mtu", name)
			}

			if iface.VXLAN == nil || iface.VXLAN.Local == "" {
				add("interfaces.%s: vxlan-bridge requires a local VTEP address", name)
			}

			if iface.VXLAN != nil && (iface.VXLAN.VNIFilter || iface.VXLAN.Remote != "" || iface.VXLAN.Group != "" || (iface.VXLAN.Learning != nil && *iface.VXLAN.Learning)) {
				add("interfaces.%s: vxlan-bridge uses EVPN; vnifilter, remote, group and learning are not supported", name)
			}

			if len(iface.Addresses) != 0 || iface.VRF != "" {
				add("interfaces.%s: put addresses and vrf on a vnet, not the transport bridge", name)
			}
		}

		if iface.Type != "vnet" {
			if iface.VNet != nil {
				add("interfaces.%s: vnet settings require type vnet", name)
			}
		} else {
			roles = []string{"a", "b"}
			parent := cfg.Interfaces[iface.Select]
			if parent == nil || parent.Type != "vxlan-bridge" {
				add("interfaces.%s: vnet parent must be a vxlan-bridge", name)
			}

			if iface.VNet == nil {
				add("interfaces.%s: vnet settings are required", name)
			} else {
				vn := iface.VNet
				if parent != nil && parent.MTU > 0 && iface.MTU > parent.MTU {
					add("interfaces.%s: vnet mtu cannot exceed transport mtu", name)
				}

				if vn.VLAN < 1 || vn.VLAN > 4094 {
					add("interfaces.%s.vnet: vlan must be between 1 and 4094", name)
				}

				if vn.VNI < 1 || vn.VNI > 16777215 {
					add("interfaces.%s.vnet: vni must be between 1 and 16777215", name)
				}

				if vlans[iface.Select] == nil {
					vlans[iface.Select] = map[int]string{}
				}

				if other := vlans[iface.Select][vn.VLAN]; other != "" {
					add("interfaces.%s.vnet: vlan already used by %s", name, other)
				}

				if other := vnis[vn.VNI]; other != "" {
					add("interfaces.%s.vnet: vni already used by %s", name, other)
				}

				vlans[iface.Select][vn.VLAN] = name
				vnis[vn.VNI] = name
				if vn.Gateway != nil {
					roles = append(roles, "g")
					validateVNetGateway(name, vn.Gateway, add)
				}
			}

			if iface.VLAN != nil || iface.Bond != nil || iface.VXLAN != nil {
				add("interfaces.%s: vnet cannot have vlan, bond or vxlan settings", name)
			}
		}

		for _, role := range roles {
			dev := VNetLinkName(name, role)
			if cfg.Interfaces[dev] != nil || cfg.VRFs[dev] != nil || cfg.VPN.Wireguard[dev] != nil || generated[dev] != "" {
				add("interfaces.%s: generated device %s conflicts with another interface", name, dev)
			}

			generated[dev] = name
		}
	}

	validateL3VNIs(cfg, add, vlans, vnis, generated)
}

func L3VNILinkName(vrf string) string {
	return VNetLinkName("vrf:"+vrf, "s")
}

func validateL3VNIs(cfg *Config, add addfunc, vlans map[string]map[int]string, vnis map[int]string, generated map[string]string) {
	for _, name := range slices.Sorted(maps.Keys(cfg.VRFs)) {
		l3 := cfg.VRFs[name].L3VNI
		if l3 == nil {
			continue
		}

		owner := "vrf " + name
		if l3.VNI < 1 || l3.VNI > 16777215 {
			add("vrfs.%s.l3vni: vni must be between 1 and 16777215", name)
		}

		if l3.VLAN < 1 || l3.VLAN > 4094 {
			add("vrfs.%s.l3vni: vlan must be between 1 and 4094", name)
		}

		if transport := cfg.Interfaces[l3.Transport]; transport == nil || transport.Type != "vxlan-bridge" {
			add("vrfs.%s.l3vni: transport must be a vxlan-bridge", name)
		}

		if other := vnis[l3.VNI]; other != "" {
			add("vrfs.%s.l3vni: vni already used by %s", name, other)
		}

		if vlans[l3.Transport] == nil {
			vlans[l3.Transport] = map[int]string{}
		}

		if other := vlans[l3.Transport][l3.VLAN]; other != "" {
			add("vrfs.%s.l3vni: vlan already used by %s", name, other)
		}

		dev := L3VNILinkName(name)
		if cfg.Interfaces[dev] != nil || cfg.VRFs[dev] != nil || generated[dev] != "" {
			add("vrfs.%s.l3vni: generated device %s conflicts with another interface", name, dev)
		}

		vnis[l3.VNI] = owner
		vlans[l3.Transport][l3.VLAN] = owner
		generated[dev] = owner
	}
}
