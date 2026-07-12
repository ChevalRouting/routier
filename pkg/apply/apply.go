package apply

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/rs/zerolog/log"
)

const stateDir = "/var/lib/routier"

type snapshot struct {
	ID    string            `json:"id"`
	Time  time.Time         `json:"time"`
	Files map[string]string `json:"files"`
}

func Write(outputs []render.Output, dryRun bool, snapshotAlso ...string) (snapID string, changed []string, err error) {
	if dryRun {
		for _, o := range outputs {
			log.Info().Str("dest", o.Dest).Msg("would write")
		}

		return "", nil, nil
	}

	snap := takeSnapshot(outputs, snapshotAlso)
	if err := saveSnapshot(snap); err != nil {
		return "", nil, fmt.Errorf("snapshot: %w", err)
	}

	for _, o := range outputs {
		if old, ok := snap.Files[o.Dest]; ok && old == o.Content {
			log.Debug().Str("dest", o.Dest).Msg("unchanged")
			continue
		}

		if err := writeFile(o); err != nil {
			log.Error().Str("dest", o.Dest).Err(err).Msg("write failed, rolling back")
			_, _ = Rollback(snap.ID)
			return "", nil, err
		}

		log.Info().Str("dest", o.Dest).Msg("wrote")
		changed = append(changed, o.Name)
	}

	return snap.ID, changed, nil
}

func modeFor(dest string) os.FileMode {
	if filepath.Dir(dest) == "/etc/wireguard" || filepath.Ext(dest) == ".yml" {
		return 0600
	}

	if filepath.Ext(dest) == ".sh" {
		return 0755
	}

	return 0644
}

func writeFile(o render.Output) error {
	if err := os.MkdirAll(filepath.Dir(o.Dest), 0755); err != nil {
		return err
	}

	return os.WriteFile(o.Dest, []byte(o.Content), modeFor(o.Dest))
}

func newSnapID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return time.Now().Format("20060102-150405") + "-" + hex.EncodeToString(b)
}

func readOrEmpty(path string) string {
	if data, err := os.ReadFile(path); err == nil {
		return string(data)
	}

	return ""
}

func takeSnapshot(outputs []render.Output, also []string) snapshot {
	s := snapshot{
		ID:    newSnapID(),
		Time:  time.Now(),
		Files: make(map[string]string),
	}

	for _, o := range outputs {
		s.Files[o.Dest] = readOrEmpty(o.Dest)
	}

	for _, dest := range also {
		s.Files[dest] = readOrEmpty(dest)
	}

	return s
}

func saveSnapshot(s snapshot) error {
	dir := filepath.Join(stateDir, "snapshots")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, s.ID+".json"), data, 0600)
}

func Rollback(id string) ([]string, error) {
	if id == "" {
		var err error
		id, err = latestSnapshot()
		if err != nil {
			return nil, err
		}
	}

	path := filepath.Join(stateDir, "snapshots", id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("snapshot %s: %w", id, err)
	}

	var s snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	var names []string
	for dest, old := range s.Files {
		current, _ := os.ReadFile(dest)
		if old == "" {
			if len(current) == 0 {
				log.Debug().Str("dest", dest).Msg("already absent, skipping")
				continue
			}

			if err := os.Remove(dest); err != nil {
				log.Error().Str("dest", dest).Err(err).Msg("remove failed")
				continue
			}

			log.Info().Str("dest", dest).Msg("removed")
		} else {
			if string(current) == old {
				log.Debug().Str("dest", dest).Msg("unchanged, skipping")
				continue
			}

			if err := os.WriteFile(dest, []byte(old), modeFor(dest)); err != nil {
				log.Error().Str("dest", dest).Err(err).Msg("restore failed")
				continue
			}

			log.Info().Str("dest", dest).Msg("restored")
		}

		if n := render.NameForDest(dest); n != "" {
			names = append(names, n)
		}
	}

	return names, nil
}

func latestSnapshot() (string, error) {
	ids, err := ListSnapshots()
	if err != nil || len(ids) == 0 {
		return "", fmt.Errorf("no snapshots found")
	}

	return ids[len(ids)-1], nil
}

type SnapshotInfo struct {
	ID    string    `json:"id"`
	Time  time.Time `json:"time"`
	Files []string  `json:"files"`
}

func ListSnapshotInfos() ([]SnapshotInfo, error) {
	dir := filepath.Join(stateDir, "snapshots")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	var infos []SnapshotInfo
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}

		var s snapshot
		if json.Unmarshal(data, &s) != nil {
			continue
		}

		dests := make([]string, 0, len(s.Files))
		for dest := range s.Files {
			dests = append(dests, dest)
		}

		sort.Strings(dests)
		infos = append(infos, SnapshotInfo{ID: s.ID, Time: s.Time, Files: dests})
	}

	sort.Slice(infos, func(i, j int) bool { return infos[i].Time.After(infos[j].Time) })
	return infos, nil
}

func ListSnapshots() ([]string, error) {
	dir := filepath.Join(stateDir, "snapshots")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			ids = append(ids, e.Name()[:len(e.Name())-5])
		}
	}

	return ids, nil
}
