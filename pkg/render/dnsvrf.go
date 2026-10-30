package render

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/ChevalRouting/routier/pkg/config"
)

const DNSVRFRoot = "/etc/bind/vrfs"

func DNSVRFDirectory(name string) string { return filepath.Join(DNSVRFRoot, name) }

func DNSVRFConfig(cfg *config.Config, name string) *config.Config {
	local := *cfg
	local.DNS = &config.DNS{Server: cfg.DNS.Server.VRFs[name]}
	local.DHCP = nil
	return &local
}

func renderDNSVRFs(cfg *config.Config, raw string) ([]Output, error) {
	if cfg.DNS == nil || cfg.DNS.Server == nil {
		return nil, nil
	}

	names := []string{}
	for name, server := range cfg.DNS.Server.VRFs {
		if server != nil && server.Enabled {
			names = append(names, name)
		}
	}

	sort.Strings(names)
	out := []Output{}
	for _, name := range names {
		outputs, err := renderDNSVRF(cfg, name, raw)
		if err != nil {
			return nil, err
		}

		out = append(out, outputs...)
	}

	return out, nil
}

func renderDNSVRF(cfg *config.Config, name, raw string) ([]Output, error) {
	local := DNSVRFConfig(cfg, name)
	data := TemplateData{Config: local, DNSVRF: name}
	tpl, err := template.New(NamedConfName).Funcs(templateFuncs(data)).Parse(raw)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	directory := DNSVRFDirectory(name)
	paths := strings.NewReplacer("/etc/bind/zones", directory+"/zones", "/var/bind", directory+"/data", "/run/named", directory+"/run", "/var/log/named", directory+"/log")
	outputs, err := renderDNSZones(local)
	if err != nil {
		return nil, err
	}

	for index := range outputs {
		output := &outputs[index]
		output.Name = "bind/vrfs/" + name + "/" + output.Name
		output.Dest = paths.Replace(output.Dest)
	}

	return append(outputs, Output{Name: "bind/vrfs/" + name + "/named.conf", Dest: directory + "/named.conf", Content: paths.Replace(buf.String())}), nil
}

func DNSVRFZone(cfg *config.Config, output Output) (*config.Config, string, bool) {
	parts := strings.Split(output.Name, "/")
	if len(parts) != 6 || parts[0] != "bind" || parts[1] != "vrfs" || parts[3] != "bind" || parts[4] != "zones" {
		return nil, "", false
	}

	if cfg.DNS == nil || cfg.DNS.Server == nil || cfg.DNS.Server.VRFs[parts[2]] == nil {
		return nil, "", false
	}

	return DNSVRFConfig(cfg, parts[2]), ZoneOriginFromName(strings.Join(parts[3:], "/")), true
}

func DNSVRFName(name string) string { return fmt.Sprintf("dns-%s", name) }

func dnsVRFAutoRules(cfg *config.Config) []string {
	if cfg.DNS == nil || cfg.DNS.Server == nil {
		return nil
	}

	rules := []string{}
	for name, server := range cfg.DNS.Server.VRFs {
		if server == nil || !server.Enabled || server.DisableAutoRules || len(server.AllowInbound) == 0 {
			continue
		}

		port := server.Port
		if port == 0 {
			port = 53
		}

		for _, source := range server.AllowFrom {
			family := "ip"
			if strings.Contains(source, ":") {
				family = "ip6"
			}

			rules = append(rules, fmt.Sprintf("iifname %q %s saddr %s meta l4proto { tcp, udp } th dport %d accept comment %q", name, family, source, port, "routier: DNS in VRF "+name))
		}
	}

	sort.Strings(rules)
	return rules
}
