package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/kea"
)

const (
	keaDHCP4Dest = "/etc/kea/kea-dhcp4.conf"
	keaDHCP6Dest = "/etc/kea/kea-dhcp6.conf"

	keaLeaseCmds = "/usr/lib/kea/hooks/libdhcp_lease_cmds.so"

	KeaLog4 = "/var/log/kea/kea-dhcp4.log"
	KeaLog6 = "/var/log/kea/kea-dhcp6.log"

	keaD2ServerIP   = "127.0.0.1"
	keaD2ServerPort = 53001
)

func LocalKeaEnabled(cfg *config.Config) bool {
	if cfg == nil || cfg.DHCP == nil || !cfg.DHCP.Enabled {
		return false
	}

	if ca := cfg.DHCP.ControlAgent; ca != nil && ca.URL != "" {
		return false
	}

	return len(cfg.DHCP.Subnets4) > 0 || len(cfg.DHCP.Subnets6) > 0
}

func LocalKeaDDNSEnabled(cfg *config.Config) bool {
	if !LocalKeaEnabled(cfg) {
		return false
	}

	d := cfg.DHCP.DDNS

	return d != nil && d.Enabled && d.Domain != ""
}

type keaControlSocket struct {
	SocketType string `json:"socket-type"`
	SocketName string `json:"socket-name"`
}

func unixControlSocket(path string) keaControlSocket {
	return keaControlSocket{SocketType: "unix", SocketName: path}
}

type keaInterfacesConfig struct {
	Interfaces []string `json:"interfaces"`
}

type keaLeaseDB struct {
	Type    string `json:"type"`
	Persist bool   `json:"persist"`
	Name    string `json:"name"`
}

type keaHook struct {
	Library string `json:"library"`
}

type keaOption struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type keaPool struct {
	Pool string `json:"pool"`
}

type keaReservation struct {
	Hostname    string   `json:"hostname,omitempty"`
	HWAddress   string   `json:"hw-address,omitempty"`
	DUID        string   `json:"duid,omitempty"`
	IPAddress   string   `json:"ip-address,omitempty"`
	IPAddresses []string `json:"ip-addresses,omitempty"`
}

type keaSubnet struct {
	ID                   int              `json:"id"`
	Subnet               string           `json:"subnet"`
	Interface            string           `json:"interface,omitempty"`
	Pools                []keaPool        `json:"pools,omitempty"`
	OptionData           []keaOption      `json:"option-data,omitempty"`
	ValidLifetime        int              `json:"valid-lifetime,omitempty"`
	Reservations         []keaReservation `json:"reservations,omitempty"`
	DDNSSendUpdates      *bool            `json:"ddns-send-updates,omitempty"`
	DDNSQualifyingSuffix string           `json:"ddns-qualifying-suffix,omitempty"`
}

type keaDHCPDDNS struct {
	EnableUpdates bool   `json:"enable-updates"`
	ServerIP      string `json:"server-ip,omitempty"`
	ServerPort    int    `json:"server-port,omitempty"`
	NCRProtocol   string `json:"ncr-protocol,omitempty"`
	NCRFormat     string `json:"ncr-format,omitempty"`
}

type keaLoggerOutput struct {
	Output string `json:"output"`
}

type keaLogger struct {
	Name          string            `json:"name"`
	OutputOptions []keaLoggerOutput `json:"output-options"`
	Severity      string            `json:"severity"`
}

type keaDDNSBehavior struct {
	SendUpdates             *bool  `json:"ddns-send-updates,omitempty"`
	QualifyingSuffix        string `json:"ddns-qualifying-suffix,omitempty"`
	OverrideClientUpdate    *bool  `json:"ddns-override-client-update,omitempty"`
	OverrideNoUpdate        *bool  `json:"ddns-override-no-update,omitempty"`
	ReplaceClientName       string `json:"ddns-replace-client-name,omitempty"`
	GeneratedPrefix         string `json:"ddns-generated-prefix,omitempty"`
	HostnameCharSet         string `json:"hostname-char-set,omitempty"`
	HostnameCharReplacement string `json:"hostname-char-replacement,omitempty"`
}

type keaDHCP4 struct {
	InterfacesConfig keaInterfacesConfig `json:"interfaces-config"`
	ControlSocket    keaControlSocket    `json:"control-socket"`
	LeaseDatabase    keaLeaseDB          `json:"lease-database"`
	HooksLibraries   []keaHook           `json:"hooks-libraries,omitempty"`
	ValidLifetime    int                 `json:"valid-lifetime,omitempty"`
	DHCPDDNS         *keaDHCPDDNS        `json:"dhcp-ddns,omitempty"`
	keaDDNSBehavior
	Subnet4 []keaSubnet `json:"subnet4"`
	Loggers []keaLogger `json:"loggers,omitempty"`
}

type keaDHCP6 struct {
	InterfacesConfig keaInterfacesConfig `json:"interfaces-config"`
	ControlSocket    keaControlSocket    `json:"control-socket"`
	LeaseDatabase    keaLeaseDB          `json:"lease-database"`
	HooksLibraries   []keaHook           `json:"hooks-libraries,omitempty"`
	ValidLifetime    int                 `json:"valid-lifetime,omitempty"`
	DHCPDDNS         *keaDHCPDDNS        `json:"dhcp-ddns,omitempty"`
	keaDDNSBehavior
	Subnet6 []keaSubnet `json:"subnet6"`
	Loggers []keaLogger `json:"loggers,omitempty"`
}

func keaLoggers(name, path string) []keaLogger {
	return []keaLogger{{
		Name:          name,
		OutputOptions: []keaLoggerOutput{{Output: path}},
		Severity:      "INFO",
	}}
}

func keaResolveIface(cfg *config.Config, name string) string {
	if name == "*" {
		return name
	}

	if i, ok := cfg.Interfaces[name]; ok && i.Device != "" {
		return i.Device
	}

	return name
}

func keaFamilyInterfaces(cfg *config.Config, subnets []config.KeaSubnet) []string {
	out := make([]string, 0, len(subnets))
	seen := make(map[string]bool)
	for _, s := range subnets {
		if s.Interface == "" {
			return []string{"*"}
		}

		dev := keaResolveIface(cfg, s.Interface)
		if !seen[dev] {
			seen[dev] = true
			out = append(out, dev)
		}
	}

	if len(out) == 0 {
		return []string{"*"}
	}

	return out
}

func keaReservations(rs []config.KeaReservation, v6 bool) []keaReservation {
	out := make([]keaReservation, 0, len(rs))
	for _, r := range rs {
		kr := keaReservation{Hostname: r.Hostname, HWAddress: r.HWAddress, DUID: r.DUID}
		if r.IPAddress != "" {
			if v6 {
				kr.IPAddresses = []string{r.IPAddress}
			} else {
				kr.IPAddress = r.IPAddress
			}
		}

		out = append(out, kr)
	}

	return out
}

func keaSubnets(cfg *config.Config, subnets []config.KeaSubnet, v6 bool, ddns *config.DHCPDDNS) []keaSubnet {
	out := make([]keaSubnet, 0, len(subnets))
	for i, s := range subnets {
		ks := keaSubnet{ID: i + 1, Subnet: s.Subnet, ValidLifetime: s.ValidLifetime}
		if s.Interface != "" {
			ks.Interface = keaResolveIface(cfg, s.Interface)
		}

		for _, p := range s.Pools {
			ks.Pools = append(ks.Pools, keaPool{Pool: p})
		}

		if s.Gateway != "" && !v6 {
			ks.OptionData = append(ks.OptionData, keaOption{Name: "routers", Data: s.Gateway})
		}

		if len(s.DNS) > 0 {
			name := "domain-name-servers"
			if v6 {
				name = "dns-servers"
			}

			ks.OptionData = append(ks.OptionData, keaOption{Name: name, Data: joinComma(s.DNS)})
		}

		if ddns != nil {
			ks.DDNSSendUpdates = s.DDNS
			if s.DDNSDomain != "" {
				ks.DDNSQualifyingSuffix = strings.TrimSuffix(s.DDNSDomain, ".")
			}
		}

		ks.Reservations = keaReservations(s.Reservations, v6)
		out = append(out, ks)
	}

	return out
}

func keaDDNSBlock() *keaDHCPDDNS {
	return &keaDHCPDDNS{
		EnableUpdates: true,
		ServerIP:      keaD2ServerIP,
		ServerPort:    keaD2ServerPort,
		NCRProtocol:   "UDP",
		NCRFormat:     "JSON",
	}
}

func keaTrue() *bool {
	v := true

	return &v
}

func keaDDNSBehaviorFor(d *config.DHCPDDNS) keaDDNSBehavior {
	b := keaDDNSBehavior{
		SendUpdates:       keaTrue(),
		QualifyingSuffix:  strings.TrimSuffix(d.Domain, "."),
		ReplaceClientName: d.ReplaceClientName,
		GeneratedPrefix:   d.GeneratedPrefix,
		HostnameCharSet:   d.HostnameCharSet,
	}
	if d.OverrideClientUpdate {
		b.OverrideClientUpdate = keaTrue()
	}

	if d.OverrideNoUpdate {
		b.OverrideNoUpdate = keaTrue()
	}

	return b
}

func joinComma(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ", "
		}

		out += v
	}

	return out
}

func marshalKea(root string, body any) (string, error) {
	wrapped := map[string]any{root: body}
	b, err := json.MarshalIndent(wrapped, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal %s: %w", root, err)
	}

	return string(b) + "\n", nil
}

func renderKea(cfg *config.Config) ([]Output, error) {
	if !LocalKeaEnabled(cfg) {
		return nil, nil
	}

	var out []Output

	var ddns *config.DHCPDDNS
	if LocalKeaDDNSEnabled(cfg) {
		ddns = cfg.DHCP.DDNS
	}

	if len(cfg.DHCP.Subnets4) > 0 {
		d4 := keaDHCP4{
			InterfacesConfig: keaInterfacesConfig{Interfaces: keaFamilyInterfaces(cfg, cfg.DHCP.Subnets4)},
			ControlSocket:    unixControlSocket(kea.LocalSock4),
			LeaseDatabase:    keaLeaseDB{Type: "memfile", Persist: true, Name: "/var/lib/kea/kea-leases4.csv"},
			HooksLibraries:   []keaHook{{Library: keaLeaseCmds}},
			Subnet4:          keaSubnets(cfg, cfg.DHCP.Subnets4, false, ddns),
			Loggers:          keaLoggers("kea-dhcp4", KeaLog4),
		}
		if ddns != nil {
			d4.DHCPDDNS = keaDDNSBlock()
			d4.keaDDNSBehavior = keaDDNSBehaviorFor(ddns)
		}

		content, err := marshalKea("Dhcp4", d4)
		if err != nil {
			return nil, err
		}

		out = append(out, Output{Name: "kea/kea-dhcp4.conf", Dest: keaDHCP4Dest, Content: content})
	}

	if len(cfg.DHCP.Subnets6) > 0 {
		d6 := keaDHCP6{
			InterfacesConfig: keaInterfacesConfig{Interfaces: keaFamilyInterfaces(cfg, cfg.DHCP.Subnets6)},
			ControlSocket:    unixControlSocket(kea.LocalSock6),
			LeaseDatabase:    keaLeaseDB{Type: "memfile", Persist: true, Name: "/var/lib/kea/kea-leases6.csv"},
			HooksLibraries:   []keaHook{{Library: keaLeaseCmds}},
			Subnet6:          keaSubnets(cfg, cfg.DHCP.Subnets6, true, ddns),
			Loggers:          keaLoggers("kea-dhcp6", KeaLog6),
		}
		if ddns != nil {
			d6.DHCPDDNS = keaDDNSBlock()
			d6.keaDDNSBehavior = keaDDNSBehaviorFor(ddns)
		}

		content, err := marshalKea("Dhcp6", d6)
		if err != nil {
			return nil, err
		}

		out = append(out, Output{Name: "kea/kea-dhcp6.conf", Dest: keaDHCP6Dest, Content: content})
	}

	return out, nil
}
