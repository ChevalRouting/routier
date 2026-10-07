package friends

import (
	"fmt"
	"net"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
)

func tunInterfaceName(friendName string) string {
	clean := strings.Map(tunInterfaceNameHandler, friendName)

	name := "tun-fr-" + clean
	if len(name) > 15 {
		name = name[:15]
	}

	return name
}

type TunnelDeriveParams struct {
	FriendName    string
	LocalHostname string
	Mode          string
	Subnet        string
	LocalAddr     string
	FriendAddr    string
}

type TunnelDeriveResult struct {
	InterfaceName string
	Local         *config.Tunnel
	Counterpart   *config.Tunnel
}

func DeriveTunnel(p TunnelDeriveParams) (*TunnelDeriveResult, error) {
	if p.FriendName == "" {
		return nil, fmt.Errorf("friend name is required")
	}

	if p.Mode == "" {
		return nil, fmt.Errorf("tunnel mode is required")
	}

	if p.LocalAddr == "" || p.FriendAddr == "" {
		return nil, fmt.Errorf("both endpoint addresses are required")
	}

	base, ipnet, err := net.ParseCIDR(p.Subnet)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet %q: %w", p.Subnet, err)
	}

	prefix, _ := ipnet.Mask.Size()
	localTunnel := base.Mask(ipnet.Mask)
	friendTunnel := nextIP(localTunnel)
	if !ipnet.Contains(friendTunnel) {
		return nil, fmt.Errorf("subnet %q is too small for two endpoints", p.Subnet)
	}

	name := tunInterfaceName(p.FriendName)

	local := &config.Tunnel{
		Mode:      p.Mode,
		Local:     p.LocalAddr,
		Remote:    p.FriendAddr,
		Addresses: []string{fmt.Sprintf("%s/%d", localTunnel, prefix)},
		Friend:    p.FriendName,
	}

	counterpart := &config.Tunnel{
		Mode:      p.Mode,
		Local:     p.FriendAddr,
		Remote:    p.LocalAddr,
		Addresses: []string{fmt.Sprintf("%s/%d", friendTunnel, prefix)},
		Friend:    p.LocalHostname,
	}

	return &TunnelDeriveResult{InterfaceName: name, Local: local, Counterpart: counterpart}, nil
}

func tunInterfaceNameHandler(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		return r
	case r >= 'A' && r <= 'Z':
		return r + ('a' - 'A')
	default:
		return '-'
	}
}
