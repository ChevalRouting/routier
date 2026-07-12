package main

import (
	"fmt"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newPlanCommand() *cobra.Command {
	var module string
	var resolve bool
	var raw bool
	cmd := &cobra.Command{
		Use:   "plan [config]",
		Short: "show rendered output without applying",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
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

				fmt.Print(string(out))
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

				fmt.Printf("--- %s -> %s ---\n", o.Name, o.Dest)
				fmt.Println(o.Content)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&module, "module", "m", "", "show only this template")
	cmd.Flags().BoolVar(&resolve, "resolve", false, "resolve interface names against current system")
	cmd.Flags().BoolVar(&raw, "raw", false, "dump parsed config as YAML, skip validation and resolution")
	return cmd
}
