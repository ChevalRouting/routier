package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/spf13/cobra"
	vnl "github.com/vishvananda/netlink"
	"gopkg.in/yaml.v3"
)

var takeoverTargets = []string{
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

func newTakeoverCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "takeover",
		Short: "back up host network config and hand networking to routier",
		RunE:  runTakeover,
	}
}

func runTakeover(_ *cobra.Command, _ []string) error {
	const backupRoot = "/var/lib/routier/host-backup"
	ts := time.Now().Format("20060102-150405")
	backupDir := filepath.Join(backupRoot, ts)

	backedUp := backupHostFiles(backupDir)

	if _, err := os.Stat(defaultConfigPath); err == nil {
		fmt.Println("routier-takeover: existing routier config found; taking over networking")
		disableNativeNetworking()
	} else if err := generateFallbackConfig(backupDir); err == nil {
		disableNativeNetworking()
	} else {
		fmt.Fprintln(os.Stderr, "routier-takeover: WARNING: no routier config and no default route detected,")
		fmt.Fprintln(os.Stderr, "routier-takeover: WARNING: leaving native network stack enabled to avoid stranding this host.")
		fmt.Fprintln(os.Stderr, "routier-takeover: WARNING: configure routier from the web UI (routier-ui, port 8080), then re-run: routier takeover")
		return nil
	}

	if backedUp {
		latest := filepath.Join(backupRoot, "latest")
		_ = os.Remove(latest)
		if err := os.Symlink(ts, latest); err != nil {
			fmt.Fprintf(os.Stderr, "routier-takeover: symlink latest: %v\n", err)
		}

		fmt.Printf("routier-takeover: done. native config backed up under %s/latest\n", backupRoot)
	}

	return nil
}

func backupHostFiles(backupDir string) bool {
	var manifest []string
	for _, path := range takeoverTargets {
		fi, err := os.Stat(path)
		if err != nil {
			continue
		}

		dest := filepath.Join(backupDir, path)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "routier-takeover: mkdir %s: %v\n", filepath.Dir(dest), err)
			continue
		}

		if fi.IsDir() {
			_ = copyDir(path, dest)
		} else {
			_ = copyFile(path, dest)
		}

		manifest = append(manifest, path)
	}

	if len(manifest) == 0 {
		fmt.Println("routier-takeover: no host network files to back up")
		return false
	}

	var buf []byte
	for _, p := range manifest {
		buf = append(buf, []byte(p+"\n")...)
	}

	if err := os.WriteFile(filepath.Join(backupDir, "manifest.txt"), buf, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "routier-takeover: write manifest: %v\n", err)
	}

	writeRestoreScript(backupDir)
	fmt.Printf("routier-takeover: backed up host config to %s\n", backupDir)
	return true
}

func writeRestoreScript(backupDir string) {
	script := `#!/bin/sh
# Restore this host-config snapshot and re-enable Alpine's native networking,
# undoing the routier takeover. Run as root, then reboot.
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
rc-update del routier default 2>/dev/null || true
rc-update del routier-ui default 2>/dev/null || true
echo "restored. reboot to apply."
`
	if err := os.WriteFile(filepath.Join(backupDir, "restore.sh"), []byte(script), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "routier-takeover: write restore.sh: %v\n", err)
	}
}

func disableNativeNetworking() {
	for _, svc := range nativeServices {
		if _, err := os.Stat("/etc/init.d/" + svc); err != nil {
			continue
		}

		_ = exec.Command("rc-update", "del", svc, "boot").Run()
		_ = exec.Command("rc-update", "del", svc, "default").Run()
		fmt.Printf("routier-takeover: disabled native service: %s (left running until reboot)\n", svc)
	}

	if _, err := os.Stat("/etc/network/interfaces"); err == nil {
		loopback := "# Managed by routier, interfaces are configured via netlink, not ifupdown.\nauto lo\niface lo inet loopback\n"
		if err := os.WriteFile("/etc/network/interfaces", []byte(loopback), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "routier-takeover: write interfaces: %v\n", err)
		}
	}
}

func generateFallbackConfig(backupDir string) error {
	routes, err := vnl.RouteList(nil, vnl.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("list routes: %w", err)
	}

	var ifn string
	for _, r := range routes {
		if r.Dst == nil {
			link, err := vnl.LinkByIndex(r.LinkIndex)
			if err == nil {
				ifn = link.Attrs().Name
			}

			break
		}
	}

	if ifn == "" {
		return fmt.Errorf("no default route")
	}

	cfg, err := config.Default()
	if err != nil {
		return fmt.Errorf("load default config: %w", err)
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "routier"
	}
	cfg.Hostname = hostname
	cfg.Interfaces = map[string]*config.Interface{
		"uplink": {Select: ifn, Addresses: []string{"dhcp"}},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf(
		"# Auto-generated by routier takeover to preserve connectivity.\n"+
			"# Replace via the routier UI.\n"+
			"# Previous network config is backed up under %s.\n",
		backupDir,
	)

	if err := os.MkdirAll("/etc/routier", 0750); err != nil {
		return err
	}

	if err := os.WriteFile(defaultConfigPath, append([]byte(header), data...), 0640); err != nil {
		return err
	}

	fmt.Printf("routier-takeover: generated fallback config for interface %q (DHCP)\n", ifn)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}

	defer in.Close()
	fi, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fi.Mode())
	if err != nil {
		return err
	}

	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if fi.IsDir() {
			return os.MkdirAll(target, fi.Mode())
		}

		return copyFile(path, target)
	})
}
