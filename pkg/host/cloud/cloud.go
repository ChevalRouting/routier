package cloud

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChevalRouting/routier/pkg/cibi"
	"gopkg.in/yaml.v3"
)

const (
	CloudConfigPath = "/var/lib/cloud/instance/cloud-config.txt"
	InstanceIDPath  = "/var/lib/cloud/data/instance-id"
	StampPath       = "/var/lib/routier/cloud-instance"
	NetworkDropIn   = "/etc/cloud/cloud.cfg.d/99-routier-network.cfg"
)

const networkDisabled = "network:\n  config: disabled\n"

type Provision struct {
	Config   []byte
	Password string
	Cibi     *cibi.Join
}

type userData struct {
	Routier *routierSection `yaml:"routier"`
}

type routierSection struct {
	Config   yaml.Node  `yaml:"config"`
	Password string     `yaml:"password"`
	Cibi     *cibi.Join `yaml:"cibi"`
}

func Load(path string) (Provision, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Provision{}, nil
	}

	if err != nil {
		return Provision{}, err
	}

	return Parse(data)
}

func Parse(data []byte) (Provision, error) {
	var ud userData
	if err := yaml.Unmarshal(data, &ud); err != nil {
		return Provision{}, fmt.Errorf("parse cloud-config: %w", err)
	}

	if ud.Routier == nil {
		return Provision{}, nil
	}

	p := Provision{Password: ud.Routier.Password, Cibi: ud.Routier.Cibi}
	node := ud.Routier.Config
	switch node.Kind {
	case 0:
	case yaml.ScalarNode:
		if strings.TrimSpace(node.Value) != "" {
			p.Config = []byte(node.Value)
		}
	case yaml.MappingNode:
		out, err := yaml.Marshal(&node)
		if err != nil {
			return Provision{}, err
		}

		p.Config = out
	default:
		return Provision{}, fmt.Errorf("routier.config must be a mapping or a YAML string")
	}

	return p, nil
}

func InstanceID(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

func Provisioned(stamp, id string) bool {
	return id != "" && InstanceID(stamp) == id
}

func MarkProvisioned(stamp, id string) error {
	if err := os.MkdirAll(filepath.Dir(stamp), 0700); err != nil {
		return err
	}

	return os.WriteFile(stamp, []byte(id+"\n"), 0600)
}

func DisableNetworkConfig(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(networkDisabled), 0644)
}
