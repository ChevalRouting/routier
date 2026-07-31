package failures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/types"
	"gopkg.in/yaml.v3"
)

var baseDir = "/var/lib/routier/failures"

const maxBundles = 20

func SetDir(d string) func() {
	prev := baseDir
	baseDir = d
	return func() { baseDir = prev }
}

type Meta struct {
	ID        string                `json:"id"`
	Time      time.Time             `json:"time"`
	Source    string                `json:"source"`
	SnapID    string                `json:"snap_id,omitempty"`
	Errors    []types.ArtifactError `json:"errors"`
	Artifacts []string              `json:"artifacts"`
}

func safeID(id string) bool {
	return id != "" && !strings.ContainsAny(id, "/\\") && !strings.Contains(id, "..")
}

func Save(id, source, snapID string, cfg *config.Config, outputs []render.Output, errs []types.ArtifactError) error {
	if !safeID(id) {
		return fmt.Errorf("invalid bundle id")
	}

	bundleDir := filepath.Join(baseDir, id)
	renderedDir := filepath.Join(bundleDir, "rendered")
	if err := os.MkdirAll(renderedDir, 0700); err != nil {
		return err
	}

	dests := make([]string, 0, len(outputs))
	for _, o := range outputs {
		p := filepath.Join(renderedDir, filepath.FromSlash(o.Dest))
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			return err
		}

		if err := os.WriteFile(p, []byte(o.Content), 0600); err != nil {
			return err
		}

		dests = append(dests, o.Dest)
	}

	sort.Strings(dests)

	if data, err := yaml.Marshal(cfg); err == nil {
		_ = os.WriteFile(filepath.Join(bundleDir, "config.yml"), data, 0600)
	}

	meta := Meta{
		ID:        id,
		Time:      time.Now(),
		Source:    source,
		SnapID:    snapID,
		Errors:    errs,
		Artifacts: dests,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(bundleDir, "meta.json"), data, 0600); err != nil {
		return err
	}

	prune()
	return nil
}

func List() []Meta {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil
	}

	var metas []Meta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		m, err := Get(e.Name())
		if err != nil {
			continue
		}

		metas = append(metas, m)
	}

	sort.Slice(metas, func(i, j int) bool { return metas[i].Time.After(metas[j].Time) })
	return metas
}

func Get(id string) (Meta, error) {
	if !safeID(id) {
		return Meta{}, fmt.Errorf("invalid bundle id")
	}

	data, err := os.ReadFile(filepath.Join(baseDir, id, "meta.json"))
	if err != nil {
		return Meta{}, err
	}

	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return Meta{}, err
	}

	return m, nil
}

func ReadArtifact(id, dest string) ([]byte, error) {
	if !safeID(id) {
		return nil, fmt.Errorf("invalid bundle id")
	}

	rendered := filepath.Join(baseDir, id, "rendered")
	p := filepath.Join(rendered, filepath.FromSlash(dest))
	if !strings.HasPrefix(filepath.Clean(p), filepath.Clean(rendered)+string(os.PathSeparator)) {
		return nil, fmt.Errorf("invalid artifact path")
	}

	return os.ReadFile(p)
}

func Dir(id string) string {
	return filepath.Join(baseDir, id)
}

func Remove(id string) error {
	if !safeID(id) {
		return fmt.Errorf("invalid bundle id")
	}

	return os.RemoveAll(filepath.Join(baseDir, id))
}

func prune() {
	metas := List()
	if len(metas) <= maxBundles {
		return
	}

	for _, m := range metas[maxBundles:] {
		_ = os.RemoveAll(filepath.Join(baseDir, m.ID))
	}
}
