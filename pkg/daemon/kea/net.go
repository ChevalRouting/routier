package kea

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

func ipToU32(ip string) uint32 {
	o := strings.Split(ip, ".")
	if len(o) != 4 {
		return 0
	}

	var n uint32
	for _, part := range o {
		v, _ := strconv.Atoi(part)
		n = (n << 8) | uint32(uint8(v))
	}

	return n
}

func prefixMask(prefixLen int) uint32 {
	if prefixLen <= 0 {
		return 0
	}

	return ^((uint32(1) << uint32(32-prefixLen)) - 1)
}

func isIPv6(ip string) bool {
	return strings.Contains(ip, ":")
}

func ipFamily(ip string) string {
	if isIPv6(ip) {
		return "dhcp6"
	}

	return "dhcp4"
}

func ipInCIDR(ip, cidr string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}

	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return false
	}

	return prefix.Contains(addr)
}

func isMAC(id string) bool {
	parts := strings.Split(id, ":")
	if len(parts) != 6 {
		return false
	}

	for _, p := range parts {
		if len(p) != 2 {
			return false
		}
	}

	return true
}

func parseCIDR(input string) (network, host string) {
	if !strings.Contains(input, "/") {
		return "", input
	}

	prefix, err := netip.ParsePrefix(input)
	if err != nil {
		return "", input
	}

	masked := prefix.Masked()
	network = masked.String()
	if prefix.Addr() == masked.Addr() {
		return network, ""
	}

	return network, prefix.Addr().String()
}

func nextAvailableIP(network string, used []string) (string, error) {
	parts := strings.Split(network, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid cidr")
	}

	if isIPv6(network) {
		return "", fmt.Errorf("automatic IPv6 IP selection not supported, please specify an IP")
	}

	prefixLen, _ := strconv.Atoi(parts[1])
	mask := prefixMask(prefixLen)
	netAddr := ipToU32(parts[0]) & mask
	broadcast := netAddr | ^mask

	usedSet := make(map[string]struct{}, len(used))
	for _, u := range used {
		usedSet[u] = struct{}{}
	}

	for start := netAddr + 1; start < broadcast; start++ {
		candidate := fmt.Sprintf("%d.%d.%d.%d", (start>>24)&0xFF, (start>>16)&0xFF, (start>>8)&0xFF, start&0xFF)
		if _, taken := usedSet[candidate]; !taken {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("no available IPs in %s", network)
}
