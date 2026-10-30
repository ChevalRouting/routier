package takeover

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	BackupRoot = "/var/lib/routier/host-backup"
	MarkerPath = "/var/lib/routier/taken-over"
)

var targets = []string{
	"/etc/network/interfaces",
	"/etc/network/interfaces.d",
	"/etc/dhcpcd.conf",
	"/etc/resolv.conf",
	"/etc/wpa_supplicant",
	"/etc/sysctl.conf",
	"/etc/sysctl.d",
	"/etc/nftables.nft",
	"/etc/conf.d/networking",
	"/etc/conf.d/dhcpcd",
	"/etc/frr",
}

var nativeServices = []string{"networking", "wpa_supplicant", "dhcpcd", "networkmanager", "connman"}

var nativeRunlevels = []string{"boot", "default"}

const loopbackInterfaces = "auto lo\niface lo inet loopback\n"

const restoreScript = `#!/bin/sh
set -eu
HERE=$(cd "$(dirname "$0")" && pwd)
while IFS= read -r path; do
	[ -n "$path" ] || continue
	mkdir -p "$(dirname "$path")"
	cp -a "$HERE$path" "$path"
	echo "restored $path"
done <"$HERE/manifest.txt"
for svc in networking dhcpcd wpa_supplicant; do
	[ -x "/etc/init.d/$svc" ] && rc-update add "$svc" boot 2>/dev/null || true
done
rc-update del routier-net boot 2>/dev/null || true
rc-update del routier default 2>/dev/null || true
rc-update del routier-ui default 2>/dev/null || true
rm -f ` + MarkerPath + `
echo "restored. reboot to apply."
`

func Done() bool {
	_, err := os.Stat(MarkerPath)
	return err == nil
}

func NativeEnabled(root string) bool {
	for _, lvl := range nativeRunlevels {
		for _, svc := range nativeServices {
			if _, err := os.Lstat(filepath.Join(root, "etc", "runlevels", lvl, svc)); err == nil {
				return true
			}
		}
	}

	return false
}

func EnsureOnConfirm() {
	if Done() {
		return
	}

	if !NativeEnabled("/") {
		if err := mark(); err != nil {
			log.Warn().Err(err).Msg("takeover marker not written")
		}

		return
	}

	dir, err := Run()
	if err != nil {
		log.Warn().Err(err).Msg("host network takeover failed")
		return
	}

	log.Info().Str("backup", dir).Msg("host networking handed over to routier")
}

func Run() (string, error) {
	ts := time.Now().Format("20060102-150405")
	dir := filepath.Join(BackupRoot, ts)

	saved, err := Backup("/", dir)
	if err != nil {
		return "", err
	}

	if len(saved) > 0 {
		latest := filepath.Join(BackupRoot, "latest")
		_ = os.Remove(latest)
		if err := os.Symlink(ts, latest); err != nil {
			return "", fmt.Errorf("link latest backup: %w", err)
		}
	}

	disableNative()
	if err := mark(); err != nil {
		return "", err
	}

	return dir, nil
}

func Backup(root, dir string) ([]string, error) {
	var saved []string
	for _, path := range targets {
		src := filepath.Join(root, path)
		fi, err := os.Stat(src)
		if err != nil {
			continue
		}

		dest := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return nil, err
		}

		if fi.IsDir() {
			err = copyDir(src, dest)
		} else {
			err = copyFile(src, dest)
		}

		if err != nil {
			return nil, fmt.Errorf("back up %s: %w", path, err)
		}

		saved = append(saved, path)
	}

	if len(saved) == 0 {
		return nil, nil
	}

	manifest := strings.Join(saved, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "manifest.txt"), []byte(manifest), 0644); err != nil {
		return nil, err
	}

	if err := os.WriteFile(filepath.Join(dir, "restore.sh"), []byte(restoreScript), 0755); err != nil {
		return nil, err
	}

	return saved, nil
}

func disableNative() {
	for _, svc := range nativeServices {
		if _, err := os.Stat("/etc/init.d/" + svc); err != nil {
			continue
		}

		for _, lvl := range nativeRunlevels {
			_ = exec.Command("rc-update", "del", svc, lvl).Run()
		}

		log.Info().Str("service", svc).Msg("disabled native network service, left running until reboot")
	}

	if _, err := os.Stat("/etc/network/interfaces"); err == nil {
		if err := os.WriteFile("/etc/network/interfaces", []byte(loopbackInterfaces), 0644); err != nil {
			log.Warn().Err(err).Msg("reset /etc/network/interfaces")
		}
	}
}

func mark() error {
	if err := os.MkdirAll(filepath.Dir(MarkerPath), 0700); err != nil {
		return err
	}

	return os.WriteFile(MarkerPath, []byte(time.Now().Format(time.RFC3339)+"\n"), 0600)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}

	defer func(action func() error) { _ = action() }(in.Close)
	fi, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fi.Mode())
	if err != nil {
		return err
	}

	defer func(action func() error) { _ = action() }(out.Close)
	_, err = io.Copy(out, in)
	return err
}

type dirCopy struct {
	src, dst string
}

func copyDir(src, dst string) error {
	c := dirCopy{src: src, dst: dst}
	return filepath.Walk(src, c.visit)
}

func (c dirCopy) visit(path string, fi os.FileInfo, err error) error {
	if err != nil {
		return fmt.Errorf("copy %s: %w", path, err)
	}

	rel, err := filepath.Rel(c.src, path)
	if err != nil {
		return err
	}

	target := filepath.Join(c.dst, rel)
	if fi.IsDir() {
		return os.MkdirAll(target, fi.Mode())
	}

	return copyFile(path, target)
}
