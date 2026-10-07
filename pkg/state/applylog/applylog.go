package applylog

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var dir = "/var/lib/routier/apply-logs"

func SetDir(d string) func() {
	prev := dir
	dir = d
	return func() { dir = prev }
}

const maxRecords = 50

type Record struct {
	ID           string     `json:"id"`
	Source       string     `json:"source"`
	ConfigPath   string     `json:"config_path,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	SnapID       string     `json:"snap_id,omitempty"`
	Result       string     `json:"result,omitempty"`
	HasBundle    bool       `json:"has_bundle,omitempty"`
	ConfirmedAt  *time.Time `json:"confirmed_at,omitempty"`
	RolledBackAt *time.Time `json:"rolledback_at,omitempty"`
}

type Recorder struct {
	rec     Record
	logFile *os.File
	prev    zerolog.Logger
}

func newID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return time.Now().Format("20060102-150405") + "-" + hex.EncodeToString(b)
}

func Start(source, configPath string) *Recorder {
	_ = os.MkdirAll(dir, 0700)
	r := &Recorder{rec: Record{
		ID:         newID(),
		Source:     source,
		ConfigPath: configPath,
		StartedAt:  time.Now(),
	}}

	if f, err := os.Create(filepath.Join(dir, r.rec.ID+".log")); err != nil {
		log.Warn().Err(err).Msg("applylog: cannot create log file")
	} else {
		r.logFile = f
	}

	r.save()

	r.prev = log.Logger
	var w io.Writer = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"}
	if r.logFile != nil {
		w = zerolog.MultiLevelWriter(w, zerolog.ConsoleWriter{Out: r.logFile, NoColor: true, TimeFormat: "15:04:05"})
	}

	log.Logger = zerolog.New(w).With().Timestamp().Logger()
	return r
}

func (r *Recorder) ID() string {
	return r.rec.ID
}

func (r *Recorder) MarkBundle() {
	r.rec.HasBundle = true
}

func (r *Recorder) Finish(snapID, result string) {
	now := time.Now()

	log.Logger = r.prev

	r.rec.FinishedAt = &now
	r.rec.SnapID = snapID
	r.rec.Result = result
	r.save()

	if r.logFile != nil {
		_ = r.logFile.Close()
	}

	prune()
}

func (r *Recorder) save() {
	data, _ := json.MarshalIndent(r.rec, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, r.rec.ID+".json"), data, 0600)
}

func MarkConfirmed(snapID string) {
	mark(snapID, func(r *Record) {
		now := time.Now()
		r.ConfirmedAt = &now
	})
}

func MarkRolledBack(snapID string) {
	mark(snapID, markRolledBackHandler)
}

func mark(snapID string, fn func(*Record)) {
	if snapID == "" {
		return
	}

	for _, rc := range List() {
		if rc.SnapID != snapID {
			continue
		}

		fn(&rc)
		data, _ := json.MarshalIndent(rc, "", "  ")
		_ = os.WriteFile(filepath.Join(dir, rc.ID+".json"), data, 0600)
		return
	}
}

func List() []Record {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var recs []Record
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}

		var r Record
		if json.Unmarshal(data, &r) == nil {
			recs = append(recs, r)
		}
	}

	sort.Slice(recs, func(i, j int) bool { return recs[i].StartedAt.After(recs[j].StartedAt) })
	return recs
}

func Read(id string) (string, error) {
	if strings.ContainsAny(id, "/\\") || strings.Contains(id, "..") {
		return "", fmt.Errorf("invalid id")
	}

	data, err := os.ReadFile(Path(id))
	return string(data), err
}

func prune() {
	recs := List()
	if len(recs) <= maxRecords {
		return
	}

	for _, r := range recs[maxRecords:] {
		_ = os.Remove(filepath.Join(dir, r.ID+".json"))
		_ = os.Remove(filepath.Join(dir, r.ID+".log"))
	}
}

func Path(id string) string { return filepath.Join(dir, id+".log") }

func markRolledBackHandler(r *Record) {
	now := time.Now()
	r.RolledBackAt = &now
	if r.Result == "" || r.Result == "applied" {
		r.Result = "rolledback"
	}
}
