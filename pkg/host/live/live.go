package live

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	MountsPath  = "/proc/mounts"
	ZoneinfoDir = "/usr/share/zoneinfo"
	daemonInit  = "/etc/init.d/routier"
)

type Disk struct {
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	Model     string `json:"model,omitempty" validate:"optional"`
	Removable bool   `json:"removable"`
	BootMedia bool   `json:"boot_media"`
}

func Booted(mounts string) bool {
	f, err := os.Open(mounts)
	if err != nil {
		return false
	}

	defer func(action func() error) { _ = action() }(f.Close)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 && fields[1] == "/" {
			return fields[2] == "tmpfs" || fields[2] == "overlay"
		}
	}

	return false
}

func InstallerOnly() bool {
	_, err := os.Stat(daemonInit)
	return err != nil
}

type blockDevice struct {
	Name        string        `json:"name"`
	Size        json.Number   `json:"size"`
	Model       string        `json:"model"`
	Removable   any           `json:"rm"`
	Type        string        `json:"type"`
	Mountpoint  string        `json:"mountpoint"`
	Mountpoints []string      `json:"mountpoints"`
	Children    []blockDevice `json:"children"`
}

type lsblkOutput struct {
	Blockdevices []blockDevice `json:"blockdevices"`
}

func Disks() ([]Disk, error) {
	out, err := exec.Command("lsblk", "-J", "-b", "-o", "NAME,SIZE,MODEL,RM,TYPE,MOUNTPOINT").Output()
	if err != nil {
		return nil, err
	}

	return ParseDisks(out)
}

func ParseDisks(data []byte) ([]Disk, error) {
	var parsed lsblkOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}

	disks := []Disk{}
	for _, dev := range parsed.Blockdevices {
		if dev.Type != "disk" || strings.HasPrefix(dev.Name, "loop") || strings.HasPrefix(dev.Name, "zram") {
			continue
		}

		size, _ := dev.Size.Int64()
		disks = append(disks, Disk{
			Path:      "/dev/" + dev.Name,
			Size:      size,
			Model:     strings.TrimSpace(dev.Model),
			Removable: truthy(dev.Removable),
			BootMedia: holdsBootMedia(dev),
		})
	}

	return disks, nil
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "1" || t == "true"
	case float64:
		return t != 0
	}

	return false
}

func holdsBootMedia(dev blockDevice) bool {
	points := append([]string{dev.Mountpoint}, dev.Mountpoints...)
	for _, mp := range points {
		if strings.HasPrefix(mp, "/media/") || strings.HasPrefix(mp, "/run/media/") || mp == "/.modloop" {
			return true
		}
	}

	for _, child := range dev.Children {
		if holdsBootMedia(child) {
			return true
		}
	}

	return false
}

var skippedZoneDirs = map[string]bool{"posix": true, "right": true, "Etc": true}

type zoneCollector struct {
	root  string
	zones []string
}

func Timezones(root string) []string {
	c := &zoneCollector{root: root}
	_ = filepath.WalkDir(root, c.visit)
	sort.Strings(c.zones)
	return c.zones
}

func (c *zoneCollector) visit(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return nil
	}

	rel, err := filepath.Rel(c.root, path)
	if err != nil || rel == "." {
		return nil
	}

	if d.IsDir() {
		if skippedZoneDirs[rel] {
			return filepath.SkipDir
		}

		return nil
	}

	if strings.Contains(rel, ".") || rel[0] < 'A' || rel[0] > 'Z' {
		return nil
	}

	c.zones = append(c.zones, rel)
	return nil
}
