package diffutil

import (
	"fmt"

	"github.com/ChevalRouting/routier/pkg/types"
	"gopkg.in/yaml.v3"
)

func normalizeYAML(data []byte) (string, error) {
	var value any
	if err := yaml.Unmarshal(data, &value); err != nil {
		return "", err
	}

	out, err := yaml.Marshal(value)
	return string(out), err
}

func YAML(current, staged []byte) ([]types.DiffLine, error) {
	before, err := normalizeYAML(current)
	if err != nil {
		return nil, fmt.Errorf("current YAML: %w", err)
	}

	after, err := normalizeYAML(staged)
	if err != nil {
		return nil, fmt.Errorf("staged YAML: %w", err)
	}

	return Lines(before, after), nil
}
