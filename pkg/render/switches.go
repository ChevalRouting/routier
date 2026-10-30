package render

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
)

const (
	SwitchArtifactDir    = "/etc/routier/out/switches/"
	SwitchArtifactPrefix = "switches/"
)

func SwitchArtifactName(name string) string {
	return SwitchArtifactPrefix + name + ".json"
}

func SwitchFromArtifactName(name string) (string, bool) {
	if !strings.HasPrefix(name, SwitchArtifactPrefix) || !strings.HasSuffix(name, ".json") {
		return "", false
	}

	return strings.TrimSuffix(strings.TrimPrefix(name, SwitchArtifactPrefix), ".json"), true
}

func renderSwitches(cfg *config.Config) ([]Output, error) {
	names := make([]string, 0, len(cfg.Switches))
	for name := range cfg.Switches {
		names = append(names, name)
	}

	sort.Strings(names)
	out := make([]Output, 0, len(names))
	for _, name := range names {
		data, err := json.MarshalIndent(cfg.Switches[name], "", "  ")
		if err != nil {
			return nil, err
		}

		out = append(out, Output{Name: SwitchArtifactName(name), Dest: SwitchArtifactDir + name + ".json", Content: string(data) + "\n"})
	}

	return out, nil
}

func ReadSwitchArtifacts() map[string]*config.Switch {
	entries, err := os.ReadDir(SwitchArtifactDir)
	if err != nil {
		return nil
	}

	out := make(map[string]*config.Switch, len(entries))
	for _, entry := range entries {
		name, ok := SwitchFromArtifactName(SwitchArtifactPrefix + entry.Name())
		if !ok {
			continue
		}

		data, err := os.ReadFile(filepath.Join(SwitchArtifactDir, entry.Name()))
		if err != nil {
			continue
		}

		var sw config.Switch
		if json.Unmarshal(data, &sw) == nil {
			out[name] = &sw
		}
	}

	return out
}
