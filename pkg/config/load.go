package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const CurrentVersion = "v5.0.0"

var versionRe = regexp.MustCompile(`^v(\d+)\.\d+\.\d+$`)

func versionMajor(v string) (string, bool) {
	m := versionRe.FindStringSubmatch(v)
	if m == nil {
		return "", false
	}

	return m[1], true
}

func Load(path string) (*Config, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	cfg, err := loadFile(abs)
	if err != nil {
		return nil, err
	}

	cfg.BaseDir = filepath.Dir(abs)
	if err := checkVersion(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func checkVersion(cfg *Config) error {
	if cfg.Version == "" {
		return fmt.Errorf("config version is required (expected vX.X.X, current is %q)", CurrentVersion)
	}

	cfgMajor, ok := versionMajor(cfg.Version)
	if !ok {
		return fmt.Errorf("config version %q is not valid (expected vX.X.X format)", cfg.Version)
	}

	curMajor, _ := versionMajor(CurrentVersion)
	if cfgMajor != curMajor {
		return fmt.Errorf("config version %q is incompatible with %q (major version mismatch)", cfg.Version, CurrentVersion)
	}

	return nil
}

func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

func LoadAndValidate(path string, resolveIfaces bool) (*Config, error) {
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	return validated(cfg, resolveIfaces)
}

func LoadBytesAndValidate(path string, data []byte, resolveIfaces bool) (*Config, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	cfg, err := LoadBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", abs, err)
	}

	cfg.BaseDir = filepath.Dir(abs)
	if err := checkVersion(cfg); err != nil {
		return nil, err
	}

	return validated(cfg, resolveIfaces)
}

func validated(cfg *Config, resolveIfaces bool) (*Config, error) {
	if errs := Validate(cfg, resolveIfaces); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}

		return nil, fmt.Errorf("validation failed: %s", strings.Join(msgs, "; "))
	}

	return cfg, nil
}

func loadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg, err := LoadBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return cfg, nil
}

func LoadBytes(data []byte) (*Config, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
