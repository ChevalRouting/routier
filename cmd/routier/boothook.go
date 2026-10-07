package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChevalRouting/routier/pkg/host/boot"
	"github.com/spf13/cobra"
)

const kernelStateFile = "/run/routier/kernel.pre"

func newBootHookCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "boot-hook <pre-commit|post-commit>",
		Short:  "apk commit hook: keep the bootloader in sync with the installed kernel",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE:   runBootHook,
	}
}

func runBootHook(_ *cobra.Command, args []string) error {
	switch args[0] {
	case "pre-commit":
		return bootHookPre()
	case "post-commit":
		return bootHookPost()
	default:
		return nil
	}
}

func bootHookPre() error {
	if err := os.MkdirAll(filepath.Dir(kernelStateFile), 0755); err != nil {
		return nil
	}

	_ = os.WriteFile(kernelStateFile, []byte(kernelFingerprint()), 0644)
	return nil
}

func bootHookPost() error {
	before, err := os.ReadFile(kernelStateFile)
	_ = os.Remove(kernelStateFile)

	after := kernelFingerprint()
	if err == nil && strings.TrimSpace(string(before)) == after {
		return nil
	}

	if boot.IsLive() {
		warn("kernel changed on the live image; this does not persist across reboots. Install to disk with setup-routier or rebuild the ISO to keep it.")
		return nil
	}

	chosen, note, rerr := boot.Reconcile("/", boot.HostArch())
	if rerr != nil {
		warn("could not update the bootloader: %v. Verify /boot before rebooting.", rerr)
		return nil
	}

	if note != "" {
		warn("%s", note)
	}

	_, _ = fmt.Fprintf(os.Stderr, "routier: bootloader set to boot linux-%s (%s)\n", chosen.Flavor, chosen.Version)
	return nil
}

func kernelFingerprint() string {
	kernels, err := boot.DetectKernels("/")
	if err != nil {
		return ""
	}

	parts := make([]string, 0, len(kernels))
	for _, k := range kernels {
		parts = append(parts, k.Flavor+":"+k.Version)
	}

	return strings.Join(parts, ",")
}

func warn(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "routier: "+format+"\n", args...)
}
