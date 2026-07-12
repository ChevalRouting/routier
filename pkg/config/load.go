package config

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const CurrentVersion = "v3.0.0"

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

	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}

	var cfg *Config
	if info.IsDir() {
		cfg, err = loadDir(abs)
		if err != nil {
			return nil, err
		}

		cfg.BaseDir = abs
	} else {
		cfg, err = loadFile(abs)
		if err != nil {
			return nil, err
		}

		cfg.BaseDir = filepath.Dir(abs)
	}

	if cfg.Version == "" {
		return nil, fmt.Errorf("config version is required (expected vX.X.X, current is %q)", CurrentVersion)
	}

	cfgMajor, ok := versionMajor(cfg.Version)
	if !ok {
		return nil, fmt.Errorf("config version %q is not valid (expected vX.X.X format)", cfg.Version)
	}

	curMajor, _ := versionMajor(CurrentVersion)
	if cfgMajor != curMajor {
		return nil, fmt.Errorf("config version %q is incompatible with %q (major version mismatch)", cfg.Version, CurrentVersion)
	}

	return cfg, nil
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
	data, _, err := MigrateBytes(data)
	if err != nil {
		return nil, err
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadDir(dir string) (*Config, error) {
	var files []string
	for _, ext := range []string{"*.yml", "*.yaml"} {
		found, _ := filepath.Glob(filepath.Join(dir, ext))
		files = append(files, found...)
	}

	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("no yaml files in %s", dir)
	}

	merged := &Config{}
	for _, f := range files {
		c, err := loadFile(f)
		if err != nil {
			return nil, err
		}

		merge(merged, c)
	}

	return merged, nil
}

func merge(dst, src *Config) {
	if src.Version != "" {
		dst.Version = src.Version
	}

	if src.Hostname != "" {
		dst.Hostname = src.Hostname
	}

	if src.Logging != nil {
		dst.Logging = src.Logging
	}

	if src.Routing != nil {
		dst.Routing = src.Routing
	}

	dst.Interfaces = mergeMap(dst.Interfaces, src.Interfaces)
	dst.Tunnels = mergeMap(dst.Tunnels, src.Tunnels)
	dst.Wireguard = mergeMap(dst.Wireguard, src.Wireguard)
	dst.Users = mergeMap(dst.Users, src.Users)
	dst.Services = mergeMap(dst.Services, src.Services)
	dst.Sysctl = mergeMapStr(dst.Sysctl, src.Sysctl)

	if src.DNS != nil {
		dst.DNS = src.DNS
	}
}

func mergeMap[V any](dst, src map[string]V) map[string]V {
	if src == nil {
		return dst
	}

	if dst == nil {
		dst = make(map[string]V)
	}

	maps.Copy(dst, src)
	return dst
}

func mergeMapStr(dst, src map[string]string) map[string]string {
	if src == nil {
		return dst
	}

	if dst == nil {
		dst = make(map[string]string)
	}

	maps.Copy(dst, src)
	return dst
}
