package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/diffutil"
	"github.com/ChevalRouting/routier/pkg/net/netlink"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type diffChange struct {
	Kind   string           `json:"kind"`
	Target string           `json:"target"`
	Detail string           `json:"detail,omitempty"`
	Diff   []types.DiffLine `json:"diff,omitempty"`
}

func newDiffCommand() *cobra.Command {
	var asJSON bool
	var exitCode bool
	cmd := &cobra.Command{
		Use:   "diff [config]",
		Short: "show what would change to converge the live system to the config",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(unusedArg2 *cobra.Command, args []string) error {
			return newDiffCommandCallback(asJSON, exitCode, unusedArg2, args)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "exit non-zero when the live system is not in sync")
	return cmd
}

func computeDiff(cfg *config.Config) ([]diffChange, error) {
	var changes []diffChange

	outputs, err := render.All(cfg, friendRenderOpt())
	if err != nil {
		return nil, err
	}

	for _, o := range outputs {
		onDisk, err := os.ReadFile(o.Dest)
		if os.IsNotExist(err) {
			changes = append(changes, diffChange{
				Kind: "file", Target: o.Dest, Detail: "create",
				Diff: diffutil.Lines("", normalize(o.Content)),
			})
			continue
		}

		if err != nil {
			return nil, err
		}

		before, after := normalize(string(onDisk)), normalize(o.Content)
		if before != after {
			changes = append(changes, diffChange{
				Kind: "file", Target: o.Dest, Detail: "modify",
				Diff: diffutil.Lines(before, after),
			})
		}
	}

	changes = append(changes, netlinkChanges(cfg)...)
	sort.Slice(changes, func(i, j int) bool { return computeDiffCallback(changes, i, j) })
	return changes, nil
}

func netlinkChanges(cfg *config.Config) []diffChange {
	var buf bytes.Buffer
	prev := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = prev }()

	_ = netlink.Reconcile(cfg, true)

	var changes []diffChange
	dec := json.NewDecoder(&buf)
	for {
		var rec map[string]any
		if err := dec.Decode(&rec); err != nil {
			break
		}

		msg, _ := rec["message"].(string)
		if !strings.HasPrefix(msg, "would ") {
			continue
		}

		changes = append(changes, diffChange{
			Kind:   "netlink",
			Target: recordTarget(rec),
			Detail: strings.TrimPrefix(msg, "would "),
		})
	}

	return changes
}

func recordTarget(rec map[string]any) string {
	for _, key := range []string{"dev", "link", "name", "iface"} {
		if v, ok := rec[key].(string); ok && v != "" {
			return v
		}
	}

	return ""
}

func normalize(s string) string { return strings.TrimRight(s, "\n") }

func printDiff(changes []diffChange) {
	if len(changes) == 0 {
		_, _ = fmt.Println("in sync")
		return
	}

	for _, c := range changes {
		target := c.Target
		if target != "" {
			target = " " + target
		}

		_, _ = fmt.Printf("%-8s %s%s\n", c.Kind, c.Detail, target)
		for _, line := range diffBody(c.Diff) {
			_, _ = fmt.Println(line)
		}
	}
}

func diffBody(lines []types.DiffLine) []string {
	const context = 3

	keep := make([]bool, len(lines))
	for i, l := range lines {
		if l.Type == "same" {
			continue
		}

		lo, hi := i-context, i+context
		if lo < 0 {
			lo = 0
		}

		if hi >= len(lines) {
			hi = len(lines) - 1
		}

		for j := lo; j <= hi; j++ {
			keep[j] = true
		}
	}

	var out []string
	last := -1
	for i, l := range lines {
		if !keep[i] {
			continue
		}

		if last >= 0 && i > last+1 {
			out = append(out, "    ...")
		}

		switch l.Type {
		case "add":
			out = append(out, "  + "+l.Text)
		case "remove":
			out = append(out, "  - "+l.Text)
		default:
			out = append(out, "    "+l.Text)
		}

		last = i
	}

	return out
}

func newDiffCommandCallback(asJSON bool, exitCode bool, _ *cobra.Command, args []string) error {
	cfg, err := config.LoadAndValidate(configArg(args), true)
	if err != nil {
		return err
	}

	changes, err := computeDiff(cfg)
	if err != nil {
		return err
	}

	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(map[string]any{"in_sync": len(changes) == 0, "changes": changes}); err != nil {
			return err
		}
	} else {
		printDiff(changes)
	}

	if exitCode && len(changes) > 0 {
		os.Exit(1)
	}

	return nil
}

func computeDiffCallback(changes []diffChange, i, j int) bool {
	if changes[i].Kind != changes[j].Kind {
		return changes[i].Kind < changes[j].Kind
	}

	return changes[i].Target < changes[j].Target
}
