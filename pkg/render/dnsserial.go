package render

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
)

var zoneSOASerial = regexp.MustCompile(`(IN SOA[^\n]*\(\s*)([0-9]+)`)

func AdvanceZoneSerial(current, next string) string {
	oldMatch := zoneSOASerial.FindStringSubmatchIndex(current)
	newMatch := zoneSOASerial.FindStringSubmatchIndex(next)
	if oldMatch == nil || newMatch == nil {
		return next
	}

	oldSerial := current[oldMatch[4]:oldMatch[5]]
	normalized := next[:newMatch[4]] + oldSerial + next[newMatch[5]:]
	if normalized == current {
		return current
	}

	serial, err := strconv.ParseUint(oldSerial, 10, 32)
	if err != nil {
		return next
	}

	return next[:newMatch[4]] + fmt.Sprint(uint32(serial)+1) + next[newMatch[5]:]
}

func AutomaticZoneSerial(cfg *config.Config, name string) bool {
	server := dnsServer(cfg)
	if server == nil {
		return false
	}

	zones := append([]config.DNSZone{}, server.Zones...)
	for _, view := range server.Views {
		zones = append(zones, view.Zones...)
	}

	for _, zone := range zones {
		if strings.EqualFold(config.NormalizeDNSName(zone.Name), config.NormalizeDNSName(name)) {
			return len(zone.Primaries) == 0 && (zone.SOA == nil || zone.SOA.Serial == 0) && !ddnsManagedZone(cfg, name)
		}
	}

	return false
}
