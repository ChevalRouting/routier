package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/host/boot"
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
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "interactive disk installer (LVM+xfs, GRUB EFI, reboot)",
		RunE:  runSetup,
	}
	cmd.Flags().BoolP("yes", "y", false, "assume yes: wipe the target disk without prompting")
	cmd.Flags().String("disk", "", "install non-interactively to this disk, with DHCP on every interface")
	cmd.Flags().String("hostname", "routier", "hostname of the installed system")
	cmd.Flags().String("timezone", "UTC", "timezone of the installed system")
	cmd.Flags().Bool("password-stdin", false, "read the admin password of the installed system from stdin")
	cmd.Flags().Bool("no-reboot", false, "do not reboot once the installation is complete")

	return cmd
}

type installer struct {
	tty       io.Writer
	scanner   *bufio.Scanner
	assumeYes bool
}

func (s *installer) msg(format string, args ...any) {
	_, _ = fmt.Fprintf(s.tty, "\n\033[1;34m==> %s\033[0m\n", fmt.Sprintf(format, args...))
}

func (s *installer) ask(prompt, def string) string {
	_, _ = fmt.Fprint(s.tty, prompt)
	if s.scanner.Scan() {
		if line := strings.TrimSpace(s.scanner.Text()); line != "" {
			return line
		}
	}

	return def
}

func (s *installer) confirm(prompt string) bool {
	if s.assumeYes {
		return true
	}

	_, _ = fmt.Fprintf(s.tty, "%s [y/N] ", prompt)
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

func bootstrapPackages() []string {
	pkgs := []string{
		"alpine-base", "grub-efi", "efibootmgr",
		"routier", "routier-openrc", "doas", "openssl",
	}

	pkgs = append(pkgs, appliancePackages()...)

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

func appliancePackages() []string {
	if pkgs := worldPackages("/etc/routier/world"); len(pkgs) > 0 {
		return pkgs
	}

	return worldPackages("/etc/apk/world")
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

type setupOptions struct {
	disk     string
	hostname string
	timezone string
	password string
	ifaces   map[string]*config.Interface
	gateway  string
	noReboot bool
}

func runSetup(cmd *cobra.Command, _ []string) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("must be run as root")
	}

	opts, err := setupOptionsFromFlags(cmd)
	if err != nil {
		return err
	}

	if opts.disk != "" {
		s := &installer{tty: os.Stdout, scanner: bufio.NewScanner(strings.NewReader("")), assumeYes: true}
		return s.install(opts)
	}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open /dev/tty: %w", err)
	}

	defer func(action func() error) { _ = action() }(tty.Close)

	assumeYes, _ := cmd.Flags().GetBool("yes")
	s := &installer{tty: tty, scanner: bufio.NewScanner(tty), assumeYes: assumeYes}

	s.msg("Routier installer")

	_, _ = fmt.Fprintln(s.tty, "Common timezones: UTC, America/New_York, America/Chicago, America/Los_Angeles,")
	_, _ = fmt.Fprintln(s.tty, "  Europe/London, Europe/Paris, Europe/Berlin, Asia/Tokyo, Asia/Shanghai")
	opts.timezone = s.ask("Timezone [UTC]: ", "UTC")
	if err := checkTimezone(opts.timezone); err != nil {
		return err
	}

	opts.ifaces, opts.gateway = s.askInterfaces(usableNICs())

	s.msg("Available disks:")
	_ = s.run("lsblk", "-d", "-o", "NAME,SIZE,MODEL")

	defaultDisk := firstSDDisk()
	opts.disk = s.ask(fmt.Sprintf("Disk to install to [%s]: ", defaultDisk), defaultDisk)
	if err := checkDisk(opts.disk); err != nil {
		return err
	}

	diskSize, _ := s.output("lsblk", "-d", "-n", "-o", "SIZE", opts.disk)
	_, _ = fmt.Fprintf(s.tty, "\n\033[1;31mWARNING:\033[0m all data on %s (%s) will be permanently erased.\n", opts.disk, diskSize)
	if !s.confirm("Continue?") {
		return nil
	}

	return s.install(opts)
}

func setupOptionsFromFlags(cmd *cobra.Command) (setupOptions, error) {
	var opts setupOptions
	opts.disk, _ = cmd.Flags().GetString("disk")
	opts.hostname, _ = cmd.Flags().GetString("hostname")
	opts.timezone, _ = cmd.Flags().GetString("timezone")
	opts.noReboot, _ = cmd.Flags().GetBool("no-reboot")

	if fromStdin, _ := cmd.Flags().GetBool("password-stdin"); fromStdin {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return opts, fmt.Errorf("read password from stdin: %w", err)
		}

		opts.password = strings.TrimRight(line, "\r\n")
		if opts.password == "" {
			return opts, fmt.Errorf("empty password on stdin")
		}
	}

	if opts.disk == "" {
		return opts, nil
	}

	if err := checkTimezone(opts.timezone); err != nil {
		return opts, err
	}

	if err := checkDisk(opts.disk); err != nil {
		return opts, err
	}

	opts.ifaces = dhcpInterfaces(usableNICs())
	return opts, nil
}

func checkTimezone(timezone string) error {
	if _, err := os.Stat("/usr/share/zoneinfo/" + timezone); err != nil {
		return fmt.Errorf("unknown timezone: %s (check /usr/share/zoneinfo/)", timezone)
	}

	return nil
}

func checkDisk(disk string) error {
	fi, err := os.Stat(disk)
	if err != nil {
		return fmt.Errorf("disk not found: %s", disk)
	}

	if fi.Mode()&os.ModeDevice == 0 || fi.Mode()&os.ModeCharDevice != 0 {
		return fmt.Errorf("not a block device: %s", disk)
	}

	return nil
}

func dhcpInterfaces(nics []string) map[string]*config.Interface {
	ifaces := make(map[string]*config.Interface, len(nics))
	for _, nic := range nics {
		ifaces[nic] = &config.Interface{Select: nic, Addresses: []string{"dhcp"}}
	}

	return ifaces
}

func (s *installer) install(opts setupOptions) error {
	hostname := opts.hostname
	if hostname == "" {
		hostname = "routier"
	}

	layout, err := s.partition(opts.disk)
	if err != nil {
		return err
	}

	swapMiB := swapSizeMiB()
	if err := s.createVolumes(layout, swapMiB); err != nil {
		return err
	}

	if err := s.formatVolumes(layout, swapMiB); err != nil {
		return err
	}

	if err := s.mountTarget(layout); err != nil {
		return err
	}

	arch, err := s.installPackages()
	if err != nil {
		return err
	}

	s.msg("Configuring...")
	if err := writeSystemIdentity(hostname, opts.timezone); err != nil {
		return err
	}

	if err := s.writeFstab(layout, swapMiB); err != nil {
		return err
	}

	if err := writeInitSystem(); err != nil {
		return err
	}

	if err := s.writeRoutierConfig(opts, hostname); err != nil {
		return err
	}

	if err := s.createRoutierUser(opts); err != nil {
		return err
	}

	flavor, err := s.buildInitramfs()
	if err != nil {
		return err
	}

	if err := s.installBootloader(arch, flavor); err != nil {
		return err
	}

	s.offerUpgrade()
	s.finish(opts, swapMiB)
	return nil
}

type diskLayout struct {
	efi string
	lvm string
}

func (s *installer) partition(disk string) (diskLayout, error) {
	s.msg("Wiping %s...", disk)
	if err := s.wipeDisk(disk); err != nil {
		return diskLayout{}, fmt.Errorf("wipe disk: %w", err)
	}

	s.msg("Partitioning %s...", disk)
	if err := s.run("parted", "-s", disk,
		"mklabel", "gpt",
		"mkpart", "ESP", "fat32", "1MiB", "513MiB",
		"set", "1", "esp", "on",
		"mkpart", "LVM", "513MiB", "100%",
	); err != nil {
		return diskLayout{}, fmt.Errorf("parted: %w", err)
	}

	_ = s.run("partprobe", disk)
	_ = exec.Command("mdev", "-s").Run()

	layout := diskLayout{efi: disk + "1", lvm: disk + "2"}
	if strings.Contains(disk, "nvme") || strings.Contains(disk, "mmcblk") {
		layout = diskLayout{efi: disk + "p1", lvm: disk + "p2"}
	}

	if err := waitForDevice(layout.efi, 10); err != nil {
		return layout, fmt.Errorf("partition %s did not appear: %w", layout.efi, err)
	}

	return layout, nil
}

func (s *installer) createVolumes(layout diskLayout, swapMiB int64) error {
	s.msg("Creating LVM (etc-routier %s, swap %dMiB, var-routier %s, root remainder)...", etcRoutierSize, swapMiB, varLibRoutierSize)
	if err := s.run("pvcreate", "-ff", "-y", layout.lvm); err != nil {
		return fmt.Errorf("pvcreate: %w", err)
	}

	if err := s.run("vgcreate", "routier", layout.lvm); err != nil {
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

	return nil
}

func (s *installer) formatVolumes(layout diskLayout, swapMiB int64) error {
	s.msg("Formatting filesystems...")
	if err := s.run("mkfs.fat", "-F32", "-n", "EFI", layout.efi); err != nil {
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

	return nil
}

func (s *installer) mountTarget(layout diskLayout) error {
	s.msg("Mounting target...")
	if err := s.run("mount", "/dev/routier/root", "/mnt"); err != nil {
		return fmt.Errorf("mount root: %w", err)
	}

	if err := os.MkdirAll("/mnt/boot/efi", 0755); err != nil {
		return err
	}

	if err := s.run("mount", layout.efi, "/mnt/boot/efi"); err != nil {
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

	return nil
}

func (s *installer) installPackages() (string, error) {
	s.msg("Installing Alpine + Routier...")
	if err := os.MkdirAll("/mnt/etc/apk/keys", 0755); err != nil {
		return "", err
	}

	if keyEntries, err := os.ReadDir("/etc/apk/keys"); err == nil {
		for _, e := range keyEntries {
			data, _ := os.ReadFile(filepath.Join("/etc/apk/keys", e.Name()))
			_ = os.WriteFile(filepath.Join("/mnt/etc/apk/keys", e.Name()), data, 0644)
		}
	}

	arch, err := s.output("apk", "--print-arch")
	if err != nil {
		return "", fmt.Errorf("apk --print-arch: %w", err)
	}

	if arch != "aarch64" && arch != "x86_64" {
		return "", fmt.Errorf("unsupported arch: %s", arch)
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
	defer func(path string) { _ = os.Remove(path) }("/mnt/etc/apk/repositories.install")

	args := append([]string{
		"-p", "/mnt",
		"--repositories-file", "/mnt/etc/apk/repositories.install",
		"add", "--initdb", "--quiet",
	}, bootstrapPackages()...)

	if err := s.run("apk", args...); err != nil {
		return "", fmt.Errorf("bootstrap apk: %w", err)
	}

	return arch, nil
}

func writeSystemIdentity(hostname, timezone string) error {
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

	return nil
}

func (s *installer) writeFstab(layout diskLayout, swapMiB int64) error {
	rootUUID, err := s.deviceUUID("/dev/routier/root")
	if err != nil {
		return err
	}

	efiUUID, err := s.deviceUUID(layout.efi)
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

	return os.WriteFile("/mnt/etc/fstab", []byte(fstab), 0644)
}

func writeInitSystem() error {
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
		_ = os.MkdirAll("/mnt/etc/runlevels/"+rl, 0755)
	}

	for rl, svcs := range map[string][]string{
		"sysinit":  {"devfs", "dmesg", "mdev", "hwdrivers"},
		"boot":     {"modules", "sysctl", "hostname", "bootmisc", "syslog", "lvm", "routier-net"},
		"default":  {"routier-firstboot", "routier", "routier-ui", "sshd", "cronie"},
		"shutdown": {"mount-ro", "killprocs", "savecache"},
	} {
		for _, svc := range svcs {
			link := "/mnt/etc/runlevels/" + rl + "/" + svc
			_ = os.Remove(link)
			if err := os.Symlink("/etc/init.d/"+svc, link); err != nil {
				return fmt.Errorf("symlink %s: %w", link, err)
			}
		}
	}

	return nil
}

func (s *installer) writeRoutierConfig(opts setupOptions, hostname string) error {
	s.msg("Writing routier configuration...")
	cfg, err := config.Default()
	if err != nil {
		return fmt.Errorf("load default config: %w", err)
	}

	cfg.Hostname = hostname
	cfg.Interfaces = opts.ifaces
	if opts.gateway != "" {
		if cfg.Routing == nil {
			cfg.Routing = &config.Routing{}
		}

		cfg.Routing.Static = append(cfg.Routing.Static, config.StaticRoute{Destination: "0.0.0.0/0", Via: opts.gateway})
	}

	cfgData, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile("/mnt/etc/routier/config.yml", cfgData, 0640); err != nil {
		return fmt.Errorf("write config.yml: %w", err)
	}

	_, _ = fmt.Fprintln(s.tty, "  config.yml")

	for _, f := range []copyEntry{
		{"/etc/motd", "/mnt/etc/motd", 0644},
		{"/etc/issue", "/mnt/etc/issue", 0644},
	} {
		if data, err := os.ReadFile(f.src); err == nil {
			if os.WriteFile(f.dst, data, f.mode) == nil {
				_, _ = fmt.Fprintln(s.tty, " ", f.src)
			}
		}
	}

	return nil
}

func (s *installer) createRoutierUser(opts setupOptions) error {
	s.msg("Creating routier user...")
	_ = s.run("chroot", "/mnt", "sh", "-c",
		"id routier >/dev/null 2>&1 || adduser -D -s /bin/ash -h /home/routier routier")

	password := opts.password
	if password == "" {
		password = firstbootPass
	}

	setPass := exec.Command("chroot", "/mnt", "chpasswd")
	setPass.Stdin = strings.NewReader("routier:" + password + "\n")
	setPass.Stdout = s.tty
	setPass.Stderr = s.tty
	if err := setPass.Run(); err != nil {
		return fmt.Errorf("set routier password: %w", err)
	}

	if opts.password == "" {
		if err := os.WriteFile("/mnt/var/lib/routier/ui-seed-password", []byte(firstbootPass+"\n"), 0600); err != nil {
			return fmt.Errorf("write ui-seed-password: %w", err)
		}
	}

	return nil
}

func (s *installer) buildInitramfs() (string, error) {
	s.msg("Building initramfs...")
	if err := s.run("mount", "--bind", "/proc", "/mnt/proc"); err != nil {
		return "", fmt.Errorf("bind /proc: %w", err)
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
		return "", fmt.Errorf("no kernel version found in /mnt/lib/modules")
	}

	kver := kvers[len(kvers)-1]

	flavor := kver[strings.LastIndex(kver, "-")+1:]
	_ = os.Symlink("vmlinuz-"+flavor, "/mnt/boot/vmlinuz")

	if err := s.run("chroot", "/mnt", "mkinitfs", kver); err != nil {
		_ = exec.Command("umount", "/mnt/proc").Run()
		return "", fmt.Errorf("mkinitfs: %w", err)
	}

	_ = os.Remove("/mnt/boot/vmlinuz")
	_ = exec.Command("umount", "/mnt/proc").Run()

	return flavor, nil
}

func (s *installer) installBootloader(arch, flavor string) error {
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

	if err := os.WriteFile("/mnt/boot/grub/grub.cfg", []byte(boot.GrubConfig(arch, flavor, boot.DefaultRootArgs)), 0644); err != nil {
		return err
	}

	return nil
}

func (s *installer) offerUpgrade() {
	s.msg("Checking for an internet connection...")
	if err := s.run("apk", "-p", "/mnt", "update"); err != nil {
		_, _ = fmt.Fprintln(s.tty, "  No connection; keeping the packages shipped on the media.")
	} else if s.confirm("Internet detected. Upgrade the installed packages to the latest now?") {
		s.msg("Upgrading packages...")
		if err := s.run("apk", "-p", "/mnt", "upgrade", "--available"); err != nil {
			_, _ = fmt.Fprintf(s.tty, "  upgrade failed: %v\n", err)
		}
	}
}

func (s *installer) finish(opts setupOptions, swapMiB int64) {
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
	if !opts.noReboot && s.confirm("Reboot now?") {
		_ = syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	}
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

func (s *installer) wipeDisk(disk string) error {
	_ = s.run("swapoff", "-a")

	mounts := mountpointsUnder(disk)
	sort.Slice(mounts, func(i, j int) bool { return len(mounts[i]) > len(mounts[j]) })
	for _, mp := range mounts {
		_ = s.run("umount", "-R", "-f", mp)
	}

	for _, vg := range volumeGroupsOn(disk) {
		_ = s.run("vgchange", "-an", vg)
		_ = s.run("vgremove", "-f", "-y", vg)
	}

	for _, part := range devicePartitions(disk) {
		_ = s.run("pvremove", "-ff", "-y", part)
		_ = s.run("wipefs", "-af", part)
	}

	_ = exec.Command("dmsetup", "remove_all").Run()

	if err := s.run("wipefs", "-af", disk); err != nil {
		return fmt.Errorf("wipefs: %w", err)
	}

	_ = s.run("partprobe", disk)
	_ = exec.Command("mdev", "-s").Run()

	return nil
}

func mountpointsUnder(disk string) []string {
	out, err := exec.Command("lsblk", "-nro", "MOUNTPOINT", disk).Output()
	if err != nil {
		return nil
	}

	var mps []string
	seen := make(map[string]struct{})
	for _, line := range strings.Split(string(out), "\n") {
		mp := strings.TrimSpace(line)
		if mp == "" || mp == "[SWAP]" {
			continue
		}

		if _, ok := seen[mp]; ok {
			continue
		}

		seen[mp] = struct{}{}
		mps = append(mps, mp)
	}

	return mps
}

func devicePartitions(disk string) []string {
	out, err := exec.Command("lsblk", "-nro", "NAME,TYPE", disk).Output()
	if err != nil {
		return nil
	}

	var parts []string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "part" {
			parts = append(parts, "/dev/"+fields[0])
		}
	}

	return parts
}

func volumeGroupsOn(disk string) []string {
	out, err := exec.Command("pvs", "--noheadings", "-o", "pv_name,vg_name").Output()
	if err != nil {
		return nil
	}

	var vgs []string
	seen := make(map[string]struct{})
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		if !strings.HasPrefix(fields[0], disk) {
			continue
		}

		vg := fields[1]
		if _, ok := seen[vg]; ok {
			continue
		}

		seen[vg] = struct{}{}
		vgs = append(vgs, vg)
	}

	return vgs
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
