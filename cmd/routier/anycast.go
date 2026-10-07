package main

import (
	"github.com/ChevalRouting/routier/pkg/config"
	anyk "github.com/m-vinc/anyk"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newAnycastCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "anycast",
		Short: "anycast IPs management",
	}

	cmd.AddCommand(newAnycastRunCommand())
	return cmd
}

func newAnycastRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run [config]",
		Short: "run one health check cycle and update BGP routes",
		Args:  cobra.MaximumNArgs(1),
		RunE:  newAnycastRunCommandHandler,
	}
}

func toAnykServices(services []config.AnycastService) []anyk.AnykService {
	out := make([]anyk.AnykService, len(services))
	for i, svc := range services {
		eps := make([]anyk.AnykEndpoint, len(svc.Endpoints))
		for j, ep := range svc.Endpoints {
			ae := anyk.AnykEndpoint{IP: ep.IP, Distance: ep.Distance}
			if ep.HTTPCheck != nil {
				ae.HTTPCheck = &anyk.AnykHTTPCheck{
					Verb:         ep.HTTPCheck.Verb,
					URL:          ep.HTTPCheck.URL,
					ExpectedCode: ep.HTTPCheck.ExpectedCode,
					Headers:      ep.HTTPCheck.Headers,
					Body:         ep.HTTPCheck.Body,
					Timeout:      ep.HTTPCheck.Timeout,
				}
			}

			if ep.DNSCheck != nil {
				ae.DNSCheck = &anyk.AnykDNSCheck{
					Resolver: ep.DNSCheck.Resolver,
					Type:     ep.DNSCheck.Type,
					Query:    ep.DNSCheck.Query,
					Expected: ep.DNSCheck.Expected,
					Timeout:  ep.DNSCheck.Timeout,
				}
			}

			eps[j] = ae
		}

		out[i] = anyk.AnykService{
			Name:       svc.Name,
			Active:     svc.Active,
			AnycastIPs: svc.AnycastIPs,
			Endpoints:  eps,
		}
	}

	return out
}

func newAnycastRunCommandHandler(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadAndValidate(configArg(args), true)
	if err != nil {
		return err
	}

	if cfg.Routing == nil || cfg.Routing.Anycast == nil {
		return nil
	}

	ctx := log.Logger.WithContext(cmd.Context())
	if err := anyk.Run(ctx, cfg.Routing.BGP.ASN, toAnykServices(cfg.Routing.Anycast.Services)); err != nil {
		log.Error().Err(err).Msg("anycast run failed")
	}

	return nil
}
