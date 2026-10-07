package main

import (
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/spf13/cobra"
)

func newValidateCommand() *cobra.Command {
	var skipResolve bool
	cmd := &cobra.Command{
		Use:   "validate [config]",
		Short: "validate config without applying",
		Args:  cobra.MaximumNArgs(1),
		PreRunE: func(_ *cobra.Command, args []string) error {
			return validateConfig(validateConfigFileFunc(configArg(args)))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return newValidateCommandCallback(skipResolve, cmd, args)
		},
	}

	cmd.Flags().BoolVar(&skipResolve, "skip-resolve", false, "skip interface resolution")
	return cmd
}

func newValidateCommandCallback(skipResolve bool, cmd *cobra.Command, args []string) error {
	c, err := contextFromContext(cmd)
	if err != nil {
		return err
	}

	log := c.Logger

	cfg, err := config.LoadAndValidate(configArg(args), !skipResolve)
	if err != nil {
		return err
	}

	log.Info().Str("hostname", cfg.Hostname).Msg("ok")
	if len(cfg.Interfaces) > 0 {
		log.Info().Int("count", len(cfg.Interfaces)).Msg("interfaces")
	}

	if cfg.Routing != nil {
		if cfg.Routing.BGP != nil {
			log.Info().Int("asn", cfg.Routing.BGP.ASN).Int("neighbors", len(cfg.Routing.BGP.Neighbors)).Msg("bgp")
		}

		if cfg.Routing.OSPF != nil {
			log.Info().Int("areas", len(cfg.Routing.OSPF.Areas)).Msg("ospf")
		}

		if len(cfg.Routing.Static) > 0 {
			log.Info().Int("count", len(cfg.Routing.Static)).Msg("static routes")
		}
	}

	if len(cfg.Wireguard) > 0 {
		log.Info().Int("count", len(cfg.Wireguard)).Msg("wireguard tunnels")
	}

	if len(cfg.Sysctl) > 0 {
		log.Info().Int("count", len(cfg.Sysctl)).Msg("sysctl keys")
	}

	if len(cfg.Users) > 0 {
		log.Info().Int("count", len(cfg.Users)).Msg("users")
	}

	if cfg.DNS != nil {
		log.Info().Int("nameservers", len(cfg.DNS.Nameservers)).Msg("dns")
	}

	if len(cfg.Services) > 0 {
		log.Info().Int("count", len(cfg.Services)).Msg("services")
	}

	return nil
}
