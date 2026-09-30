package dhcp

import "github.com/ChevalRouting/routier/pkg/render"

func leaseLogFiles(source string) ([]string, bool) {
	switch source {
	case "":
		return []string{render.KeaLog4, render.KeaLog6}, true
	case "kea-dhcp4":
		return []string{render.KeaLog4}, true
	case "kea-dhcp6":
		return []string{render.KeaLog6}, true
	case "kea-dhcp-ddns":
		return []string{render.KeaLogDDNS}, true
	default:
		return nil, false
	}
}
