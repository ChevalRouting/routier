package friends

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/ChevalRouting/routier/pkg/types"
)

var actionRE = regexp.MustCompile(`(?s)\{\{-?\s*(.*?)\s*-?\}\}`)

type Vars struct {
	Hostname    string
	Fingerprint string
	Version     string
	Interfaces  map[string][]string
	Exports     map[string]string
}

func resolvePath(v Vars, path string) (string, error) {
	parts := strings.Split(path, ".")
	switch parts[0] {
	case "hostname":
		return v.Hostname, nil
	case "fingerprint":
		return v.Fingerprint, nil
	case "version":
		return v.Version, nil
	case "interfaces":
		if len(parts) < 3 {
			return "", fmt.Errorf("interfaces path needs <iface>.<address|addresses>")
		}

		addrs := v.Interfaces[parts[1]]
		switch parts[2] {
		case "address":
			if len(addrs) == 0 {
				return "", fmt.Errorf("friend interface %q has no address", parts[1])
			}

			return addrs[0], nil
		case "addresses":
			return strings.Join(addrs, ","), nil
		}
	}

	return "", fmt.Errorf("unknown friend path %q", path)
}

func HasTemplate(raw string) bool {
	return strings.Contains(raw, "{{")
}

const DefaultCachePath = "/var/lib/routier/friends_cache.json"

func CachedVars() map[string]Vars {
	return LoadCacheVars(DefaultCachePath)
}

type cachedVarsFile struct {
	Status     map[string]*types.FriendStatus     `json:"status"`
	Interfaces map[string][]types.FriendInterface `json:"interfaces"`
}

func LoadCacheVars(path string) map[string]Vars {
	out := map[string]Vars{}

	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}

	var f cachedVarsFile
	if json.Unmarshal(data, &f) != nil {
		return out
	}

	for name, st := range f.Status {
		out[name] = Vars{Hostname: st.Hostname, Fingerprint: st.Fingerprint, Version: st.Version, Interfaces: map[string][]string{}}
	}

	for name, ifaces := range f.Interfaces {
		v, ok := out[name]
		if !ok {
			v = Vars{Interfaces: map[string][]string{}}
		}

		if v.Interfaces == nil {
			v.Interfaces = map[string][]string{}
		}

		for _, i := range ifaces {
			v.Interfaces[i.Name] = i.Addresses
		}

		out[name] = v
	}

	return out
}

func Interpolate(raw string, data map[string]Vars) (string, error) {
	if !HasTemplate(raw) {
		return raw, nil
	}

	funcs := template.FuncMap{
		"friend": func(name, path string) (string, error) {
			v, ok := data[name]
			if !ok {
				return "", fmt.Errorf("unknown friend %q", name)
			}

			return resolvePath(v, path)
		},
	}

	var firstErr error
	out := actionRE.ReplaceAllStringFunc(raw, func(action string) string {
		inner := actionRE.FindStringSubmatch(action)[1]
		if fields := strings.Fields(inner); len(fields) == 0 || fields[0] != "friend" {
			return action
		}

		t, err := template.New("friend").Funcs(funcs).Parse(action)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("parse friend template: %w", err)
			}

			return action
		}

		var b bytes.Buffer
		if err := t.Execute(&b, nil); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("interpolate config: %w", err)
			}

			return action
		}

		return b.String()
	})

	if firstErr != nil {
		return "", firstErr
	}

	return out, nil
}
