package render

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/ChevalRouting/routier/pkg/config"
)

const wgTemplate = `# routier:{{ .Name }}
[Interface]
Table = {{ if .WG.Table }}{{ .WG.Table }}{{ else }}off{{ end }}
{{- if .WG.PrivateKey }}
PrivateKey = {{ .WG.PrivateKey }}
{{- end }}
{{- if .WG.ListenPort }}
ListenPort = {{ .WG.ListenPort }}
{{- end }}
{{- range .WG.Addresses }}
Address = {{ . }}
{{- end }}
{{- if .WG.MTU }}
MTU = {{ .WG.MTU }}
{{- end }}
{{- range .WG.PreUp }}
PreUp = {{ . }}
{{- end }}
{{- if .WG.PrivateKeyFile }}
PostUp = wg set %i private-key {{ .WG.PrivateKeyFile }}
{{- end }}
{{- range .WG.Peers }}{{- if .PresharedKeyFile }}
PostUp = wg set %i peer {{ .PublicKey }} preshared-key {{ .PresharedKeyFile }}
{{- end }}{{- end }}
{{- range .WG.PostUp }}
PostUp = {{ . }}
{{- end }}
{{- range .WG.PreDown }}
PreDown = {{ . }}
{{- end }}
{{- range .WG.PostDown }}
PostDown = {{ . }}
{{- end }}
{{ range .WG.Peers }}
[Peer]
{{- if .Name }}
# {{ .Name }}
{{- end }}
PublicKey = {{ .PublicKey }}
{{- if .PresharedKey }}
PresharedKey = {{ .PresharedKey }}
{{- end }}
{{- if .Endpoint }}
Endpoint = {{ .Endpoint }}
{{- end }}
{{- if .AllowedIPs }}
AllowedIPs = {{ joinstr .AllowedIPs ", " }}
{{- end }}
{{- if .Keepalive }}
PersistentKeepalive = {{ .Keepalive }}
{{- end }}
{{ end }}`

type wgCtx struct {
	Name string
	WG   *config.Wireguard
}

func renderWireguard(cfg *config.Config) ([]Output, error) {
	funcs := template.FuncMap{
		"joinstr": func(s []string, sep string) string {
			return strings.Join(s, sep)
		},
	}

	names := make([]string, 0, len(cfg.Wireguard))
	for n := range cfg.Wireguard {
		names = append(names, n)
	}

	sort.Strings(names)

	var out []Output
	for _, name := range names {
		wg := cfg.Wireguard[name]

		tmplStr := wgTemplate
		userPath := filepath.Join(UserDir, "wireguard", name+".conf")
		if raw, err := os.ReadFile(userPath); err == nil {
			tmplStr = string(raw)
		}

		t, err := template.New("wg-" + name).Funcs(funcs).Parse(tmplStr)
		if err != nil {
			return nil, fmt.Errorf("wireguard template %s: %w", name, err)
		}

		var buf bytes.Buffer
		if err := t.Execute(&buf, wgCtx{Name: name, WG: wg}); err != nil {
			return nil, fmt.Errorf("wireguard %s: %w", name, err)
		}

		out = append(out, Output{
			Name:    "wireguard/" + name + ".conf",
			Dest:    "/etc/wireguard/" + name + ".conf",
			Content: buf.String(),
		})
	}

	return out, nil
}
