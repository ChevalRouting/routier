package render

import (
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
)

func renderAnycastGatewayTable(cfg *config.Config) string {
	var names []string
	for name, iface := range cfg.Interfaces {
		if iface.Type == "vnet" && iface.VNet != nil && iface.VNet.Gateway != nil {
			names = append(names, name)
		}
	}

	if len(names) == 0 {
		return ""
	}

	slices.Sort(names)
	var forward, output strings.Builder
	for _, name := range names {
		vn := cfg.Interfaces[name].VNet
		fabric, mac := config.VNetLinkName(name, "b"), config.VNetGatewayMAC(vn)
		comment := fmt.Sprintf("comment \"routier: %s gateway stays on this site\"", name)
		_, _ = fmt.Fprintf(&forward, "\t\tiifname %q ether saddr %s drop %s\n", fabric, mac, comment)
		_, _ = fmt.Fprintf(&forward, "\t\toifname %q ether saddr %s drop %s\n", fabric, mac, comment)
		_, _ = fmt.Fprintf(&forward, "\t\toifname %q ether daddr %s drop %s\n", fabric, mac, comment)
		if ips := gatewayIPv4s(vn.Gateway); ips != "" {
			_, _ = fmt.Fprintf(&forward, "\t\toifname %q arp daddr ip %s drop %s\n", fabric, ips, comment)
		}

		_, _ = fmt.Fprintf(&output, "\t\toifname %q ether saddr %s drop %s\n", fabric, mac, comment)
	}

	var b strings.Builder
	_, _ = b.WriteString("table bridge routier_anycast {\n")
	_, _ = b.WriteString("\tchain forward {\n\t\ttype filter hook forward priority filter; policy accept;\n")
	_, _ = b.WriteString(forward.String())
	_, _ = b.WriteString("\t}\n\tchain output {\n\t\ttype filter hook output priority filter; policy accept;\n")
	_, _ = b.WriteString(output.String())
	_, _ = b.WriteString("\t}\n}\n")
	return b.String()
}

func gatewayIPv4s(gw *config.VNetGateway) string {
	var ips []string
	for _, ip := range gw.IPs {
		if addr, err := netip.ParseAddr(ip); err == nil && addr.Is4() {
			ips = append(ips, addr.String())
		}
	}

	switch len(ips) {
	case 0:
		return ""
	case 1:
		return ips[0]
	}

	return "{ " + strings.Join(ips, ", ") + " }"
}
