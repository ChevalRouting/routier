package kea

import (
	"crypto/rand"
	"fmt"
	"net/netip"
	"strings"
)

func (c *Client) subnetFor(input string) (string, cfgSubnet, error) {
	network, host := parseCIDR(input)
	family := network
	if family == "" {
		family = host
	}

	service := ipFamily(family)

	subnets, err := c.configSubnets(service)
	if err != nil {
		return service, cfgSubnet{}, err
	}

	for _, s := range subnets {
		if s.Subnet == network || (host != "" && ipInCIDR(host, s.Subnet)) {
			return service, s, nil
		}
	}

	return service, cfgSubnet{}, fmt.Errorf("no configured subnet matches %s", input)
}

func (c *Client) FreeIP(input string, exclusions []string) (string, error) {
	_, ip, err := c.freeReservationIP(input, exclusions)
	return ip, err
}

func (c *Client) SubnetCIDR(input string) (string, error) {
	_, sub, err := c.subnetFor(input)
	if err != nil {
		return "", err
	}

	return sub.Subnet, nil
}

type addrRange struct{ lo, hi netip.Addr }

func (c *Client) freeReservationIP(input string, exclusions []string) (subnet, ip string, err error) {
	_, sub, err := c.subnetFor(input)
	if err != nil {
		return "", "", err
	}

	prefix, perr := netip.ParsePrefix(sub.Subnet)
	if perr != nil {
		return "", "", fmt.Errorf("subnet %q is not a valid cidr: %w", sub.Subnet, perr)
	}

	prefix = prefix.Masked()

	reserved := make(map[netip.Addr]bool)
	for _, r := range sub.reservedIPs() {
		if a, e := netip.ParseAddr(r); e == nil {
			reserved[a] = true
		}
	}

	pools := parsePools(sub.Pools)

	exRanges, exAddrs := parseExclusions(exclusions)
	pools = append(pools, exRanges...)
	for _, a := range exAddrs {
		reserved[a] = true
	}

	free := func(a netip.Addr) bool { return freeReservationIPCallback(prefix, reserved, pools, a) }

	if prefix.Addr().Is6() {
		if a, ok := randomFreeAddr(prefix, free); ok {
			return sub.Subnet, a.String(), nil
		}

		return "", "", fmt.Errorf("could not find a free address in %s", sub.Subnet)
	}

	const maxScan = 1 << 20
	candidate := prefix.Addr().Next()
	for i := 0; i < maxScan; i++ {
		if !candidate.IsValid() || !prefix.Contains(candidate) || !prefix.Contains(candidate.Next()) {
			break
		}

		if rng, ok := poolFor(candidate, pools); ok {
			candidate = rng.hi.Next()
			continue
		}

		if !reserved[candidate] {
			return sub.Subnet, candidate.String(), nil
		}

		candidate = candidate.Next()
	}

	return "", "", fmt.Errorf("no address available outside the pools of %s", sub.Subnet)
}

func randomFreeAddr(prefix netip.Prefix, free func(netip.Addr) bool) (netip.Addr, bool) {
	net := prefix.Addr().As16()
	bits := prefix.Bits()

	const attempts = 4096
	for i := 0; i < attempts; i++ {
		var rnd [16]byte
		if _, err := rand.Read(rnd[:]); err != nil {
			return netip.Addr{}, false
		}

		var out [16]byte
		for b := 0; b < 16; b++ {
			mask := hostByteMask(bits, b*8)
			out[b] = (net[b] & mask) | (rnd[b] & ^mask)
		}

		candidate := netip.AddrFrom16(out)
		if prefix.Contains(candidate) && free(candidate) {
			return candidate, true
		}
	}

	return netip.Addr{}, false
}

func hostByteMask(prefixBits, bitPos int) byte {
	switch {
	case prefixBits >= bitPos+8:
		return 0xff
	case prefixBits <= bitPos:
		return 0x00
	default:
		return byte(0xff << uint(8-(prefixBits-bitPos)))
	}
}

func parseExclusions(list []string) (ranges []addrRange, addrs []netip.Addr) {
	for _, e := range list {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}

		if strings.Contains(e, "-") {
			parts := strings.SplitN(e, "-", 2)
			lo, err1 := netip.ParseAddr(strings.TrimSpace(parts[0]))
			hi, err2 := netip.ParseAddr(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil && lo.Compare(hi) <= 0 {
				ranges = append(ranges, addrRange{lo, hi})
			}

			continue
		}

		if a, err := netip.ParseAddr(e); err == nil {
			addrs = append(addrs, a)
		}
	}

	return ranges, addrs
}

func parsePools(pools []cfgPool) []addrRange {
	var out []addrRange
	for _, p := range pools {
		parts := strings.SplitN(p.Pool, "-", 2)
		if len(parts) != 2 {
			continue
		}

		lo, err1 := netip.ParseAddr(strings.TrimSpace(parts[0]))
		hi, err2 := netip.ParseAddr(strings.TrimSpace(parts[1]))
		if err1 == nil && err2 == nil && lo.Compare(hi) <= 0 {
			out = append(out, addrRange{lo, hi})
		}
	}

	return out
}

func poolFor(a netip.Addr, pools []addrRange) (addrRange, bool) {
	for _, r := range pools {
		if a.Compare(r.lo) >= 0 && a.Compare(r.hi) <= 0 {
			return r, true
		}
	}

	return addrRange{}, false
}

func freeReservationIPCallback(prefix netip.Prefix, reserved map[netip.Addr]bool, pools []addrRange, a netip.Addr) bool {
	if !a.IsValid() || a == prefix.Addr() {
		return false
	}

	if _, inPool := poolFor(a, pools); inPool {
		return false
	}

	return !reserved[a]
}
