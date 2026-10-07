package main

import (
	"fmt"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/svc"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newPlanCommand() *cobra.Command {
	var module string
	var resolve bool
	var raw bool
	var check bool
	cmd := &cobra.Command{
		Use:   "plan [config]",
		Short: "show rendered output without applying",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(unusedArg4 *cobra.Command, args []string) error {
			return newPlanCommandCallback(module, resolve, raw, check, unusedArg4, args)
		},
	}

	cmd.Flags().StringVarP(&module, "module", "m", "", "show only this template")
	cmd.Flags().BoolVar(&resolve, "resolve", false, "resolve interface names against current system")
	cmd.Flags().BoolVar(&raw, "raw", false, "dump parsed config as YAML, skip validation and resolution")
	cmd.Flags().BoolVar(&check, "check", false, "validate rendered artifacts with their external tools")
	return cmd
}

func newPlanCommandCallback(module string, resolve bool, raw bool, check bool, _ *cobra.Command, args []string) error {
	cfgPath := configArg(args)
	if raw {
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}

		out, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}

		_, _ = fmt.Print(string(out))
		return nil
	}

	cfg, err := config.LoadAndValidate(cfgPath, resolve)
	if err != nil {
		return err
	}

	outputs, err := render.All(cfg, friendRenderOpt())
	if err != nil {
		return err
	}

	for _, o := range outputs {
		if module != "" && o.Name != module {
			continue
		}

		_, _ = fmt.Printf("--- %s -> %s ---\n", o.Name, o.Dest)
		_, _ = fmt.Println(o.Content)
	}

	if check {
		errs := svc.ValidateArtifacts(outputs)
		if len(errs) == 0 {
			_, _ = fmt.Println("=== validation ok ===")
			return nil
		}

		_, _ = fmt.Println("=== validation failed ===")
		for _, e := range errs {
			loc := e.Dest
			if e.Line > 0 {
				loc = fmt.Sprintf("%s:%d", e.Dest, e.Line)
			}

			_, _ = fmt.Printf("  %s: %s\n", loc, e.Message)
		}

		return fmt.Errorf("artifact validation failed")
	}

	return nil
}
