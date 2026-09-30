package render

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/rs/zerolog/log"
)

type TemplateData struct {
	*config.Config
	SelfExe string
	Friends map[string]friends.Vars
}

type Option func(*TemplateData)

func WithFriends(f map[string]friends.Vars) Option {
	return func(d *TemplateData) { d.Friends = f }
}

func selfExe() string {
	exe, err := os.Executable()
	if err != nil {
		return "routier"
	}

	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}

	return exe
}

//go:embed defaults
var embedded embed.FS

const UserDir = "/etc/routier/templates"

func Embedded() embed.FS {
	return embedded
}

func ReadEmbedded(p string) ([]byte, error) {
	return embedded.ReadFile(p)
}

type Output struct {
	Name    string
	Dest    string
	Content string
}

func All(cfg *config.Config, opts ...Option) ([]Output, error) {
	tpls, err := collectTemplates()
	if err != nil {
		return nil, err
	}

	data := TemplateData{Config: cfg, SelfExe: selfExe()}
	for _, o := range opts {
		o(&data)
	}

	funcs := templateFuncs(data)
	var out []Output
	for name, raw := range tpls {
		dest := destFor(name)
		if dest == "" || !shouldRender(cfg, name) {
			continue
		}

		t, err := template.New(name).Funcs(funcs).Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("template %s: %w", name, err)
		}

		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("template %s: %w", name, err)
		}

		out = append(out, Output{Name: name, Dest: dest, Content: buf.String()})
	}

	wgOut, err := renderWireguard(cfg)
	if err != nil {
		return nil, err
	}

	out = append(out, wgOut...)
	svcOut, err := renderServiceConfigs(data, funcs)
	if err != nil {
		return nil, err
	}

	out = append(out, svcOut...)
	keaOut, err := renderKea(cfg)
	if err != nil {
		return nil, err
	}

	out = append(out, keaOut...)
	keaDDNSOut, err := renderKeaDDNS(cfg)
	if err != nil {
		return nil, err
	}

	out = append(out, keaDDNSOut...)
	dnsOut, err := renderDNSZones(cfg)
	if err != nil {
		return nil, err
	}

	out = append(out, dnsOut...)
	dhcpcdOut, err := renderDhcpcd(cfg)
	if err != nil {
		return nil, err
	}

	out = append(out, dhcpcdOut...)
	lldpOut, err := renderLLDP(cfg)
	if err != nil {
		return nil, err
	}

	out = append(out, lldpOut...)
	return out, nil
}

func collectTemplates() (map[string]string, error) {
	tpls := make(map[string]string)
	err := fs.WalkDir(embedded, "defaults", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		data, err := embedded.ReadFile(path)
		if err != nil {
			return err
		}

		tpls[strings.TrimPrefix(path, "defaults/")] = string(data)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if info, e := os.Stat(UserDir); e == nil && info.IsDir() {
		err = filepath.WalkDir(UserDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}

			fileData, err := os.ReadFile(path)
			if err != nil {
				log.Warn().Err(err).Str("path", path).Msg("user template: skipping unreadable file")
				return nil
			}

			name, _ := filepath.Rel(UserDir, path)
			tpls[name] = string(fileData)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return tpls, nil
}

func shouldRender(cfg *config.Config, name string) bool {
	switch {
	case strings.HasPrefix(name, "wireguard/"):
		return false
	case strings.HasPrefix(name, "interfaces/"):
		return false
	case strings.HasPrefix(name, "tunnels/"):
		return false
	case strings.HasPrefix(name, "nftables/"):
		return len(cfg.Interfaces) > 0 || cfg.Nftables != nil
	case strings.HasPrefix(name, "frr/"):
		return cfg.Routing != nil
	case strings.HasPrefix(name, "radvd/"):
		return cfg.Routing != nil && cfg.Routing.RADVD != nil && len(cfg.Routing.RADVD.Interfaces) > 0
	case strings.HasPrefix(name, "sysctl/"):
		return len(cfg.Sysctl) > 0
	case strings.HasPrefix(name, "keepalived/"):
		return cfg.HA != nil && len(cfg.HA.VRRP) > 0
	case strings.HasPrefix(name, "conntrackd/"):
		return cfg.HA != nil && cfg.HA.Conntrackd != nil
	case strings.HasPrefix(name, "ssh/"):
		return cfg.SSH != nil
	case strings.HasPrefix(name, "modules-load.d/"):
		return len(cfg.BootModules) > 0
	case strings.HasPrefix(name, "gai/"):
		return cfg.GAI != nil
	case strings.HasPrefix(name, "bind/"):
		return LocalDNSServerEnabled(cfg)
	}

	return true
}

func NameForDest(dest string) string {
	m := map[string]string{
		"/etc/dhcpcd.conf":                    "dhcpcd/dhcpcd.conf",
		"/etc/nftables.d/routier.nft":         "nftables/routier.nft",
		"/etc/frr/frr.conf":                   "frr/frr.conf",
		"/etc/frr/daemons":                    "frr/daemons",
		"/etc/sysctl.d/99-routier.conf":       "sysctl/routier.conf",
		"/etc/keepalived/keepalived.conf":     "keepalived/keepalived.conf",
		"/etc/radvd.conf":                     "radvd/radvd.conf",
		"/etc/conntrackd/conntrackd.conf":     "conntrackd/conntrackd.conf",
		"/etc/ssh/sshd_config.d/routier.conf": "ssh/sshd_config",
		"/etc/modules-load.d/routier.conf":    "modules-load.d/routier.conf",
		"/etc/gai.conf":                       "gai/gai.conf",
		lldpdConfDest:                         lldpdConfName,
		lldpdOptsDest:                         lldpdOptsName,
		NamedConfDest:                         NamedConfName,
	}
	if n, ok := m[dest]; ok {
		return n
	}

	if strings.HasPrefix(dest, "/etc/wireguard/") {
		return "wireguard/" + filepath.Base(dest)
	}

	if strings.HasPrefix(dest, namedZoneDest) {
		return namedZonePfx + filepath.Base(dest)
	}

	return ""
}

func destFor(name string) string {
	m := map[string]string{
		"nftables/routier.nft":        "/etc/nftables.d/routier.nft",
		"frr/frr.conf":                "/etc/frr/frr.conf",
		"frr/daemons":                 "/etc/frr/daemons",
		"sysctl/routier.conf":         "/etc/sysctl.d/99-routier.conf",
		"keepalived/keepalived.conf":  "/etc/keepalived/keepalived.conf",
		"radvd/radvd.conf":            "/etc/radvd.conf",
		"conntrackd/conntrackd.conf":  "/etc/conntrackd/conntrackd.conf",
		"ssh/sshd_config":             "/etc/ssh/sshd_config.d/routier.conf",
		"modules-load.d/routier.conf": "/etc/modules-load.d/routier.conf",
		"gai/gai.conf":                "/etc/gai.conf",
		lldpdConfName:                 lldpdConfDest,
		lldpdOptsName:                 lldpdOptsDest,
		NamedConfName:                 NamedConfDest,
	}
	if d, ok := m[name]; ok {
		return d
	}

	if strings.HasPrefix(name, "wireguard/") {
		return "/etc/wireguard/" + filepath.Base(name)
	}

	if strings.HasPrefix(name, namedZonePfx) {
		return namedZoneDest + filepath.Base(name)
	}

	return "/etc/routier/out/" + name
}
