package model

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/ChevalRouting/routier/pkg/types"
	"gopkg.in/yaml.v3"
)

//go:embed catalog
var builtin embed.FS

const ExtraDir = "/etc/routier/switch-models"

var (
	defaultOnce    sync.Once
	defaultCatalog *Catalog
	defaultErr     error
)

var ErrModelNotFound = errors.New("switch model not found")

type matchSpec struct {
	SysDescr string `yaml:"sys_descr"`
}

type faceplateSpec struct {
	Group string `yaml:"group"`
	Ports string `yaml:"ports"`
	Kind  string `yaml:"kind"`
	Rows  int    `yaml:"rows"`
	Order string `yaml:"order"`
}

type modelSpec struct {
	ID          string                  `yaml:"id"`
	Vendor      string                  `yaml:"vendor"`
	Name        string                  `yaml:"name"`
	Driver      string                  `yaml:"driver"`
	Generic     bool                    `yaml:"generic"`
	Match       matchSpec               `yaml:"match"`
	Credentials []types.CredentialField `yaml:"credentials"`
	Faceplate   []faceplateSpec         `yaml:"faceplate"`
}

type entry struct {
	model   types.SwitchModel
	matcher *regexp.Regexp
}

type Catalog struct {
	entries []entry
}

func Load(extraDir string) (*Catalog, error) {
	catalog := &Catalog{}
	sub, err := fs.Sub(builtin, "catalog")
	if err != nil {
		return nil, err
	}

	if err := catalog.loadFS(sub); err != nil {
		return nil, err
	}

	if info, err := os.Stat(extraDir); extraDir != "" && err == nil && info.IsDir() {
		if err := catalog.loadFS(os.DirFS(extraDir)); err != nil {
			return nil, err
		}
	}

	slices.SortFunc(catalog.entries, compareEntries)
	return catalog, nil
}

func Default() (*Catalog, error) {
	defaultOnce.Do(loadDefault)
	return defaultCatalog, defaultErr
}

func loadDefault() {
	defaultCatalog, defaultErr = Load(ExtraDir)
}

func compareEntries(a, b entry) int {
	if a.model.Generic != b.model.Generic {
		if a.model.Generic {
			return 1
		}

		return -1
	}

	return strings.Compare(a.model.ID, b.model.ID)
}

type loader struct {
	catalog *Catalog
	fsys    fs.FS
}

func (l loader) visit(name string, d fs.DirEntry, err error) error {
	if err != nil || d.IsDir() || (path.Ext(name) != ".yaml" && path.Ext(name) != ".yml") {
		return err
	}

	return l.catalog.loadFile(l.fsys, name)
}

func (c *Catalog) loadFS(fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", loader{catalog: c, fsys: fsys}.visit)
}

func (c *Catalog) loadFile(fsys fs.FS, name string) error {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return err
	}

	var spec modelSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return fmt.Errorf("switch model %s: %w", name, err)
	}

	loaded, err := build(spec)
	if err != nil {
		return fmt.Errorf("switch model %s: %w", name, err)
	}

	c.entries = slices.DeleteFunc(c.entries, func(e entry) bool { return e.model.ID == loaded.model.ID })
	c.entries = append(c.entries, loaded)
	return nil
}

func build(spec modelSpec) (entry, error) {
	if spec.ID == "" || spec.Driver == "" || spec.Match.SysDescr == "" {
		return entry{}, errors.New("id, driver and match.sys_descr are required")
	}

	matcher, err := regexp.Compile(spec.Match.SysDescr)
	if err != nil {
		return entry{}, fmt.Errorf("match.sys_descr: %w", err)
	}

	model := types.SwitchModel{
		ID: spec.ID, Vendor: spec.Vendor, Name: spec.Name, Driver: spec.Driver, Generic: spec.Generic,
		Credentials: spec.Credentials, Faceplate: []types.FaceplateGroup{},
	}
	for _, group := range spec.Faceplate {
		ports, err := ExpandPorts(group.Ports)
		if err != nil {
			return entry{}, fmt.Errorf("faceplate %s: %w", group.Group, err)
		}

		rows := max(group.Rows, 1)
		model.Faceplate = append(model.Faceplate, types.FaceplateGroup{Group: group.Group, Kind: group.Kind, Rows: rows, Order: group.Order, Ports: ports})
	}

	return entry{model: model, matcher: matcher}, nil
}

func (c *Catalog) All() []types.SwitchModel {
	models := make([]types.SwitchModel, 0, len(c.entries))
	for _, e := range c.entries {
		models = append(models, e.model)
	}

	return models
}

func (c *Catalog) Get(id string) (types.SwitchModel, error) {
	for _, e := range c.entries {
		if e.model.ID == id {
			return e.model, nil
		}
	}

	return types.SwitchModel{}, fmt.Errorf("%w: %s", ErrModelNotFound, id)
}

func (c *Catalog) Match(sysDescr string) (types.SwitchModel, bool) {
	for _, e := range c.entries {
		if e.matcher.MatchString(sysDescr) {
			return e.model, true
		}
	}

	return types.SwitchModel{}, false
}
