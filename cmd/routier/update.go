package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/host/boot"
	"github.com/ChevalRouting/routier/pkg/host/updates"
	"github.com/spf13/cobra"
)

func newUpdateCommand() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "refresh package indexes and list available upgrades",
		RunE: func(cmd *cobra.Command, unusedArg2 []string) error {
			return newUpdateCommandCallback(asJSON, cmd, unusedArg2)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "output machine-readable JSON")
	return cmd
}

func newUpgradeCommand() *cobra.Command {
	var yes, reboot bool
	var logPath string
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "upgrade all packages via apk, syncing the bootloader and guarding the live image",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpgrade(cmd, yes, reboot, logPath)
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "do not prompt for confirmation")
	cmd.Flags().BoolVar(&reboot, "reboot", false, "reboot afterwards if the kernel changed")
	cmd.Flags().StringVar(&logPath, "log", updates.LogPath, "tee combined output to this file")
	return cmd
}

func runUpgrade(cmd *cobra.Command, yes, reboot bool, logPath string) error {
	ctx := cmd.Context()

	if boot.IsLive() {
		return fmt.Errorf("cannot upgrade the live image: changes do not persist. Install to disk with setup-routier or rebuild the ISO")
	}

	if !yes && !confirmUpgrade(cmd) {
		return nil
	}

	out := cmd.OutOrStdout()
	if logPath != "" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0755); err == nil {
			if f, ferr := os.Create(logPath); ferr == nil {
				defer func(action func() error) { _ = action() }(f.Close)
				out = io.MultiWriter(out, f)
			}
		}
	}

	avail, _ := updates.Check(ctx)
	started := time.Now()
	_ = updates.WriteStatus(updates.JobStatus{
		State:      updates.StateRunning,
		StartedAt:  started,
		SelfUpdate: avail.SelfUpdate,
	})

	upErr := updates.Upgrade(ctx, out)

	status := updates.JobStatus{
		State:          updates.StateDone,
		StartedAt:      started,
		FinishedAt:     time.Now(),
		RebootRequired: updates.RebootPending(),
		SelfUpdate:     avail.SelfUpdate,
	}
	if upErr != nil {
		status.State = updates.StateFailed
		status.Error = upErr.Error()
	}

	_ = updates.WriteStatus(status)

	if upErr != nil {
		return upErr
	}

	_, _ = fmt.Fprintln(out, "\nupgrade complete.")
	if status.RebootRequired {
		_, _ = fmt.Fprintln(out, "a reboot is required to activate the new kernel.")
		if reboot {
			_, _ = fmt.Fprintln(out, "rebooting...")
			return exec.Command("reboot").Run()
		}
	}

	return nil
}

func printAvailable(w io.Writer, avail updates.Available) {
	if avail.Live {
		_, _ = fmt.Fprintln(w, "note: running from the live image, upgrades will not persist across reboots.")
	}

	if len(avail.Packages) == 0 {
		_, _ = fmt.Fprintln(w, "system is up to date.")
		return
	}

	_, _ = fmt.Fprintf(w, "%d package(s) can be upgraded:\n", len(avail.Packages))
	for _, p := range avail.Packages {
		_, _ = fmt.Fprintf(w, "  %s  %s -> %s\n", p.Name, p.Old, p.New)
	}

	if avail.SelfUpdate {
		_, _ = fmt.Fprintln(w, "the web UI will restart during this upgrade.")
	}

	if avail.RebootRequired {
		_, _ = fmt.Fprintln(w, "a reboot will be required to activate the new kernel.")
	}
}

func confirmUpgrade(cmd *cobra.Command) bool {
	_, _ = fmt.Fprint(cmd.OutOrStdout(), "Upgrade all packages now? [y/N] ")
	line, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

func newUpdateCommandCallback(asJSON bool, cmd *cobra.Command, _ []string) error {
	avail, err := updates.Check(cmd.Context())
	if err != nil {
		return err
	}

	if asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(avail)
	}

	printAvailable(cmd.OutOrStdout(), avail)
	return nil
}
