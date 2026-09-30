package render

import (
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/daemon/kea"
)

const (
	keaDDNSDest = "/etc/kea/kea-dhcp-ddns.conf"
	KeaLogDDNS  = "/var/log/kea/kea-dhcp-ddns.log"

	DDNSKeyName = "routier-ddns"
)

type keaTSIGKey struct {
	Name      string `json:"name"`
	Algorithm string `json:"algorithm"`
	Secret    string `json:"secret"`
}

type keaDNSServer struct {
	IPAddress string `json:"ip-address"`
	Port      int    `json:"port,omitempty"`
}

type keaDDNSDomain struct {
	Name       string         `json:"name"`
	KeyName    string         `json:"key-name,omitempty"`
	DNSServers []keaDNSServer `json:"dns-servers"`
}

type keaDDNSDomainList struct {
	DDNSDomains []keaDDNSDomain `json:"ddns-domains"`
}

type keaDHCPDdns struct {
	IPAddress     string            `json:"ip-address"`
	Port          int               `json:"port"`
	ControlSocket keaControlSocket  `json:"control-socket"`
	TSIGKeys      []keaTSIGKey      `json:"tsig-keys,omitempty"`
	ForwardDDNS   keaDDNSDomainList `json:"forward-ddns"`
	ReverseDDNS   keaDDNSDomainList `json:"reverse-ddns"`
	Loggers       []keaLogger       `json:"loggers,omitempty"`
}

func keaTSIGAlgorithm(algorithm string) string {
	return strings.ToUpper(algorithm)
}

func keaDDNSDomains(names []string, server keaDNSServer) []keaDDNSDomain {
	out := make([]keaDDNSDomain, 0, len(names))
	for _, name := range names {
		out = append(out, keaDDNSDomain{
			Name:       config.NormalizeDNSName(name),
			KeyName:    DDNSKeyName,
			DNSServers: []keaDNSServer{server},
		})
	}

	return out
}

func renderKeaDDNS(cfg *config.Config) ([]Output, error) {
	if !ddnsActive(cfg) {
		return nil, nil
	}

	d := cfg.DHCP.DDNS
	server := keaDNSServer{IPAddress: DNSQueryAddress(cfg), Port: dnsServerPort(cfg)}

	d2 := keaDHCPDdns{
		IPAddress:     keaD2ServerIP,
		Port:          keaD2ServerPort,
		ControlSocket: unixControlSocket(kea.LocalSockD2),
		TSIGKeys: []keaTSIGKey{{
			Name:      DDNSKeyName,
			Algorithm: keaTSIGAlgorithm(d.Algorithm),
			Secret:    d.Key,
		}},
		ForwardDDNS: keaDDNSDomainList{DDNSDomains: keaDDNSDomains(ddnsForwardZones(cfg), server)},
		ReverseDDNS: keaDDNSDomainList{DDNSDomains: keaDDNSDomains(ddnsReverseZoneNames(cfg), server)},
		Loggers:     keaLoggers("kea-dhcp-ddns", KeaLogDDNS),
	}

	content, err := marshalKea("DhcpDdns", d2)
	if err != nil {
		return nil, err
	}

	return []Output{{Name: "kea/kea-dhcp-ddns.conf", Dest: keaDDNSDest, Content: content}}, nil
}
