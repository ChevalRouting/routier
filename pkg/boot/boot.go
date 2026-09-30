package boot

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type Kernel struct {
	Version string
	Flavor  string
}

func IsLive() bool {
	return liveCmdline("/proc/cmdline")
}

func liveCmdline(path string) bool {
	data, _ := os.ReadFile(path)
	for _, field := range strings.Fields(string(data)) {
		if !strings.HasPrefix(field, "modules=") {
			continue
		}

		for _, mod := range strings.Split(strings.TrimPrefix(field, "modules="), ",") {
			if mod == "loop" {
				return true
			}
		}
	}

	return false
}

func HostArch() string {
	switch runtime.GOARCH {
	case "arm64":
		return "aarch64"
	case "amd64":
		return "x86_64"
	default:
		return runtime.GOARCH
	}
}

func RunningRelease() string {
	data, _ := os.ReadFile("/proc/sys/kernel/osrelease")
	return strings.TrimSpace(string(data))
}

func RunningFlavor() string {
	return flavorOf(RunningRelease())
}

func flavorOf(version string) string {
	i := strings.LastIndex(version, "-")
	if i < 0 {
		return ""
	}

	return version[i+1:]
}

func DetectKernels(root string) ([]Kernel, error) {
	entries, err := os.ReadDir(filepath.Join(root, "lib", "modules"))
	if err != nil {
		return nil, fmt.Errorf("read modules: %w", err)
	}

	var kernels []Kernel
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		flavor := flavorOf(e.Name())
		if flavor == "" {
			continue
		}

		if _, err := os.Stat(filepath.Join(root, "boot", "vmlinuz-"+flavor)); err != nil {
			continue
		}

		kernels = append(kernels, Kernel{Version: e.Name(), Flavor: flavor})
	}

	sort.Slice(kernels, func(i, j int) bool { return kernels[i].Version < kernels[j].Version })
	return kernels, nil
}

func chooseKernel(kernels []Kernel, prefer string) (Kernel, string) {
	byFlavor := make(map[string][]Kernel)
	for _, k := range kernels {
		byFlavor[k.Flavor] = append(byFlavor[k.Flavor], k)
	}

	flavor := prefer
	if _, ok := byFlavor[flavor]; !ok {
		flavor = kernels[len(kernels)-1].Flavor
	}

	group := byFlavor[flavor]
	chosen := group[len(group)-1]

	if len(byFlavor) == 1 {
		return chosen, ""
	}

	var others []string
	for f := range byFlavor {
		if f != flavor {
			others = append(others, f)
		}
	}

	sort.Strings(others)
	warn := fmt.Sprintf("multiple kernel flavors installed (keeping %q); Alpine supports one flavor per install, remove the extras with: apk del linux-%s",
		flavor, strings.Join(others, " linux-"))
	return chosen, warn
}

func Reconcile(root, arch string) (Kernel, string, error) {
	kernels, err := DetectKernels(root)
	if err != nil {
		return Kernel{}, "", err
	}

	if len(kernels) == 0 {
		return Kernel{}, "", fmt.Errorf("no installed kernel found under %s", filepath.Join(root, "lib", "modules"))
	}

	chosen, warn := chooseKernel(kernels, RunningFlavor())

	grubDir := filepath.Join(root, "boot", "grub")
	if _, err := os.Stat(grubDir); err != nil {
		return chosen, warn, nil
	}

	cfg := GrubConfig(arch, chosen.Flavor)
	if err := os.WriteFile(filepath.Join(grubDir, "grub.cfg"), []byte(cfg), 0644); err != nil {
		return chosen, warn, fmt.Errorf("write grub.cfg: %w", err)
	}

	return chosen, warn, nil
}

func GrubConfig(arch, flavor string) string {
	console := map[string]string{"aarch64": "ttyAMA0", "x86_64": "ttyS0"}[arch]
	return fmt.Sprintf(`insmod part_gpt
insmod fat
insmod lvm
insmod xfs

set default=0
set timeout=3

menuentry "Routier" {
    search --no-floppy --label --set=root root
    linux  /boot/vmlinuz-%s root=/dev/routier/root rootfstype=xfs rootwait console=%s,115200 quiet
    initrd /boot/initramfs-%s
}
`, flavor, console, flavor)
}
