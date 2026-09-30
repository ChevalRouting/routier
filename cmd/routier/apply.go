package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/diffutil"
	"github.com/ChevalRouting/routier/pkg/state/failures"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/host/motd"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newApplyCommand() *cobra.Command {
	var (
		timeout      int
		noArm        bool
		noDiff       bool
		source       string
		friendsCache string
	)
	cmd := &cobra.Command{
		Use:   "apply [config]",
		Short: "apply config with rollback safety",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := contextFromContext(cmd)
			if err != nil {
				return err
			}

			log := c.Logger

			cfg, err := loadInterpolatedConfig(configArg(args), friendsCache)
			if err != nil {
				return err
			}

			if !noDiff {
				printApplyDiff(cfg)
			}

			armTimeout := 0
			if !noArm {
				armTimeout = timeout
			}

			res, err := managers.Apply(cmd.Context(), cfg, friends.LoadCacheVars(friendsCache), managers.ApplyOptions{
				Source:     source,
				ConfigPath: configArg(args),
			}, armTimeout)
			if err != nil {
				printArtifactErrors(err)
				return err
			}

			if res.Warning != "" {
				log.Warn().Msg(res.Warning)
			}

			if res.Armed {
				log.Info().Int("timeout", timeout).Msg("confirm pending: routier confirm")
			}

			motd.Write(cmd.Context(), "", "")
			log.Info().Msg("applied")
			return nil
		},
	}

	cmd.Flags().IntVar(&timeout, "timeout", 60, "rollback timeout seconds")
	cmd.Flags().BoolVar(&noArm, "no-confirm", false, "skip rollback timer")
	cmd.Flags().BoolVar(&noDiff, "no-diff", false, "skip the config + rendered-file diff preview")
	cmd.Flags().StringVar(&source, "source", "cli", "log-history source attribution")
	cmd.Flags().StringVar(&friendsCache, "friends-cache", defaultFriendsCache, "path to the friend interpolation cache")
	_ = cmd.Flags().MarkHidden("source")
	return cmd
}

func printArtifactErrors(err error) {
	var ve *failures.ValidationError
	if !errors.As(err, &ve) {
		return
	}

	fmt.Fprintln(os.Stderr, "rendered artifact validation failed:")
	for _, e := range ve.Errors {
		loc := e.Dest
		if e.Line > 0 {
			loc = fmt.Sprintf("%s:%d", e.Dest, e.Line)
		}

		fmt.Fprintf(os.Stderr, "  %s\n", strings.TrimPrefix(loc+": "+e.Message, ": "))
	}

	if ve.BundleID != "" {
		fmt.Fprintf(os.Stderr, "rendered output saved: routier failures show %s\n", ve.BundleID)
	}
}

const defaultFriendsCache = friends.DefaultCachePath

func friendRenderOpt() render.Option {
	return render.WithFriends(friends.LoadCacheVars(defaultFriendsCache))
}

func loadInterpolatedConfig(path, cachePath string) (*config.Config, error) {
	info, err := os.Stat(path)
	if err == nil && !info.IsDir() {
		raw, rerr := os.ReadFile(path)
		if rerr == nil && friends.HasTemplate(string(raw)) {
			resolved, ierr := friends.Interpolate(string(raw), friends.LoadCacheVars(cachePath))
			if ierr != nil {
				return nil, fmt.Errorf("friend interpolation: %w", ierr)
			}

			cfg, lerr := config.LoadBytes([]byte(resolved))
			if lerr != nil {
				return nil, lerr
			}

			cfg.BaseDir = filepath.Dir(path)
			if errs := config.Validate(cfg, true); len(errs) > 0 {
				msgs := make([]string, len(errs))
				for i, e := range errs {
					msgs[i] = e.Error()
				}

				return nil, fmt.Errorf("validation failed: %s", strings.Join(msgs, "; "))
			}

			return cfg, nil
		}
	}

	return config.LoadAndValidate(path, true)
}

func renderDestMap(cfg *config.Config) map[string]string {
	config.ResolveInterfaces(cfg)

	outputs, err := render.All(cfg, friendRenderOpt())
	if err != nil {
		return map[string]string{}
	}

	m := make(map[string]string, len(outputs))
	for _, o := range outputs {
		m[o.Dest] = o.Content
	}

	return m
}

func printApplyDiff(cfg *config.Config) {
	before, _ := os.ReadFile(managers.LastAppliedPath)
	after, _ := yaml.Marshal(cfg)

	fmt.Println("=== config changes ===")
	printChangedLines(diffutil.Lines(string(before), string(after)))

	beforeFiles := map[string]string{}
	if len(before) > 0 {
		if prev, err := config.LoadBytes(before); err == nil {
			beforeFiles = renderDestMap(prev)
		}
	}

	fmt.Println("\n=== rendered files ===")
	diffs := diffutil.Files(beforeFiles, renderDestMap(cfg))
	if len(diffs) == 0 {
		fmt.Println("  (no changes)")
		fmt.Println()
		return
	}

	for _, fd := range diffs {
		fmt.Printf("  %s  %s\n", statusMark(fd.Status), fd.File)
		printChangedLines(fd.Lines)
	}

	fmt.Println()
}

func statusMark(status string) string {
	switch status {
	case "added":
		return "A"
	case "removed":
		return "R"
	default:
		return "M"
	}
}

func printChangedLines(lines []types.DiffLine) {
	changed := false
	for _, l := range lines {
		switch l.Type {
		case "add":
			fmt.Printf("    + %s\n", l.Text)
			changed = true
		case "remove":
			fmt.Printf("    - %s\n", l.Text)
			changed = true
		}
	}

	if !changed {
		fmt.Println("    (no changes)")
	}
}
