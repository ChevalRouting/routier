package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/macro"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

func openMacroDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db %s: %w", path, err)
	}

	return db, nil
}

func newMacrosCommand() *cobra.Command {
	var dbPath string
	cmd := &cobra.Command{
		Use:   "macros",
		Short: "manage config macros",
	}

	cmd.PersistentFlags().StringVar(&dbPath, "db", "/var/lib/routier/web.db", "path to SQLite database")
	cmd.AddCommand(
		newMacrosListCommand(&dbPath),
		newMacrosShowCommand(&dbPath),
		newMacrosApplyCommand(&dbPath),
		newMacrosDeleteCommand(&dbPath),
	)
	return cmd
}

func newMacrosListCommand(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list macros",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			db, err := openMacroDB(*dbPath)
			if err != nil {
				return err
			}

			defer db.Close()

			macros, err := webdb.ListMacros(db)
			if err != nil {
				return err
			}

			if len(macros) == 0 {
				fmt.Println("no macros saved")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tSECTIONS\tAPPLIED\tCREATED BY\tDESCRIPTION")
			for _, m := range macros {
				applied := "never"
				if m.AppliedAt != nil {
					applied = fmt.Sprintf("%dx", m.ApplyCount)
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					m.ID, m.Name, strings.Join(m.Sections, ","),
					applied, m.CreatedBy, m.Description)
			}

			return w.Flush()
		},
	}
}

func newMacrosShowCommand(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "show <name-or-id>",
		Short: "show macro details",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			db, err := openMacroDB(*dbPath)
			if err != nil {
				return err
			}

			defer db.Close()

			m, err := webdb.LoadMacroByName(db, args[0])
			if err != nil {
				return err
			}

			if m == nil {
				m, err = webdb.LoadMacro(db, args[0])
				if err != nil {
					return err
				}
			}

			if m == nil {
				return fmt.Errorf("macro %q not found", args[0])
			}

			fmt.Printf("ID:          %s\n", m.ID)
			fmt.Printf("Name:        %s\n", m.Name)
			fmt.Printf("Description: %s\n", m.Description)
			fmt.Printf("Sections:    %s\n", strings.Join(m.Sections, ", "))
			fmt.Printf("Created at:  %s\n", m.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
			fmt.Printf("Created by:  %s\n", m.CreatedBy)
			fmt.Printf("Apply count: %d\n", m.ApplyCount)
			if m.AppliedAt != nil {
				fmt.Printf("Last applied: %s\n", m.AppliedAt.Format("2006-01-02 15:04:05 UTC"))
			}

			return nil
		},
	}
}

func newMacrosApplyCommand(dbPath *string) *cobra.Command {
	var (
		dryRun     bool
		force      bool
		configPath string
		noArm      bool
		timeout    int
	)
	cmd := &cobra.Command{
		Use:   "apply <name-or-id>",
		Short: "apply a macro to the live config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openMacroDB(*dbPath)
			if err != nil {
				return err
			}

			defer db.Close()

			m, err := webdb.LoadMacroByName(db, args[0])
			if err != nil {
				return err
			}

			if m == nil {
				m, err = webdb.LoadMacro(db, args[0])
				if err != nil {
					return err
				}
			}

			if m == nil {
				return fmt.Errorf("macro %q not found", args[0])
			}

			live, err := config.LoadAndValidate(configPath, true)
			if err != nil {
				return fmt.Errorf("load live config: %w", err)
			}

			var base, mod config.Config
			if err := yaml.Unmarshal([]byte(m.BaseYAML), &base); err != nil {
				return fmt.Errorf("parse macro base: %w", err)
			}

			if err := yaml.Unmarshal([]byte(m.ModYAML), &mod); err != nil {
				return fmt.Errorf("parse macro mod: %w", err)
			}

			if !force {
				conflicts := macro.DetectConflicts(&base, live, &mod, m.Sections)
				if len(conflicts) > 0 {
					fmt.Fprintf(os.Stderr, "conflicts detected (use --force to override):\n")
					for _, c := range conflicts {
						if c.Key != "" {
							fmt.Fprintf(os.Stderr, "  %s[%s]: %s\n", c.Section, c.Key, c.Message)
						} else {
							fmt.Fprintf(os.Stderr, "  %s: %s\n", c.Section, c.Message)
						}
					}

					return fmt.Errorf("cannot apply macro: %d conflict(s)", len(conflicts))
				}
			}

			merged := macro.ApplyDelta(&base, live, &mod, m.Sections)

			if errs := config.Validate(merged, true); len(errs) > 0 {
				msgs := make([]string, len(errs))
				for i, e := range errs {
					msgs[i] = e.Error()
				}

				return fmt.Errorf("validation failed: %s", strings.Join(msgs, "; "))
			}

			if dryRun {
				if _, err := render.All(merged, friendRenderOpt()); err != nil {
					return fmt.Errorf("render failed: %w", err)
				}

				log.Info().Str("macro", m.Name).Strs("sections", m.Sections).Msg("dry run: macro would apply")
				return nil
			}

			newData, err := yaml.Marshal(merged)
			if err != nil {
				return fmt.Errorf("marshal merged config: %w", err)
			}

			oldData, _ := os.ReadFile(configPath)
			if err := os.WriteFile(configPath, newData, 0600); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			armTimeout := 0
			if !noArm {
				armTimeout = timeout
			}

			res, err := managers.Apply(cmd.Context(), merged, friends.LoadCacheVars(defaultFriendsCache), managers.ApplyOptions{
				Source:     "macro:" + m.Name,
				ConfigPath: configPath,
			}, armTimeout)
			if err != nil {
				if len(oldData) > 0 {
					_ = os.WriteFile(configPath, oldData, 0600)
				}

				return err
			}

			_ = webdb.MarkMacroApplied(db, m.ID)

			if res.Warning != "" {
				log.Warn().Msg(res.Warning)
			}

			if res.SnapID != "" && !noArm {
				log.Info().Int("timeout", timeout).Msg("confirm pending: routier confirm")
			}

			log.Info().Str("macro", m.Name).Msg("macro applied")
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "check conflicts and validate without applying")
	cmd.Flags().BoolVar(&force, "force", false, "apply even if conflicts are detected")
	cmd.Flags().StringVar(&configPath, "config", defaultConfigPath, "path to live config file")
	cmd.Flags().BoolVar(&noArm, "no-confirm", false, "skip rollback timer")
	cmd.Flags().IntVar(&timeout, "timeout", 60, "rollback timeout seconds")
	return cmd
}

func newMacrosDeleteCommand(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name-or-id>",
		Short: "delete a macro",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			db, err := openMacroDB(*dbPath)
			if err != nil {
				return err
			}

			defer db.Close()

			m, err := webdb.LoadMacroByName(db, args[0])
			if err != nil {
				return err
			}

			if m == nil {
				m, err = webdb.LoadMacro(db, args[0])
				if err != nil {
					return err
				}
			}

			if m == nil {
				return fmt.Errorf("macro %q not found", args[0])
			}

			if err := webdb.DeleteMacro(db, m.ID); err != nil {
				return err
			}

			fmt.Printf("deleted macro %q\n", m.Name)
			return nil
		},
	}
}
