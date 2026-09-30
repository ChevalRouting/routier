package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/host/boot"
)

const (
	LogPath     = "/var/lib/routier/upgrade.log"
	StatusPath  = "/run/routier/upgrade.json"
	versionPath = "/etc/routier/version"
)

type Package struct {
	Name string `json:"name"`
	Old  string `json:"old"`
	New  string `json:"new"`
}

type Available struct {
	Version        string    `json:"version"`
	Packages       []Package `json:"packages"`
	RebootRequired bool      `json:"reboot_required"`
	SelfUpdate     bool      `json:"self_update"`
	Live           bool      `json:"live"`
	CheckedAt      time.Time `json:"checked_at"`
}

type JobState string

const (
	StateIdle    JobState = "idle"
	StateRunning JobState = "running"
	StateDone    JobState = "done"
	StateFailed  JobState = "failed"
)

type JobStatus struct {
	State          JobState  `json:"state"`
	StartedAt      time.Time `json:"started_at,omitempty"`
	FinishedAt     time.Time `json:"finished_at,omitempty"`
	RebootRequired bool      `json:"reboot_required"`
	SelfUpdate     bool      `json:"self_update"`
	Error          string    `json:"error,omitempty"`
}

func Check(ctx context.Context) (Available, error) {
	result := Available{
		Version:   InstalledVersion(),
		Packages:  []Package{},
		Live:      boot.IsLive(),
		CheckedAt: time.Now(),
	}

	if err := run(ctx, io.Discard, "apk", "update"); err != nil {
		return result, fmt.Errorf("apk update: %w", err)
	}

	out, err := exec.CommandContext(ctx, "apk", "version", "-l", "<").Output()
	if err != nil {
		return result, fmt.Errorf("apk version: %w", err)
	}

	result.Packages = parseUpgradable(string(out))
	result.SelfUpdate = containsAny(result.Packages, "routier", "routier-ui", "routier-openrc")
	result.RebootRequired = containsKernel(result.Packages)
	return result, nil
}

func Upgrade(ctx context.Context, w io.Writer) error {
	if boot.IsLive() {
		return fmt.Errorf("cannot upgrade the live image: changes do not persist. Install to disk with setup-routier or rebuild the ISO")
	}

	if err := run(ctx, w, "apk", "update"); err != nil {
		return fmt.Errorf("apk update: %w", err)
	}

	if err := run(ctx, w, "apk", "upgrade", "--available"); err != nil {
		return fmt.Errorf("apk upgrade: %w", err)
	}

	return nil
}

func RebootPending() bool {
	running := boot.RunningRelease()
	if running == "" {
		return false
	}

	flavor := boot.RunningFlavor()
	kernels, _ := boot.DetectKernels("/")

	var newest string
	for _, k := range kernels {
		if k.Flavor == flavor && k.Version > newest {
			newest = k.Version
		}
	}

	return newest != "" && newest != running
}

func InstalledVersion() string {
	data, err := os.ReadFile(versionPath)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

func WriteStatus(status JobStatus) error {
	if err := os.MkdirAll(filepath.Dir(StatusPath), 0755); err != nil {
		return err
	}

	data, err := json.Marshal(status)
	if err != nil {
		return err
	}

	return os.WriteFile(StatusPath, data, 0644)
}

func ReadStatus() JobStatus {
	data, err := os.ReadFile(StatusPath)
	if err != nil {
		return JobStatus{State: StateIdle}
	}

	var status JobStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return JobStatus{State: StateIdle}
	}

	return status
}

func run(ctx context.Context, w io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

func parseUpgradable(out string) []Package {
	pkgs := []Package{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Installed:") || strings.HasPrefix(line, "WARNING:") {
			continue
		}

		left, right, ok := strings.Cut(line, " < ")
		if !ok {
			continue
		}

		name, old := splitNameVersion(strings.TrimSpace(left))
		if name == "" {
			continue
		}

		pkgs = append(pkgs, Package{Name: name, Old: old, New: strings.TrimSpace(right)})
	}

	return pkgs
}

func splitNameVersion(token string) (string, string) {
	fields := strings.Split(token, "-")
	if len(fields) < 3 {
		return "", ""
	}

	version := strings.Join(fields[len(fields)-2:], "-")
	name := strings.Join(fields[:len(fields)-2], "-")
	return name, version
}

func containsAny(pkgs []Package, names ...string) bool {
	for _, p := range pkgs {
		for _, n := range names {
			if p.Name == n {
				return true
			}
		}
	}

	return false
}

func containsKernel(pkgs []Package) bool {
	kernels, _ := boot.DetectKernels("/")
	flavors := map[string]bool{}
	for _, k := range kernels {
		flavors[k.Flavor] = true
	}

	for _, p := range pkgs {
		flavor, ok := strings.CutPrefix(p.Name, "linux-")
		if ok && flavors[flavor] {
			return true
		}
	}

	return false
}
