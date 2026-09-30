package ipcalc

import (
	"errors"
	"fmt"
	"math/big"
	"net/netip"
	"strconv"
	"strings"
)

type SubnetInfo struct {
	Family        string `json:"family"`
	Address       string `json:"address"`
	Prefix        int    `json:"prefix"`
	Netmask       string `json:"netmask"`
	Wildcard      string `json:"wildcard"`
	Network       string `json:"network"`
	HostRoute     bool   `json:"host_route,omitempty" validate:"optional"`
	HostMin       string `json:"host_min"`
	HostMax       string `json:"host_max"`
	Broadcast     string `json:"broadcast,omitempty" validate:"optional"`
	Hosts         string `json:"hosts"`
	Class         string `json:"class,omitempty" validate:"optional"`
	Scope         string `json:"scope"`
	AddressBits   string `json:"address_bits"`
	NetmaskBits   string `json:"netmask_bits"`
	WildcardBits  string `json:"wildcard_bits"`
	NetworkBits   string `json:"network_bits"`
	HostMinBits   string `json:"host_min_bits"`
	HostMaxBits   string `json:"host_max_bits"`
	BroadcastBits string `json:"broadcast_bits,omitempty" validate:"optional"`
}

type ReverseDNS struct {
	Family string `json:"family"`
	Name   string `json:"name"`
	Zone   string `json:"zone,omitempty" validate:"optional"`
}

type RangeCIDRs struct {
	Family string   `json:"family"`
	CIDRs  []string `json:"cidrs"`
}

func Subnet(input string) (*SubnetInfo, error) {
	addr, prefix, err := parseAddrOrPrefix(input)
	if err != nil {
		return nil, err
	}

	is4 := addr.Is4()
	width := addr.BitLen()
	hostBits := width - prefix

	full := allOnes(width)
	hostmask := allOnes(hostBits)
	netmask := new(big.Int).Xor(full, hostmask)
	addrN := toBig(addr)
	network := new(big.Int).And(addrN, netmask)
	last := new(big.Int).Or(network, hostmask)

	info := &SubnetInfo{
		Family:       family(is4),
		Address:      addr.String(),
		Prefix:       prefix,
		Netmask:      fromBig(netmask, is4).String(),
		Wildcard:     fromBig(hostmask, is4).String(),
		Network:      fromBig(network, is4).String() + "/" + strconv.Itoa(prefix),
		Scope:        scope(addr),
		AddressBits:  bits(addrN, width),
		NetmaskBits:  bits(netmask, width),
		WildcardBits: bits(hostmask, width),
		NetworkBits:  bits(network, width),
	}

	var hmin, hmax *big.Int
	if is4 {
		info.Class = ipv4Class(addr)
		switch prefix {
		case 32:
			hmin, hmax = network, network
			info.Hosts = "1"
			info.HostRoute = true
		case 31:
			hmin, hmax = network, last
			info.Hosts = "2"
		default:
			hmin = new(big.Int).Add(network, one)
			hmax = new(big.Int).Sub(last, one)
			info.Hosts = new(big.Int).Sub(hostmask, one).String()
			info.Broadcast = fromBig(last, is4).String()
			info.BroadcastBits = bits(last, width)
		}
	} else {
		hmin, hmax = network, last
		info.Hosts = new(big.Int).Add(hostmask, one).String()
		info.HostRoute = prefix == 128
	}

	info.HostMin = fromBig(hmin, is4).String()
	info.HostMax = fromBig(hmax, is4).String()
	info.HostMinBits = bits(hmin, width)
	info.HostMaxBits = bits(hmax, width)

	return info, nil
}

func Reverse(input string) (*ReverseDNS, error) {
	addr, prefix, err := parseAddrOrPrefix(input)
	if err != nil {
		return nil, err
	}

	is4 := addr.Is4()
	labels := reverseLabels(addr)

	suffix, labelBits := "ip6.arpa", 4
	if is4 {
		suffix, labelBits = "in-addr.arpa", 8
	}

	out := &ReverseDNS{
		Family: family(is4),
		Name:   strings.Join(labels, ".") + "." + suffix,
	}

	if strings.Contains(input, "/") && prefix%labelBits == 0 {
		drop := (addr.BitLen() - prefix) / labelBits
		zone := labels[drop:]
		if len(zone) == 0 {
			out.Zone = suffix
		} else {
			out.Zone = strings.Join(zone, ".") + "." + suffix
		}
	}

	return out, nil
}

func Range(startStr, endStr string) (*RangeCIDRs, error) {
	a, err := netip.ParseAddr(strings.TrimSpace(startStr))
	if err != nil {
		return nil, fmt.Errorf("invalid start address: %w", err)
	}

	b, err := netip.ParseAddr(strings.TrimSpace(endStr))
	if err != nil {
		return nil, fmt.Errorf("invalid end address: %w", err)
	}

	a, b = a.Unmap(), b.Unmap()
	if a.Is4() != b.Is4() {
		return nil, errors.New("start and end must be the same address family")
	}

	is4 := a.Is4()
	width := a.BitLen()
	start, end := toBig(a), toBig(b)
	if start.Cmp(end) > 0 {
		start, end = end, start
	}

	cidrs := []string{}
	for start.Cmp(end) <= 0 {
		maxAlign := trailingZeros(start, width)
		remaining := new(big.Int).Add(new(big.Int).Sub(end, start), one)
		maxCount := remaining.BitLen() - 1

		size := maxAlign
		if maxCount < size {
			size = maxCount
		}

		cidrs = append(cidrs, fromBig(start, is4).String()+"/"+strconv.Itoa(width-size))
		if size >= width {
			break
		}

		start = new(big.Int).Add(start, new(big.Int).Lsh(one, uint(size)))
	}

	return &RangeCIDRs{Family: family(is4), CIDRs: cidrs}, nil
}

func FormatBinary(raw string, prefix int, is4 bool) string {
	group := 16
	if is4 {
		group = 8
	}

	var sb strings.Builder
	for i, c := range raw {
		if i > 0 && i%group == 0 {
			sb.WriteByte('.')
		}
		if i == prefix {
			sb.WriteByte(' ')
		}
		sb.WriteRune(c)
	}

	return sb.String()
}

var one = big.NewInt(1)

func parseAddrOrPrefix(input string) (netip.Addr, int, error) {
	s := strings.TrimSpace(input)
	if strings.Contains(s, "/") {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return netip.Addr{}, 0, fmt.Errorf("invalid CIDR: %w", err)
		}

		return p.Addr().Unmap(), p.Bits(), nil
	}

	a, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}, 0, fmt.Errorf("invalid address: %w", err)
	}

	a = a.Unmap()
	return a, a.BitLen(), nil
}

func allOnes(bitLen int) *big.Int {
	if bitLen <= 0 {
		return big.NewInt(0)
	}

	return new(big.Int).Sub(new(big.Int).Lsh(one, uint(bitLen)), one)
}

func toBig(a netip.Addr) *big.Int {
	return new(big.Int).SetBytes(a.AsSlice())
}

func fromBig(n *big.Int, is4 bool) netip.Addr {
	width := 16
	if is4 {
		width = 4
	}

	buf := make([]byte, width)
	n.FillBytes(buf)
	a, _ := netip.AddrFromSlice(buf)
	return a
}

func bits(n *big.Int, width int) string {
	var sb strings.Builder
	for i := width - 1; i >= 0; i-- {
		if n.Bit(i) == 1 {
			sb.WriteByte('1')
		} else {
			sb.WriteByte('0')
		}
	}

	return sb.String()
}

func trailingZeros(n *big.Int, width int) int {
	if n.Sign() == 0 {
		return width
	}

	return int(n.TrailingZeroBits())
}

func reverseLabels(a netip.Addr) []string {
	if a.Is4() {
		b := a.As4()
		return []string{strconv.Itoa(int(b[3])), strconv.Itoa(int(b[2])), strconv.Itoa(int(b[1])), strconv.Itoa(int(b[0]))}
	}

	b := a.As16()
	labels := make([]string, 0, 32)
	for i := 15; i >= 0; i-- {
		labels = append(labels, strconv.FormatInt(int64(b[i]&0x0f), 16))
		labels = append(labels, strconv.FormatInt(int64(b[i]>>4), 16))
	}

	return labels
}

func ipv4Class(a netip.Addr) string {
	switch f := a.As4()[0]; {
	case f < 128:
		return "A"
	case f < 192:
		return "B"
	case f < 224:
		return "C"
	case f < 240:
		return "D (multicast)"
	default:
		return "E (reserved)"
	}
}

func family(is4 bool) string {
	if is4 {
		return "v4"
	}

	return "v6"
}

type scopeRange struct {
	prefix netip.Prefix
	label  string
}

var v4Scopes = []scopeRange{
	{netip.MustParsePrefix("127.0.0.0/8"), "Loopback"},
	{netip.MustParsePrefix("10.0.0.0/8"), "Private"},
	{netip.MustParsePrefix("172.16.0.0/12"), "Private"},
	{netip.MustParsePrefix("192.168.0.0/16"), "Private"},
	{netip.MustParsePrefix("100.64.0.0/10"), "CGNAT (shared)"},
	{netip.MustParsePrefix("169.254.0.0/16"), "Link-local (APIPA)"},
	{netip.MustParsePrefix("224.0.0.0/4"), "Multicast"},
	{netip.MustParsePrefix("240.0.0.0/4"), "Reserved"},
}

var v6Scopes = []scopeRange{
	{netip.MustParsePrefix("::1/128"), "Loopback"},
	{netip.MustParsePrefix("::/128"), "Unspecified"},
	{netip.MustParsePrefix("fe80::/10"), "Link-local"},
	{netip.MustParsePrefix("fc00::/7"), "Unique local"},
	{netip.MustParsePrefix("ff00::/8"), "Multicast"},
}

func scope(a netip.Addr) string {
	if a.Is4() {
		for _, s := range v4Scopes {
			if s.prefix.Contains(a) {
				return s.label
			}
		}

		return "Public"
	}

	for _, s := range v6Scopes {
		if s.prefix.Contains(a) {
			return s.label
		}
	}

	return "Global unicast"
}
