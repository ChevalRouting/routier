package render

import (
	"fmt"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
)

func bindTransferKeys(cfg *config.Config) []config.DNSTransferKey {
	s := dnsServer(cfg)
	if s == nil {
		return nil
	}

	zones := append([]config.DNSZone{}, s.Zones...)
	for _, view := range s.Views {
		zones = append(zones, view.Zones...)
	}

	seen := map[string]bool{}
	var keys []config.DNSTransferKey
	for _, zone := range zones {
		if key := zone.TransferKey; key != nil && !seen[config.NormalizeDNSName(key.Name)] {
			seen[config.NormalizeDNSName(key.Name)] = true
			keys = append(keys, *key)
		}
	}

	return keys
}

func bindPrimary(zone config.DNSZone, address string) string {
	if zone.TransferKey != nil {
		return fmt.Sprintf("%s key %q", address, zone.TransferKey.Name)
	}

	return address
}

func bindTransferPolicy(zone config.DNSZone) string {
	if len(zone.Secondaries) == 0 {
		return ""
	}

	var allowed, notify []string
	for _, address := range zone.Secondaries {
		if zone.TransferKey == nil {
			allowed = append(allowed, address+";")
		} else {
			allowed = append(allowed, fmt.Sprintf("{ !{ !%s; any; }; key %q; };", address, zone.TransferKey.Name))
		}

		notify = append(notify, bindPrimary(zone, address)+";")
	}

	return fmt.Sprintf("allow-transfer { %s };\nnotify explicit;\nalso-notify { %s };", strings.Join(allowed, " "), strings.Join(notify, " "))
}

func bindPolicyLines(policy string) []string {
	if policy == "" {
		return nil
	}

	return strings.Split(policy, "\n")
}
