package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ChevalRouting/routier/pkg/boot"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	etcRoutierSize    = "500MiB"
	varLibRoutierSize = "20%VG"
	rootSize          = "100%FREE"
	maxSwapMiB        = 4096
)

func newSetupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "interactive disk installer (LVM+xfs, GRUB EFI, reboot)",
		RunE:  runSetup,
	}
}

type installer struct {
	tty     *os.File
	scanner *bufio.Scanner
}

func (s *installer) msg(format string, args ...any) {
	fmt.Fprintf(s.tty, "\n\033[1;34m==> %s\033[0m\n", fmt.Sprintf(format, args...))
}

func (s *installer) ask(prompt, def string) string {
	fmt.Fprint(s.tty, prompt)
	if s.scanner.Scan() {
		if line := strings.TrimSpace(s.scanner.Text()); line != "" {
			return line
		}
	}

	return def
}

func (s *installer) confirm(prompt string) bool {
	fmt.Fprintf(s.tty, "%s [y/N] ", prompt)
	if s.scanner.Scan() {
		r := strings.TrimSpace(s.scanner.Text())
		return r == "y" || r == "Y"
	}

	return false
}

func (s *installer) run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = s.tty
	cmd.Stderr = s.tty
	return cmd.Run()
}

func (s *installer) output(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return strings.TrimSpace(string(out)), err
}

func (s *installer) deviceUUID(dev string) (string, error) {
	out, err := s.output("blkid", dev)
	if err != nil {
		return "", fmt.Errorf("read UUID of %s: %w", dev, err)
	}

	uuid := blkidAttr(out, "UUID")
	if uuid == "" {
		return "", fmt.Errorf("no UUID for %s", dev)
	}

	return uuid, nil
}

func blkidAttr(line, key string) string {
	marker := " " + key + "=\""
	i := strings.Index(line, marker)
	if i < 0 {
		return ""
	}

	rest := line[i+len(marker):]
	j := strings.Index(rest, "\"")
	if j < 0 {
		return ""
	}

	return rest[:j]
}

type copyEntry struct {
	src, dst string
	mode     os.FileMode
}

type xfsFormat struct {
	label, dev string
}

type configMount struct {
	dev, dst string
	mode     os.FileMode
}

func (s *installer) askInterfaces(nics []string) (map[string]*config.Interface, string) {
	ifaces := make(map[string]*config.Interface, len(nics))
	hasInternet := false

	for _, nic := range nics {
		s.msg("Interface %s", nic)
		role := strings.ToLower(s.ask(fmt.Sprintf("  Role for %s [internet/local/none] (none): ", nic), "none"))

		if role == "none" || role == "n" || role == "unused" {
			ifaces[nic] = &config.Interface{Select: nic}
			continue
		}

		if role == "local" || role == "l" {
			iface := &config.Interface{Select: nic}
			if addr := s.ask("    Router address (CIDR, e.g. 192.168.1.1/24): ", ""); addr != "" {
				iface.Addresses = []string{addr}
			}
			ifaces[nic] = iface
			continue
		}

		if role == "internet" || role == "i" {
			if hasInternet {
				s.msg("  Internet uplink already selected; leaving %s unused", nic)
				ifaces[nic] = &config.Interface{Select: nic}
				continue
			}
			hasInternet = true
			ifaces[nic] = &config.Interface{Select: nic, Addresses: []string{"dhcp", "slaac"}}
			continue
		}

		s.msg("  Unknown role %q; leaving %s unused", role, nic)
		ifaces[nic] = &config.Interface{Select: nic}
	}

	return ifaces, ""
}

// bootstrapPackages returns the package set installed onto the target system:
// everything the live image declares in /etc/apk/world, keeping the installed
// system at parity with the ISO, plus a few install-only extras that world does
// not carry. A small essential set is always included so the target stays
// bootable even if world cannot be read.
func bootstrapPackages() []string {
	pkgs := []string{
		"alpine-base", "grub-efi", "efibootmgr",
		"routier", "routier-openrc", "doas", "openssl",
	}

	pkgs = append(pkgs, worldPackages("/etc/apk/world")...)

	seen := make(map[string]struct{}, len(pkgs))
	out := pkgs[:0]
	for _, name := range pkgs {
		if _, ok := seen[name]; ok {
			continue
		}

		seen[name] = struct{}{}
		out = append(out, name)
	}

	return out
}

func worldPackages(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		if name := strings.TrimSpace(line); name != "" && !strings.HasPrefix(name, "#") {
			out = append(out, name)
		}
	}

	return out
}

func runSetup(_ *cobra.Command, _ []string) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("must be run as root")
	}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open /dev/tty: %w", err)
	}

	defer tty.Close()

	s := &installer{tty: tty, scanner: bufio.NewScanner(tty)}

	s.msg("Routier installer")

	hostname := "routier"

	fmt.Fprintln(s.tty, "Common timezones: UTC, America/New_York, America/Chicago, America/Los_Angeles,")
	fmt.Fprintln(s.tty, "  Europe/London, Europe/Paris, Europe/Berlin, Asia/Tokyo, Asia/Shanghai")
	timezone := s.ask("Timezone [UTC]: ", "UTC")
	if _, err := os.Stat("/usr/share/zoneinfo/" + timezone); err != nil {
		return fmt.Errorf("unknown timezone: %s (check /usr/share/zoneinfo/)", timezone)
	}

	ifaces, gateway := s.askInterfaces(usableNICs())

	s.msg("Available disks:")
	_ = s.run("lsblk", "-d", "-o", "NAME,SIZE,MODEL")

	defaultDisk := firstSDDisk()
	disk := s.ask(fmt.Sprintf("Disk to install to [%s]: ", defaultDisk), defaultDisk)

	fi, err := os.Stat(disk)
	if err != nil {
		return fmt.Errorf("disk not found: %s", disk)
	}

	if fi.Mode()&os.ModeDevice == 0 || fi.Mode()&os.ModeCharDevice != 0 {
		return fmt.Errorf("not a block device: %s", disk)
	}

	diskSize, _ := s.output("lsblk", "-d", "-n", "-o", "SIZE", disk)
	fmt.Fprintf(s.tty, "\n\033[1;31mWARNING:\033[0m all data on %s (%s) will be permanently erased.\n", disk, diskSize)
	if !s.confirm("Continue?") {
		return nil
	}

	s.msg("Partitioning %s...", disk)
	if err := s.run("wipefs", "-af", disk); err != nil {
		return fmt.Errorf("wipefs: %w", err)
	}

	if err := s.run("parted", "-s", disk,
		"mklabel", "gpt",
		"mkpart", "ESP", "fat32", "1MiB", "513MiB",
		"set", "1", "esp", "on",
		"mkpart", "LVM", "513MiB", "100%",
	); err != nil {
		return fmt.Errorf("parted: %w", err)
	}

	_ = s.run("partprobe", disk)
	_ = exec.Command("mdev", "-s").Run()

	var part1, part2 string
	if strings.Contains(disk, "nvme") || strings.Contains(disk, "mmcblk") {
		part1, part2 = disk+"p1", disk+"p2"
	} else {
		part1, part2 = disk+"1", disk+"2"
	}

	if err := waitForDevice(part1, 10); err != nil {
		return fmt.Errorf("partition %s did not appear: %w", part1, err)
	}

	swapMiB := swapSizeMiB()

	s.msg("Creating LVM (etc-routier %s, swap %dMiB, var-routier %s, root remainder)...", etcRoutierSize, swapMiB, varLibRoutierSize)
	if err := s.run("pvcreate", "-ff", "-y", part2); err != nil {
		return fmt.Errorf("pvcreate: %w", err)
	}

	if err := s.run("vgcreate", "routier", part2); err != nil {
		return fmt.Errorf("vgcreate: %w", err)
	}

	if err := s.run("lvcreate", "-y", "-L", etcRoutierSize, "-n", "config", "routier"); err != nil {
		return fmt.Errorf("lvcreate etcroutier: %w", err)
	}

	if swapMiB > 0 {
		if err := s.run("lvcreate", "-y", "-L", fmt.Sprintf("%dMiB", swapMiB), "-n", "swap", "routier"); err != nil {
			return fmt.Errorf("lvcreate swap: %w", err)
		}
	}

	if err := s.run("lvcreate", "-y", "-l", varLibRoutierSize, "-n", "data", "routier"); err != nil {
		return fmt.Errorf("lvcreate routier-data: %w", err)
	}

	if err := s.run("lvcreate", "-y", "-l", rootSize, "-n", "root", "routier"); err != nil {
		return fmt.Errorf("lvcreate root: %w", err)
	}

	s.msg("Formatting filesystems...")
	if err := s.run("mkfs.fat", "-F32", "-n", "EFI", part1); err != nil {
		return fmt.Errorf("mkfs.fat: %w", err)
	}

	for _, fs := range []xfsFormat{
		{"root", "/dev/routier/root"},
		{"etcroutier", "/dev/routier/config"},
		{"routier-data", "/dev/routier/data"},
	} {
		args := []string{"-f", "-q"}
		if fs.dev == "/dev/routier/root" {
			args = append(args, "-L", "root")
		}

		args = append(args, fs.dev)
		if err := s.run("mkfs.xfs", args...); err != nil {
			return fmt.Errorf("mkfs.xfs %s: %w", fs.label, err)
		}
	}

	if swapMiB > 0 {
		if err := s.run("mkswap", "-L", "swap", "/dev/routier/swap"); err != nil {
			return fmt.Errorf("mkswap: %w", err)
		}

		_ = s.run("swapon", "/dev/routier/swap")
	}

	s.msg("Mounting target...")
	if err := s.run("mount", "/dev/routier/root", "/mnt"); err != nil {
		return fmt.Errorf("mount root: %w", err)
	}

	if err := os.MkdirAll("/mnt/boot/efi", 0755); err != nil {
		return err
	}

	if err := s.run("mount", part1, "/mnt/boot/efi"); err != nil {
		return fmt.Errorf("mount efi: %w", err)
	}

	for _, m := range []configMount{
		{"/dev/routier/config", "/mnt/etc/routier", 0750},
		{"/dev/routier/data", "/mnt/var/lib/routier", 0700},
	} {
		if err := os.MkdirAll(m.dst, 0755); err != nil {
			return err
		}

		if err := s.run("mount", m.dev, m.dst); err != nil {
			return fmt.Errorf("mount %s: %w", m.dst, err)
		}

		if err := os.Chmod(m.dst, m.mode); err != nil {
			return err
		}
	}

	s.msg("Installing Alpine + Routier...")
	if err := os.MkdirAll("/mnt/etc/apk/keys", 0755); err != nil {
		return err
	}

	if keyEntries, err := os.ReadDir("/etc/apk/keys"); err == nil {
		for _, e := range keyEntries {
			data, _ := os.ReadFile(filepath.Join("/etc/apk/keys", e.Name()))
			_ = os.WriteFile(filepath.Join("/mnt/etc/apk/keys", e.Name()), data, 0644)
		}
	}

	arch, err := s.output("apk", "--print-arch")
	if err != nil {
		return fmt.Errorf("apk --print-arch: %w", err)
	}

	if arch != "aarch64" && arch != "x86_64" {
		return fmt.Errorf("unsupported arch: %s", arch)
	}

	reposRaw, _ := os.ReadFile("/etc/apk/repositories")
	var remoteRepos []string
	for _, line := range strings.Split(string(reposRaw), "\n") {
		if l := strings.TrimSpace(line); l != "" && !strings.HasPrefix(l, "#") && !strings.HasPrefix(l, "/") {
			remoteRepos = append(remoteRepos, l)
		}
	}

	_ = os.WriteFile("/mnt/etc/apk/repositories", []byte(strings.Join(remoteRepos, "\n")+"\n"), 0644)

	installRepos := string(reposRaw)
	if media := bootRepository(arch); media != "" {
		installRepos = media + "\n" + installRepos
		s.msg("Using packages embedded on the installer media (%s)", media)
	}

	_ = os.WriteFile("/mnt/etc/apk/repositories.install", []byte(installRepos), 0644)
	defer os.Remove("/mnt/etc/apk/repositories.install")

	args := append([]string{
		"-p", "/mnt",
		"--repositories-file", "/mnt/etc/apk/repositories.install",
		"add", "--initdb", "--quiet",
	}, bootstrapPackages()...)

	if err := s.run("apk", args...); err != nil {
		return fmt.Errorf("bootstrap apk: %w", err)
	}

	s.msg("Configuring...")

	if err := os.WriteFile("/mnt/etc/hostname", []byte(hostname+"\n"), 0644); err != nil {
		return err
	}

	hosts := fmt.Sprintf("127.0.0.1\tlocalhost localhost.localdomain %s\n::1\tlocalhost localhost.localdomain %s\n", hostname, hostname)
	if err := os.WriteFile("/mnt/etc/hosts", []byte(hosts), 0644); err != nil {
		return err
	}

	tzData, err := os.ReadFile("/usr/share/zoneinfo/" + timezone)
	if err != nil {
		return fmt.Errorf("read timezone: %w", err)
	}

	if err := os.WriteFile("/mnt/etc/localtime", tzData, 0644); err != nil {
		return err
	}

	if err := os.WriteFile("/mnt/etc/timezone", []byte(timezone+"\n"), 0644); err != nil {
		return err
	}

	rootUUID, err := s.deviceUUID("/dev/routier/root")
	if err != nil {
		return err
	}

	efiUUID, err := s.deviceUUID(part1)
	if err != nil {
		return err
	}

	etcUUID, err := s.deviceUUID("/dev/routier/config")
	if err != nil {
		return err
	}

	varUUID, err := s.deviceUUID("/dev/routier/data")
	if err != nil {
		return err
	}

	fstab := fmt.Sprintf("UUID=%s / xfs defaults,noatime 0 0\n", rootUUID)
	fstab += fmt.Sprintf("UUID=%s /boot/efi vfat defaults 0 0\n", efiUUID)
	fstab += fmt.Sprintf("UUID=%s /etc/routier xfs defaults,noatime 0 0\n", etcUUID)
	fstab += fmt.Sprintf("UUID=%s /var/lib/routier xfs defaults,noatime 0 0\n", varUUID)
	if swapMiB > 0 {
		fstab += "/dev/routier/swap none swap defaults 0 0\n"
	}

	if err := os.WriteFile("/mnt/etc/fstab", []byte(fstab), 0644); err != nil {
		return err
	}

	if err := os.MkdirAll("/mnt/etc/mkinitfs", 0755); err != nil {
		return err
	}

	if err := os.WriteFile("/mnt/etc/mkinitfs/mkinitfs.conf",
		[]byte(`features="ata base keymap lvm nvme scsi usb virtio xfs"`+"\n"), 0644); err != nil {
		return err
	}

	inittab := "# /etc/inittab\n\n" +
		"::sysinit:/sbin/openrc sysinit\n" +
		"::sysinit:/sbin/openrc boot\n" +
		"::wait:/sbin/openrc default\n\n" +
		"tty1::respawn:/sbin/getty 38400 tty1\n" +
		"tty2::respawn:/sbin/getty 38400 tty2\n" +
		"tty3::respawn:/sbin/getty 38400 tty3\n" +
		"tty4::respawn:/sbin/getty 38400 tty4\n" +
		"tty5::respawn:/sbin/getty 38400 tty5\n" +
		"tty6::respawn:/sbin/getty 38400 tty6\n\n" +
		"::ctrlaltdel:/sbin/reboot\n\n" +
		"::shutdown:/sbin/openrc shutdown\n"
	if err := os.WriteFile("/mnt/etc/inittab", []byte(inittab), 0644); err != nil {
		return fmt.Errorf("write inittab: %w", err)
	}

	for _, rl := range []string{"sysinit", "boot", "default", "shutdown"} {
		os.MkdirAll("/mnt/etc/runlevels/"+rl, 0755)
	}

	for rl, svcs := range map[string][]string{
		"sysinit":  {"devfs", "dmesg", "mdev", "hwdrivers"},
		"boot":     {"modules", "sysctl", "hostname", "bootmisc", "syslog", "lvm", "routier-net"},
		"default":  {"routier-firstboot", "routier", "routier-ui", "sshd", "cronie"},
		"shutdown": {"mount-ro", "killprocs", "savecache"},
	} {
		for _, svc := range svcs {
			link := "/mnt/etc/runlevels/" + rl + "/" + svc
			os.Remove(link)
			if err := os.Symlink("/etc/init.d/"+svc, link); err != nil {
				return fmt.Errorf("symlink %s: %w", link, err)
			}
		}
	}

	s.msg("Writing routier configuration...")
	cfg, err := config.Default()
	if err != nil {
		return fmt.Errorf("load default config: %w", err)
	}

	cfg.Hostname = hostname
	cfg.Interfaces = ifaces
	if gateway != "" {
		if cfg.Routing == nil {
			cfg.Routing = &config.Routing{}
		}

		cfg.Routing.Static = append(cfg.Routing.Static, config.StaticRoute{Destination: "0.0.0.0/0", Via: gateway})
	}

	cfgData, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile("/mnt/etc/routier/config.yml", cfgData, 0640); err != nil {
		return fmt.Errorf("write config.yml: %w", err)
	}

	fmt.Fprintln(s.tty, "  config.yml")

	for _, f := range []copyEntry{
		{"/etc/motd", "/mnt/etc/motd", 0644},
		{"/etc/issue", "/mnt/etc/issue", 0644},
	} {
		if data, err := os.ReadFile(f.src); err == nil {
			if os.WriteFile(f.dst, data, f.mode) == nil {
				fmt.Fprintln(s.tty, " ", f.src)
			}
		}
	}

	s.msg("Creating routier user...")
	_ = s.run("chroot", "/mnt", "sh", "-c",
		"id routier >/dev/null 2>&1 || adduser -D -s /bin/ash -h /home/routier routier")

	setPass := exec.Command("chroot", "/mnt", "chpasswd")
	setPass.Stdin = strings.NewReader("routier:" + firstbootPass + "\n")
	setPass.Stdout = s.tty
	setPass.Stderr = s.tty
	if err := setPass.Run(); err != nil {
		return fmt.Errorf("set routier password: %w", err)
	}

	if err := os.WriteFile("/mnt/var/lib/routier/ui-seed-password", []byte(firstbootPass+"\n"), 0600); err != nil {
		return fmt.Errorf("write ui-seed-password: %w", err)
	}

	s.msg("Building initramfs...")
	if err := s.run("mount", "--bind", "/proc", "/mnt/proc"); err != nil {
		return fmt.Errorf("bind /proc: %w", err)
	}

	modDirs, _ := os.ReadDir("/mnt/lib/modules")
	var kvers []string
	for _, e := range modDirs {
		if e.IsDir() {
			kvers = append(kvers, e.Name())
		}
	}

	sort.Strings(kvers)
	if len(kvers) == 0 {
		return fmt.Errorf("no kernel version found in /mnt/lib/modules")
	}

	kver := kvers[len(kvers)-1]

	flavor := kver[strings.LastIndex(kver, "-")+1:]
	_ = os.Symlink("vmlinuz-"+flavor, "/mnt/boot/vmlinuz")

	if err := s.run("chroot", "/mnt", "mkinitfs", kver); err != nil {
		_ = exec.Command("umount", "/mnt/proc").Run()
		return fmt.Errorf("mkinitfs: %w", err)
	}

	os.Remove("/mnt/boot/vmlinuz")
	_ = exec.Command("umount", "/mnt/proc").Run()

	s.msg("Installing GRUB...")

	grubTarget := map[string]string{"aarch64": "arm64-efi", "x86_64": "x86_64-efi"}[arch]

	for _, bind := range []string{"/dev", "/proc", "/sys"} {
		if err := s.run("mount", "--bind", bind, "/mnt"+bind); err != nil {
			return fmt.Errorf("bind %s: %w", bind, err)
		}
	}

	if s.run("mount", "-t", "efivarfs", "efivarfs", "/mnt/sys/firmware/efi/efivars") != nil {
		_ = s.run("mount", "--bind", "/sys/firmware/efi/efivars", "/mnt/sys/firmware/efi/efivars")
	}

	if err := s.run("chroot", "/mnt", "grub-install",
		"--target="+grubTarget,
		"--efi-directory=/boot/efi",
		"--boot-directory=/boot",
		"--removable",
	); err != nil {
		return fmt.Errorf("grub-install: %w", err)
	}

	if err := os.MkdirAll("/mnt/boot/grub", 0755); err != nil {
		return err
	}

	if err := os.WriteFile("/mnt/boot/grub/grub.cfg", []byte(boot.GrubConfig(arch, flavor)), 0644); err != nil {
		return err
	}

	s.msg("Checking for an internet connection...")
	if err := s.run("apk", "-p", "/mnt", "update"); err != nil {
		fmt.Fprintln(s.tty, "  No connection; keeping the packages shipped on the media.")
	} else if s.confirm("Internet detected. Upgrade the installed packages to the latest now?") {
		s.msg("Upgrading packages...")
		if err := s.run("apk", "-p", "/mnt", "upgrade", "--available"); err != nil {
			fmt.Fprintf(s.tty, "  upgrade failed: %v\n", err)
		}
	}

	_ = s.run("chroot", "/mnt", "passwd", "-l", "root")

	s.msg("Unmounting...")
	for _, m := range []string{
		"/mnt/sys/firmware/efi/efivars", "/mnt/sys", "/mnt/proc", "/mnt/dev",
		"/mnt/etc/routier", "/mnt/var/lib/routier",
		"/mnt/boot/efi", "/mnt",
	} {
		_ = exec.Command("umount", m).Run()
	}

	if swapMiB > 0 {
		_ = exec.Command("swapoff", "/dev/routier/swap").Run()
	}

	_ = exec.Command("vgchange", "-an", "routier").Run()

	s.msg("Installation complete. Remove the installer media and reboot.")
	if s.confirm("Reboot now?") {
		_ = syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	}

	return nil
}

func swapSizeMiB() int64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}

	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}

		kb, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0
		}

		half := kb / 2 / 1024
		if half > maxSwapMiB {
			return maxSwapMiB
		}

		return half
	}

	return 0
}

func waitForDevice(path string, attempts int) error {
	for i := 0; i < attempts; i++ {
		if _, err := os.Stat(path); err == nil {
			return nil
		}

		_ = exec.Command("mdev", "-s").Run()
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timed out after %d attempts", attempts)
}

func bootRepository(arch string) string {
	for _, root := range []string{"/media", "/run/media"} {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}

		for _, e := range entries {
			repo := filepath.Join(root, e.Name(), "apks")
			if _, err := os.Stat(filepath.Join(repo, arch, "APKINDEX.tar.gz")); err == nil {
				return repo
			}
		}
	}

	return ""
}

func firstSDDisk() string {
	entries, _ := os.ReadDir("/sys/class/block")
	var disks []string
	for _, e := range entries {
		name := e.Name()

		if _, err := os.Stat("/sys/class/block/" + name + "/partition"); err == nil {
			continue
		}

		if !isWholeDisk(name) || isRemovableDisk(name) {
			continue
		}

		disks = append(disks, name)
	}

	if len(disks) == 0 {
		return "/dev/vda"
	}

	sort.Strings(disks)
	return "/dev/" + disks[0]
}

func isWholeDisk(name string) bool {
	for _, prefix := range []string{"sd", "vd", "nvme", "mmcblk"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

func isRemovableDisk(name string) bool {
	data, _ := os.ReadFile("/sys/class/block/" + name + "/removable")
	return strings.TrimSpace(string(data)) == "1"
}
