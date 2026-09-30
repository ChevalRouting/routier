package boot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeRoot(t *testing.T, kernels map[string]string, grub bool) string {
	t.Helper()
	root := t.TempDir()

	for ver, flavor := range kernels {
		if err := os.MkdirAll(filepath.Join(root, "lib", "modules", ver), 0755); err != nil {
			t.Fatal(err)
		}

		if flavor == "" {
			continue
		}

		if err := os.MkdirAll(filepath.Join(root, "boot"), 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(root, "boot", "vmlinuz-"+flavor), nil, 0644); err != nil {
			t.Fatal(err)
		}
	}

	if grub {
		if err := os.MkdirAll(filepath.Join(root, "boot", "grub"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func TestDetectKernelsSkipsMissingVmlinuz(t *testing.T) {
	root := makeRoot(t, map[string]string{"6.12.4-0-lts": "lts", "6.6.1-0-virt": ""}, false)

	kernels, err := DetectKernels(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(kernels) != 1 || kernels[0].Flavor != "lts" || kernels[0].Version != "6.12.4-0-lts" {
		t.Fatalf("got %+v", kernels)
	}
}

func TestChooseKernelPrefersRunningFlavor(t *testing.T) {
	kernels := []Kernel{
		{Version: "6.12.4-0-lts", Flavor: "lts"},
		{Version: "6.6.30-0-virt", Flavor: "virt"},
	}

	chosen, warn := chooseKernel(kernels, "virt")
	if chosen.Flavor != "virt" {
		t.Fatalf("chosen = %q, want virt", chosen.Flavor)
	}

	if warn == "" || !strings.Contains(warn, "apk del linux-lts") {
		t.Fatalf("warn = %q", warn)
	}
}

func TestChooseKernelFallsBackToHighest(t *testing.T) {
	kernels := []Kernel{
		{Version: "6.6.30-0-virt", Flavor: "virt"},
		{Version: "6.12.4-0-lts", Flavor: "lts"},
	}

	chosen, warn := chooseKernel(kernels, "")
	if chosen.Flavor != "lts" {
		t.Fatalf("chosen = %q, want lts", chosen.Flavor)
	}

	if warn == "" {
		t.Fatal("want multi-flavor warning")
	}
}

func TestChooseKernelSingleFlavorNoWarn(t *testing.T) {
	kernels := []Kernel{
		{Version: "6.12.3-0-lts", Flavor: "lts"},
		{Version: "6.12.4-0-lts", Flavor: "lts"},
	}

	chosen, warn := chooseKernel(kernels, "lts")
	if chosen.Version != "6.12.4-0-lts" {
		t.Fatalf("chosen = %q, want newest", chosen.Version)
	}

	if warn != "" {
		t.Fatalf("unexpected warn %q", warn)
	}
}

func TestReconcileWritesGrubConfig(t *testing.T) {
	root := makeRoot(t, map[string]string{"6.12.4-0-lts": "lts"}, true)

	chosen, _, err := Reconcile(root, "x86_64")
	if err != nil {
		t.Fatal(err)
	}

	if chosen.Flavor != "lts" {
		t.Fatalf("chosen = %q", chosen.Flavor)
	}

	data, err := os.ReadFile(filepath.Join(root, "boot", "grub", "grub.cfg"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "/boot/vmlinuz-lts") || !strings.Contains(string(data), "ttyS0") {
		t.Fatalf("grub.cfg = %s", data)
	}
}

func TestReconcileNoGrubTreeSkips(t *testing.T) {
	root := makeRoot(t, map[string]string{"6.12.4-0-lts": "lts"}, false)

	if _, _, err := Reconcile(root, "x86_64"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(root, "boot", "grub", "grub.cfg")); !os.IsNotExist(err) {
		t.Fatal("should not have written grub.cfg without a grub tree")
	}
}

func TestReconcileNoKernelErrors(t *testing.T) {
	root := makeRoot(t, map[string]string{}, true)

	if _, _, err := Reconcile(root, "x86_64"); err == nil {
		t.Fatal("want error when no kernel installed")
	}
}

func TestLiveCmdline(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "live")
	disk := filepath.Join(dir, "disk")

	if err := os.WriteFile(live, []byte("modules=loop,squashfs,cdrom quiet\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(disk, []byte("root=/dev/routier/root rootfstype=xfs quiet\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if !liveCmdline(live) {
		t.Fatal("live cmdline not detected")
	}

	if liveCmdline(disk) {
		t.Fatal("disk cmdline misdetected as live")
	}
}
