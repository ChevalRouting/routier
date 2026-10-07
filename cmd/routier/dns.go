package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/daemon/bind"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/spf13/cobra"
)

func namedClient() (*bind.Client, error) {
	return bind.LoadLocal()
}

func dnsServerConfig(configPath string) (*config.Config, *config.DNSServer, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, nil, err
	}

	if cfg.DNS == nil || cfg.DNS.Server == nil || !cfg.DNS.Server.Enabled {
		return cfg, nil, fmt.Errorf("no dns server is configured")
	}

	return cfg, cfg.DNS.Server, nil
}

func dnsZoneInputs(s *config.DNSServer) []bind.ZoneInput {
	out := make([]bind.ZoneInput, 0, len(s.Zones))
	for _, z := range s.Zones {
		in := bind.ZoneInput{Name: z.Name}
		if z.SOA != nil {
			in.Serial = z.SOA.Serial
		}

		for _, rec := range z.Records {
			in.Records = append(in.Records, bind.ZoneRecord{
				Name: rec.Name, Type: rec.Type, Value: rec.Value,
				TTL: rec.TTL, Priority: rec.Priority,
			})
		}

		out = append(out, in)
	}

	return out
}

func newDNSCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "dns",
		Short: "inspect the local DNS server (BIND)",
		Args:  cobra.NoArgs,
		RunE: func(unusedArg1 *cobra.Command, unusedArg2 []string) error {
			return newDNSCommandCallback(configPath, unusedArg1, unusedArg2)
		},
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", defaultConfigPath, "path to config file")
	cmd.AddCommand(
		newDNSStatsCommand(),
		newDNSZonesCommand(&configPath),
		newDNSQueryCommand(),
		newDNSFlushCommand(),
		newDNSReloadCommand(),
	)

	return cmd
}

func newDNSStatsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "show resolver statistics",
		Args:  cobra.NoArgs,
		RunE:  newDNSStatsCommandHandler,
	}
}

func newDNSZonesCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "zones [name]",
		Short: "list authoritative zones and their records",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(unusedArg1 *cobra.Command, args []string) error {
			return newDNSZonesCommandCallback(configPath, unusedArg1, args)
		},
	}
}

func newDNSQueryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "query <name> [type]",
		Short: "resolve a name through the local resolver",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  newDNSQueryCommandHandler,
	}
}

func newDNSFlushCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "flush [name]",
		Short: "flush the resolver cache, or one name within it",
		Args:  cobra.MaximumNArgs(1),
		RunE:  newDNSFlushCommandHandler,
	}
}

func newDNSReloadCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reload [zone]",
		Short: "reload the resolver config, or one authoritative zone",
		Args:  cobra.MaximumNArgs(1),
		RunE:  newDNSReloadCommandHandler,
	}
}

func newDNSStatsCommandHandler(_ *cobra.Command, _ []string) error {
	client, err := namedClient()
	if err != nil {
		return err
	}

	stats, err := client.Statistics()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	for _, s := range bind.KeyStatistics(stats) {
		_, _ = fmt.Fprintf(w, "%s\t%.0f\n", s.Name, s.Value)
	}

	return w.Flush()
}

func newDNSQueryCommandHandler(_ *cobra.Command, args []string) error {
	client, err := namedClient()
	if err != nil {
		return err
	}

	qtype := ""
	if len(args) == 2 {
		qtype = args[1]
	}

	result, err := client.Query(args[0], qtype)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "rcode\t%s\n", result.RCode)
	if result.QueryMS > 0 {
		_, _ = fmt.Fprintf(w, "time\t%dms\n", result.QueryMS)
	}

	for _, a := range result.Answers {
		_, _ = fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", a.Name, a.TTL, a.Type, a.Data)
	}

	return w.Flush()
}

func newDNSFlushCommandHandler(_ *cobra.Command, args []string) error {
	client, err := namedClient()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		if err := client.FlushAll(); err != nil {
			return err
		}

		_, _ = fmt.Println("cache flushed")

		return nil
	}

	if err := client.FlushName(args[0]); err != nil {
		return err
	}

	_, _ = fmt.Printf("flushed %s\n", args[0])

	return nil
}

func newDNSReloadCommandHandler(_ *cobra.Command, args []string) error {
	client, err := namedClient()
	if err != nil {
		return err
	}

	if len(args) == 1 {
		if err := client.ReloadZone(args[0]); err != nil {
			return err
		}

		_, _ = fmt.Printf("reloaded zone %s\n", args[0])

		return nil
	}

	if err := client.Reconfig(); err != nil {
		return err
	}

	_, _ = fmt.Println("config reloaded")

	return nil
}

func newDNSCommandCallback(configPath string, _ *cobra.Command, _ []string) error {
	cfg, s, err := dnsServerConfig(configPath)
	if err != nil {
		return err
	}

	client, err := namedClient()
	if err != nil {
		return err
	}

	config.ResolveInterfaces(cfg)

	view, err := client.Overview(bind.OverviewInput{
		Listen: render.DNSListenAddresses(cfg),
		Zones:  dnsZoneInputs(s),
	})
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "mode\t%s\n", s.ResolvedMode())
	_, _ = fmt.Fprintf(w, "recursion\t%t\n", s.Recurses())
	_, _ = fmt.Fprintf(w, "running\t%t\n", view.Running)
	for _, addr := range view.Listen {
		_, _ = fmt.Fprintf(w, "listen\t%s\n", addr)
	}

	for _, up := range s.Upstreams {
		_, _ = fmt.Fprintf(w, "upstream\t%s\n", up)
	}

	for _, z := range view.Zones {
		_, _ = fmt.Fprintf(w, "zone\t%s (serial %d, records %d, answered %t)\n",
			z.Name, z.Serial, len(z.Records), z.Answered)
	}

	return w.Flush()
}

func newDNSZonesCommandCallback(configPath *string, _ *cobra.Command, args []string) error {
	_, s, err := dnsServerConfig(*configPath)
	if err != nil {
		return err
	}

	inputs := dnsZoneInputs(s)

	var views []bind.ZoneView
	if client, err := namedClient(); err == nil {
		views = client.Zones(inputs)
	} else {
		views = bind.DeclaredZones(inputs)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	for _, z := range views {
		if len(args) == 1 && config.NormalizeDNSName(z.Name) != config.NormalizeDNSName(args[0]) {
			continue
		}

		_, _ = fmt.Fprintf(w, "%s\tserial %d\tanswered %t\n", z.Name, z.Serial, z.Answered)
		for _, rec := range z.Records {
			_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\n", rec.Name, rec.Type, rec.Value)
		}
	}

	return w.Flush()
}
