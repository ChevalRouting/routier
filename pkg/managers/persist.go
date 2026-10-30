package managers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ChevalRouting/routier/pkg/config"
	"gopkg.in/yaml.v3"
)

const DefaultConfigPath = "/etc/routier/config.yml"

var ErrApplyPending = errors.New("an apply is waiting for confirmation; confirm or roll it back first")

func RequireNoPending() error {
	p, err := PendingStatus()
	if err != nil {
		return fmt.Errorf("read pending apply: %w", err)
	}

	if p != nil {
		return ErrApplyPending
	}

	return nil
}

func (o ApplyOptions) configPath() string {
	if o.ConfigPath == "" {
		return DefaultConfigPath
	}

	return o.ConfigPath
}

func withDDNSKey(data []byte, cfg *config.Config) ([]byte, error) {
	if cfg.DHCP == nil || cfg.DHCP.DDNS == nil {
		return data, nil
	}

	var persisted config.Config
	if err := yaml.Unmarshal(data, &persisted); err != nil {
		return nil, fmt.Errorf("decode config to persist: %w", err)
	}

	if persisted.DHCP == nil || persisted.DHCP.DDNS == nil {
		return data, nil
	}

	persisted.DHCP.DDNS.Key = cfg.DHCP.DDNS.Key
	persisted.DHCP.DDNS.Algorithm = cfg.DHCP.DDNS.Algorithm

	return yaml.Marshal(&persisted)
}

func writeConfigFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
