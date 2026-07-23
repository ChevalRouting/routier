package config

import _ "embed"

//go:embed default.yml
var defaultYAML []byte

func DefaultBytes() []byte {
	return defaultYAML
}

func Default() (*Config, error) {
	return LoadBytes(defaultYAML)
}
