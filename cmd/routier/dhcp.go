package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/daemon/kea"
	"github.com/spf13/cobra"
)

func keaClient(configPath string) (*kea.Client, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = &config.Config{}
	}

	url, user, password := "", "", ""
	if cfg.DHCP != nil && cfg.DHCP.ControlAgent != nil {
		ca := cfg.DHCP.ControlAgent
		url, user, password = ca.URL, ca.User, ca.Password
	}

	return kea.Resolve(url, user, password)
}

func newDHCPCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "dhcp",
		Short: "manage the Kea DHCP server (leases and reservations)",
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", defaultConfigPath, "path to config file")
	cmd.AddCommand(newDHCPLeasesCommand(&configPath), newDHCPReservationsCommand(&configPath))
	return cmd
}

func newDHCPLeasesCommand(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "leases",
		Short: "list active DHCP leases",
		Args:  cobra.NoArgs,
		RunE: func(unusedArg1 *cobra.Command, unusedArg2 []string) error {
			return newDHCPLeasesCommandCallback(configPath, unusedArg1, unusedArg2)
		},
	}

	cmd.AddCommand(newDHCPClearCommand(configPath))
	return cmd
}

func newDHCPClearCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <ip>",
		Short: "delete an active lease",
		Args:  cobra.ExactArgs(1),
		RunE: func(unusedArg1 *cobra.Command, args []string) error {
			return newDHCPClearCommandCallback(configPath, unusedArg1, args)
		},
	}
}

func newDHCPReservationsCommand(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reservations",
		Aliases: []string{"reservation", "res"},
		Short:   "list host reservations",
		Args:    cobra.NoArgs,
		RunE: func(unusedArg1 *cobra.Command, unusedArg2 []string) error {
			return newDHCPReservationsCommandCallback(configPath, unusedArg1, unusedArg2)
		},
	}

	cmd.AddCommand(newDHCPAddCommand(configPath), newDHCPDelCommand(configPath), newDHCPPersistCommand(configPath))
	return cmd
}

func newDHCPAddCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "add <hostname> <mac-or-duid> <ip-or-cidr>",
		Short: "reserve an address (an IP is auto-picked when only a cidr is given)",
		Args:  cobra.ExactArgs(3),
		RunE: func(unusedArg1 *cobra.Command, args []string) error {
			return newDHCPAddCommandCallback(configPath, unusedArg1, args)
		},
	}
}

func newDHCPDelCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "del <ip>",
		Short: "delete a reservation",
		Args:  cobra.ExactArgs(1),
		RunE: func(unusedArg1 *cobra.Command, args []string) error {
			return newDHCPDelCommandCallback(configPath, unusedArg1, args)
		},
	}
}

func newDHCPPersistCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "persist <ip>",
		Short: "promote an active lease to a reservation",
		Args:  cobra.ExactArgs(1),
		RunE: func(unusedArg1 *cobra.Command, args []string) error {
			return newDHCPPersistCommandCallback(configPath, unusedArg1, args)
		},
	}
}

func newDHCPLeasesCommandCallback(configPath *string, _ *cobra.Command, _ []string) error {
	client, err := keaClient(*configPath)
	if err != nil {
		return err
	}

	views, err := client.Overview()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "SUBNET\tIP\tMAC/DUID\tHOSTNAME")
	any := false
	for _, v := range views {
		for _, l := range v.Leases {
			id := l.HWAddress
			if id == "" {
				id = l.DUID
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", v.Subnet, l.IPAddress, id, l.Hostname)
			any = true
		}
	}

	if !any {
		_, _ = fmt.Println("no active leases")
		return nil
	}

	return w.Flush()
}

func newDHCPClearCommandCallback(configPath *string, _ *cobra.Command, args []string) error {
	client, err := keaClient(*configPath)
	if err != nil {
		return err
	}

	if err := client.ClearLease(args[0]); err != nil {
		return err
	}

	_, _ = fmt.Printf("cleared lease %s\n", args[0])
	return nil
}

func newDHCPReservationsCommandCallback(configPath *string, _ *cobra.Command, _ []string) error {
	client, err := keaClient(*configPath)
	if err != nil {
		return err
	}

	views, err := client.Overview()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "SUBNET\tIP\tMAC/DUID\tHOSTNAME")
	any := false
	for _, v := range views {
		for _, h := range v.Reservations {
			ip := h.IPAddress
			if ip == "" && len(h.IPAddresses) > 0 {
				ip = h.IPAddresses[0]
			}

			id := h.HWAddress
			if id == "" {
				id = h.DUID
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", v.Subnet, ip, id, h.Hostname)
			any = true
		}
	}

	if !any {
		_, _ = fmt.Println("no reservations")
		return nil
	}

	return w.Flush()
}

func newDHCPAddCommandCallback(configPath *string, _ *cobra.Command, args []string) error {
	client, err := keaClient(*configPath)
	if err != nil {
		return err
	}

	ip, err := client.AddReservation(args[0], args[1], args[2])
	if err != nil {
		return err
	}

	_, _ = fmt.Printf("reserved %s for %s (%s)\n", ip, args[0], args[1])
	return nil
}

func newDHCPDelCommandCallback(configPath *string, _ *cobra.Command, args []string) error {
	client, err := keaClient(*configPath)
	if err != nil {
		return err
	}

	if err := client.DelReservation(args[0]); err != nil {
		return err
	}

	_, _ = fmt.Printf("deleted reservation %s\n", args[0])
	return nil
}

func newDHCPPersistCommandCallback(configPath *string, _ *cobra.Command, args []string) error {
	client, err := keaClient(*configPath)
	if err != nil {
		return err
	}

	ip, _, err := client.PersistLease(args[0])
	if err != nil {
		return err
	}

	_, _ = fmt.Printf("persisted %s as a reservation\n", ip)
	return nil
}
