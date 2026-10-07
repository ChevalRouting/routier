package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newEditCommand() *cobra.Command {
	var (
		timeout int
		noArm   bool
		noApply bool
	)
	cmd := &cobra.Command{
		Use:   "edit [config]",
		Short: "edit config in $EDITOR and apply on exit",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return newEditCommandCallback(timeout, noArm, noApply, cmd, args)
		},
	}

	cmd.Flags().IntVar(&timeout, "timeout", 60, "rollback timeout seconds")
	cmd.Flags().BoolVar(&noArm, "no-confirm", false, "skip rollback timer")
	cmd.Flags().BoolVar(&noApply, "no-apply", false, "validate only, do not apply")
	return cmd
}

func newEditCommandCallback(timeout int, noArm bool, noApply bool, cmd *cobra.Command, args []string) error {
	cfgPath := configArg(args)

	src, err := os.ReadFile(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read config: %w", err)
	}

	tmp, err := os.CreateTemp("", "routier-edit-*.yml")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := tmp.Name()
	defer func(path string) { _ = os.Remove(path) }(tmpPath)

	if _, err := tmp.Write(src); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}

	_ = tmp.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	parts := strings.Fields(editor)
	editorArgs := append(parts[1:], tmpPath)
	editorCmd := exec.CommandContext(cmd.Context(), parts[0], editorArgs...)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor exited: %w", err)
	}

	cfg, err := config.LoadAndValidate(tmpPath, true)
	if err != nil {
		return fmt.Errorf("validation failed (config not saved): %w", err)
	}

	if noApply {
		log.Info().Str("path", cfgPath).Msg("validated, skipping apply (--no-apply)")
		return nil
	}

	edited, err := os.ReadFile(tmpPath)
	if err != nil {
		return err
	}

	if err := os.WriteFile(cfgPath, edited, 0o644); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	armTimeout := 0
	if !noArm {
		armTimeout = timeout
	}

	res, err := managers.Apply(cmd.Context(), cfg, friends.LoadCacheVars(defaultFriendsCache), managers.ApplyOptions{}, armTimeout)
	if err != nil {
		return err
	}

	if res.Warning != "" {
		log.Warn().Msg(res.Warning)
	}

	if res.Armed {
		log.Info().Int("timeout", timeout).Msg("confirm pending: routier confirm")
	}

	log.Info().Msg("applied")
	return nil
}
