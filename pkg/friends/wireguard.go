package friends

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"net"
	"sort"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
	"golang.org/x/crypto/curve25519"
)

func isStaticAddr(addr string) bool {
	switch addr {
	case "dhcp", "dhcp4", "dhcp6", "slaac":
		return false
	}

	return true
}

func hostIP(addr string) string {
	if ip, _, err := net.ParseCIDR(addr); err == nil {
		return ip.String()
	}

	return addr
}

func InterfaceAddresses(cfg *config.Config) []types.FriendInterface {
	vrrpVIPs := map[string][]string{}
	if cfg.HA != nil {
		for _, v := range cfg.HA.VRRP {
			vrrpVIPs[v.Interface] = append(vrrpVIPs[v.Interface], v.VIPs...)
		}
	}

	out := make([]types.FriendInterface, 0, len(cfg.Interfaces))
	for name, iface := range cfg.Interfaces {
		var addrs []string
		for _, a := range iface.Addresses {
			if isStaticAddr(a) {
				addrs = append(addrs, hostIP(a))
			}
		}

		for _, vip := range vrrpVIPs[name] {
			if isStaticAddr(vip) {
				addrs = append(addrs, hostIP(vip))
			}
		}

		out = append(out, types.FriendInterface{Name: name, Addresses: addrs})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

	return out
}

func ResolveEndpointAddr(ifaces []types.FriendInterface, iface, addr string) (string, error) {
	var match *types.FriendInterface
	for i := range ifaces {
		if ifaces[i].Name == iface {
			match = &ifaces[i]
		}
	}

	if match == nil {
		return "", fmt.Errorf("interface %q not found", iface)
	}

	if addr != "" {
		for _, a := range match.Addresses {
			if a == addr || hostIP(addr) == a {
				return hostIP(addr), nil
			}
		}

		return "", fmt.Errorf("address %q is not on interface %q", addr, iface)
	}

	if len(match.Addresses) == 0 {
		return "", fmt.Errorf("interface %q has no static address", iface)
	}

	if len(match.Addresses) > 1 {
		return "", fmt.Errorf("interface %q has multiple addresses, specify one", iface)
	}

	return match.Addresses[0], nil
}

func GenerateWGKeyPair() (privB64, pubB64 string, err error) {
	var private [32]byte
	if _, err = rand.Read(private[:]); err != nil {
		return
	}

	private[0] &= 248
	private[31] = (private[31] & 127) | 64

	pub, err := curve25519.X25519(private[:], curve25519.Basepoint)
	if err != nil {
		return
	}

	privB64 = base64.StdEncoding.EncodeToString(private[:])
	pubB64 = base64.StdEncoding.EncodeToString(pub)

	return
}

func GeneratePSK() (string, error) {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(key[:]), nil
}

func randomPort() (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(55001))
	if err != nil {
		return 0, err
	}

	return 10000 + int(n.Int64()), nil
}

func wgInterfaceName(friendName string) string {
	clean := strings.Map(wgInterfaceNameHandler, friendName)

	name := "wg-fr-" + clean
	if len(name) > 15 {
		name = name[:15]
	}

	return name
}

func nextIP(ip net.IP) net.IP {
	out := make(net.IP, len(ip))
	copy(out, ip)

	for i := len(out) - 1; i >= 0; i-- {
		out[i]++
		if out[i] != 0 {
			break
		}
	}

	return out
}

type WGDeriveParams struct {
	FriendName    string
	LocalHostname string
	Subnet        string
	LocalAddr     string
	LocalPort     int
	FriendAddr    string
	FriendPort    int
}

type WGDeriveResult struct {
	InterfaceName string
	Local         *config.Wireguard
	Counterpart   *config.Wireguard
}

func DeriveWireguard(p WGDeriveParams) (*WGDeriveResult, error) {
	if p.FriendName == "" {
		return nil, fmt.Errorf("friend name is required")
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

	localPort, friendPort := p.LocalPort, p.FriendPort
	if localPort == 0 || friendPort == 0 {
		shared, err := randomPort()
		if err != nil {
			return nil, err
		}

		if localPort == 0 {
			localPort = shared
		}

		if friendPort == 0 {
			friendPort = shared
		}
	}

	localPriv, localPub, err := GenerateWGKeyPair()
	if err != nil {
		return nil, err
	}

	friendPriv, friendPub, err := GenerateWGKeyPair()
	if err != nil {
		return nil, err
	}

	psk, err := GeneratePSK()
	if err != nil {
		return nil, err
	}

	name := wgInterfaceName(p.FriendName)

	local := &config.Wireguard{
		Friend:     p.FriendName,
		PrivateKey: localPriv,
		ListenPort: localPort,
		Table:      "off",
		Addresses:  []string{fmt.Sprintf("%s/%d", localTunnel, prefix)},
		Peers: []config.WGPeer{{
			Name:         p.FriendName,
			PublicKey:    friendPub,
			PresharedKey: psk,
			Endpoint:     net.JoinHostPort(p.FriendAddr, fmt.Sprint(friendPort)),
			AllowedIPs:   []string{friendTunnel.String() + "/32"},
			Keepalive:    25,
		}},
	}

	counterpart := &config.Wireguard{
		Friend:     p.LocalHostname,
		PrivateKey: friendPriv,
		ListenPort: friendPort,
		Table:      "off",
		Addresses:  []string{fmt.Sprintf("%s/%d", friendTunnel, prefix)},
		Peers: []config.WGPeer{{
			Name:         p.LocalHostname,
			PublicKey:    localPub,
			PresharedKey: psk,
			Endpoint:     net.JoinHostPort(p.LocalAddr, fmt.Sprint(localPort)),
			AllowedIPs:   []string{localTunnel.String() + "/32"},
			Keepalive:    25,
		}},
	}

	return &WGDeriveResult{InterfaceName: name, Local: local, Counterpart: counterpart}, nil
}

func wgInterfaceNameHandler(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		return r
	case r >= 'A' && r <= 'Z':
		return r + ('a' - 'A')
	default:
		return '-'
	}
}
