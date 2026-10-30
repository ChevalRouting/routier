package config

import (
	"fmt"
	"net/netip"
	"strings"
)

func validateDNSVRFs(cfg *Config, server *DNSServer, add addfunc) {
	for name, instance := range server.VRFs {
		if name == "." || name == ".." || !isValidIfname(name) || isReservedIfname(name) || cfg.VRFs[name] == nil {
			add("dns.server.vrfs.%s: VRF does not exist", name)
			continue
		}

		if instance == nil {
			add("dns.server.vrfs.%s: resolver is required", name)
			continue
		}

		if len(instance.VRFs) > 0 {
			add("dns.server.vrfs.%s: nested resolvers are not supported", name)
			continue
		}

		for _, iface := range instance.AllowInbound {
			if cfg.Interfaces[iface] == nil || cfg.Interfaces[iface].VRF != name {
				add("dns.server.vrfs.%s: inbound interface %s does not belong to the VRF", name, iface)
			}
		}

		local := *cfg
		local.DNS = &DNS{Server: instance}
		validateDNSServer(&local, dnsVRFErrors(name, add))
		for _, listen := range instance.Listen {
			addresses, err := ResolveListen(cfg, listen)
			if err != nil {
				continue
			}

			for _, address := range addresses {
				if !dnsVRFAddress(cfg, name, address) {
					add("dns.server.vrfs.%s: listen address %s does not belong to the VRF", name, address)
				}
			}
		}
	}
}

func dnsVRFErrors(name string, add addfunc) addfunc {
	return func(format string, args ...any) {
		add("%s", strings.Replace(fmt.Sprintf(format, args...), "dns.server", "dns.server.vrfs."+name, 1))
	}
}

func dnsVRFAddress(cfg *Config, name, address string) bool {
	for _, iface := range cfg.Interfaces {
		if iface.VRF != name {
			continue
		}

		for _, value := range iface.Addresses {
			prefix, err := netip.ParsePrefix(value)
			if err == nil && prefix.Addr().String() == address {
				return true
			}
		}
	}

	return false
}
