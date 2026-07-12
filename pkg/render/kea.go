package render

import (
	"encoding/json"
	"fmt"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/kea"
)

const (
	keaDHCP4Dest = "/etc/kea/kea-dhcp4.conf"
	keaDHCP6Dest = "/etc/kea/kea-dhcp6.conf"

	keaLeaseCmds = "/usr/lib/kea/hooks/libdhcp_lease_cmds.so"

	KeaLog4 = "/var/log/kea/kea-dhcp4.log"
	KeaLog6 = "/var/log/kea/kea-dhcp6.log"
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
	ID            int              `json:"id"`
	Subnet        string           `json:"subnet"`
	Interface     string           `json:"interface,omitempty"`
	Pools         []keaPool        `json:"pools,omitempty"`
	OptionData    []keaOption      `json:"option-data,omitempty"`
	ValidLifetime int              `json:"valid-lifetime,omitempty"`
	Reservations  []keaReservation `json:"reservations,omitempty"`
}

type keaLoggerOutput struct {
	Output string `json:"output"`
}

type keaLogger struct {
	Name          string            `json:"name"`
	OutputOptions []keaLoggerOutput `json:"output-options"`
	Severity      string            `json:"severity"`
}

type keaDHCP4 struct {
	InterfacesConfig keaInterfacesConfig `json:"interfaces-config"`
	ControlSocket    keaControlSocket    `json:"control-socket"`
	LeaseDatabase    keaLeaseDB          `json:"lease-database"`
	HooksLibraries   []keaHook           `json:"hooks-libraries,omitempty"`
	ValidLifetime    int                 `json:"valid-lifetime,omitempty"`
	Subnet4          []keaSubnet         `json:"subnet4"`
	Loggers          []keaLogger         `json:"loggers,omitempty"`
}

type keaDHCP6 struct {
	InterfacesConfig keaInterfacesConfig `json:"interfaces-config"`
	ControlSocket    keaControlSocket    `json:"control-socket"`
	LeaseDatabase    keaLeaseDB          `json:"lease-database"`
	HooksLibraries   []keaHook           `json:"hooks-libraries,omitempty"`
	ValidLifetime    int                 `json:"valid-lifetime,omitempty"`
	Subnet6          []keaSubnet         `json:"subnet6"`
	Loggers          []keaLogger         `json:"loggers,omitempty"`
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

	for _, iface := range cfg.Interfaces {
		if v, ok := iface.VLANs[name]; ok && v.Device != "" {
			return v.Device
		}
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

func keaSubnets(cfg *config.Config, subnets []config.KeaSubnet, v6 bool) []keaSubnet {
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

		ks.Reservations = keaReservations(s.Reservations, v6)
		out = append(out, ks)
	}

	return out
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

	if len(cfg.DHCP.Subnets4) > 0 {
		d4 := keaDHCP4{
			InterfacesConfig: keaInterfacesConfig{Interfaces: keaFamilyInterfaces(cfg, cfg.DHCP.Subnets4)},
			ControlSocket:    unixControlSocket(kea.LocalSock4),
			LeaseDatabase:    keaLeaseDB{Type: "memfile", Persist: true, Name: "/var/lib/kea/kea-leases4.csv"},
			HooksLibraries:   []keaHook{{Library: keaLeaseCmds}},
			Subnet4:          keaSubnets(cfg, cfg.DHCP.Subnets4, false),
			Loggers:          keaLoggers("kea-dhcp4", KeaLog4),
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
			Subnet6:          keaSubnets(cfg, cfg.DHCP.Subnets6, true),
			Loggers:          keaLoggers("kea-dhcp6", KeaLog6),
		}
		content, err := marshalKea("Dhcp6", d6)
		if err != nil {
			return nil, err
		}

		out = append(out, Output{Name: "kea/kea-dhcp6.conf", Dest: keaDHCP6Dest, Content: content})
	}

	return out, nil
}
