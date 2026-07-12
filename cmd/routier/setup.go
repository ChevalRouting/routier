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

	"github.com/spf13/cobra"
)

func isLiveISO() bool {
	data, _ := os.ReadFile("/proc/cmdline")
	for _, field := range strings.Fields(string(data)) {
		if strings.HasPrefix(field, "modules=") {
			for _, mod := range strings.Split(strings.TrimPrefix(field, "modules="), ",") {
				if mod == "loop" {
					return true
				}
			}
		}
	}

	return false
}

func newSetupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "interactive disk installer (LVM+ext4, GRUB EFI, reboot)",
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

type copyEntry struct {
	src, dst string
	mode     os.FileMode
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

	hostname := s.ask("Hostname [routier]: ", "routier")

	fmt.Fprintln(s.tty, "Common timezones: UTC, America/New_York, America/Chicago, America/Los_Angeles,")
	fmt.Fprintln(s.tty, "  Europe/London, Europe/Paris, Europe/Berlin, Asia/Tokyo, Asia/Shanghai")
	timezone := s.ask("Timezone [UTC]: ", "UTC")
	if _, err := os.Stat("/usr/share/zoneinfo/" + timezone); err != nil {
		return fmt.Errorf("unknown timezone: %s (check /usr/share/zoneinfo/)", timezone)
	}

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

	s.msg("Creating LVM...")
	if err := s.run("pvcreate", "-ff", "-y", part2); err != nil {
		return fmt.Errorf("pvcreate: %w", err)
	}

	if err := s.run("vgcreate", "routier", part2); err != nil {
		return fmt.Errorf("vgcreate: %w", err)
	}

	szStr, err := s.output("blockdev", "--getsize64", part2)
	if err != nil {
		return fmt.Errorf("blockdev: %w", err)
	}

	sz, _ := strconv.ParseInt(szStr, 10, 64)
	pvMiB := sz / 1024 / 1024

	var swapMiB, rootMiB int64
	if pvMiB > 6144 {
		swapMiB = 2048
		rootMiB = pvMiB - swapMiB - 32
	} else {
		rootMiB = pvMiB - 32
	}

	if err := s.run("lvcreate", "-y", "-L", fmt.Sprintf("%dMiB", rootMiB), "-n", "root", "routier"); err != nil {
		return fmt.Errorf("lvcreate root: %w", err)
	}

	if swapMiB > 0 {
		if err := s.run("lvcreate", "-y", "-L", fmt.Sprintf("%dMiB", swapMiB), "-n", "swap", "routier"); err != nil {
			return fmt.Errorf("lvcreate swap: %w", err)
		}
	}

	s.msg("Formatting filesystems...")
	if err := s.run("mkfs.fat", "-F32", "-n", "EFI", part1); err != nil {
		return fmt.Errorf("mkfs.fat: %w", err)
	}

	if err := s.run("mkfs.ext4", "-q", "-L", "root", "/dev/routier/root"); err != nil {
		return fmt.Errorf("mkfs.ext4: %w", err)
	}

	if swapMiB > 0 {
		if err := s.run("mkswap", "-L", "swap", "/dev/routier/swap"); err != nil {
			return fmt.Errorf("mkswap: %w", err)
		}
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

	if swapMiB > 0 {
		_ = s.run("swapon", "/dev/routier/swap")
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

	reposRaw, _ := os.ReadFile("/etc/apk/repositories")
	var remoteRepos []string
	for _, line := range strings.Split(string(reposRaw), "\n") {
		if l := strings.TrimSpace(line); l != "" && !strings.HasPrefix(l, "#") && !strings.HasPrefix(l, "/") {
			remoteRepos = append(remoteRepos, l)
		}
	}

	_ = os.WriteFile("/mnt/etc/apk/repositories", []byte(strings.Join(remoteRepos, "\n")+"\n"), 0644)
	_ = os.WriteFile("/mnt/etc/apk/repositories.install", reposRaw, 0644)
	defer os.Remove("/mnt/etc/apk/repositories.install")

	arch, err := s.output("apk", "--print-arch")
	if err != nil {
		return fmt.Errorf("apk --print-arch: %w", err)
	}

	if arch != "aarch64" && arch != "x86_64" {
		return fmt.Errorf("unsupported arch: %s", arch)
	}

	if err := s.run("apk",
		"-p", "/mnt",
		"--repositories-file", "/mnt/etc/apk/repositories.install",
		"add", "--initdb", "--quiet",
		"alpine-base", "linux-virt", "tzdata",
		"grub-efi", "efibootmgr",
		"lvm2", "lvm2-openrc", "e2fsprogs", "dosfstools",
		"util-linux-openrc", "doas",
		"openssh", "openssl",
		"htop",
		"routier", "routier-openrc",
	); err != nil {
		return fmt.Errorf("bootstrap apk: %w", err)
	}

	s.msg("Configuring...")

	if err := os.WriteFile("/mnt/etc/hostname", []byte(hostname+"\n"), 0644); err != nil {
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

	rootUUID, err := s.output("blkid", "-s", "UUID", "-o", "value", "/dev/routier/root")
	if err != nil {
		return fmt.Errorf("blkid root: %w", err)
	}

	efiUUID, err := s.output("blkid", "-s", "UUID", "-o", "value", part1)
	if err != nil {
		return fmt.Errorf("blkid efi: %w", err)
	}

	fstab := fmt.Sprintf("UUID=%s\t/\text4\tdefaults,noatime\t0 1\n", rootUUID)
	fstab += fmt.Sprintf("UUID=%s\t/boot/efi\tvfat\tdefaults\t0 2\n", efiUUID)
	if swapMiB > 0 {
		fstab += "/dev/routier/swap\tnone\tswap\tdefaults\t0 0\n"
	}

	if err := os.WriteFile("/mnt/etc/fstab", []byte(fstab), 0644); err != nil {
		return err
	}

	if err := os.MkdirAll("/mnt/etc/mkinitfs", 0755); err != nil {
		return err
	}

	if err := os.WriteFile("/mnt/etc/mkinitfs/mkinitfs.conf",
		[]byte(`features="ata base ext4 keymap lvm nvme scsi usb virtio"`+"\n"), 0644); err != nil {
		return err
	}

	serialDev := map[string]string{"aarch64": "ttyAMA0", "x86_64": "ttyS0"}[arch]

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
		serialDev + "::respawn:/sbin/getty -L 115200 " + serialDev + " vt100\n\n" +
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

	s.msg("Persisting live configuration...")
	if data, err := os.ReadFile("/etc/routier/config.yml"); err == nil {
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "hostname:") {
				lines[i] = "hostname: " + hostname
				break
			}
		}

		os.MkdirAll("/mnt/etc/routier", 0755)
		if os.WriteFile("/mnt/etc/routier/config.yml", []byte(strings.Join(lines, "\n")), 0644) == nil {
			fmt.Fprintln(s.tty, "  config.yml")
		}
	}

	os.MkdirAll("/mnt/var/lib/routier", 0750)
	for _, f := range []copyEntry{
		{"/var/lib/routier/web.db", "/mnt/var/lib/routier/web.db", 0640},
		{"/var/lib/routier/ui-seed-password", "/mnt/var/lib/routier/ui-seed-password", 0600},
		{"/etc/motd", "/mnt/etc/motd", 0644},
		{"/etc/issue", "/mnt/etc/issue", 0644},
	} {
		if data, err := os.ReadFile(f.src); err == nil {
			if os.WriteFile(f.dst, data, f.mode) == nil {
				fmt.Fprintln(s.tty, " ", f.src)
			}
		}
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

	if err := os.WriteFile("/mnt/boot/grub/grub.cfg", []byte(grubCfg(arch)), 0644); err != nil {
		return err
	}

	_ = s.run("chroot", "/mnt", "passwd", "-l", "root")

	s.msg("Unmounting...")
	for _, m := range []string{
		"/mnt/sys/firmware/efi/efivars", "/mnt/sys", "/mnt/proc", "/mnt/dev",
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

func firstSDDisk() string {
	entries, _ := os.ReadDir("/sys/class/block")
	for _, e := range entries {
		name := e.Name()
		if len(name) == 3 && strings.HasPrefix(name, "sd") {
			if _, err := os.Stat("/sys/class/block/" + name + "/partition"); os.IsNotExist(err) {
				return "/dev/" + name
			}
		}
	}

	return "/dev/vda"
}

func grubCfg(arch string) string {
	console := map[string]string{"aarch64": "ttyAMA0", "x86_64": "ttyS0"}[arch]
	return fmt.Sprintf(`insmod part_gpt
insmod fat
insmod lvm
insmod ext2

set default=0
set timeout=3

menuentry "Routier" {
    search --no-floppy --label --set=root root
    linux  /boot/vmlinuz-virt root=/dev/routier/root rootfstype=ext4 console=%s,115200 quiet
    initrd /boot/initramfs-virt
}
`, console)
}
