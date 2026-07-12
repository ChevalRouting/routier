package main

import (
	"fmt"

	"github.com/ChevalRouting/routier/pkg/iptools"
	"github.com/spf13/cobra"
)

func newIpcalcCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ipcalc",
		Short: "IP address utilities (subnet breakdown, reverse DNS, range to CIDR)",
	}

	cmd.AddCommand(
		newIpcalcSubnetCommand(),
		newIpcalcReverseCommand(),
		newIpcalcRangeCommand(),
	)

	return cmd
}

func newIpcalcSubnetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "subnet <address|cidr>",
		Short: "show an ipcalc-style subnet breakdown",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := iptools.Subnet(args[0])
			if err != nil {
				return err
			}

			printSubnet(cmd, info)
			return nil
		},
	}
}

func newIpcalcReverseCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reverse <address|cidr>",
		Short: "show the reverse-DNS (PTR) name and delegation zone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := iptools.Reverse(args[0])
			if err != nil {
				return err
			}

			cmd.Println(out.Name)
			if out.Zone != "" {
				cmd.Printf("zone: %s\n", out.Zone)
			}

			return nil
		},
	}
}

func newIpcalcRangeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "range <start> <end>",
		Short: "split an inclusive address range into CIDR blocks",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := iptools.Range(args[0], args[1])
			if err != nil {
				return err
			}

			for _, c := range out.CIDRs {
				cmd.Println(c)
			}

			return nil
		},
	}
}

func printSubnet(cmd *cobra.Command, info *iptools.SubnetInfo) {
	is4 := info.Family == "v4"
	bin := func(b string) string {
		if b == "" {
			return ""
		}

		return iptools.FormatBinary(b, info.Prefix, is4)
	}

	row := func(label, value, binary string) {
		cmd.Printf("%-10s %-22s %s\n", label+":", value, binary)
	}

	netmask := info.Netmask
	if is4 {
		netmask = fmt.Sprintf("%s = %d", info.Netmask, info.Prefix)
	}

	row("Address", info.Address, bin(info.AddressBits))
	row("Netmask", netmask, bin(info.NetmaskBits))
	row("Wildcard", info.Wildcard, bin(info.WildcardBits))
	cmd.Println("=>")

	networkLabel := "Network"
	if info.HostRoute {
		networkLabel = "Hostroute"
	}

	row(networkLabel, info.Network, bin(info.NetworkBits))
	if !info.HostRoute {
		row("HostMin", info.HostMin, bin(info.HostMinBits))
		row("HostMax", info.HostMax, bin(info.HostMaxBits))
	}
	if info.Broadcast != "" {
		row("Broadcast", info.Broadcast, bin(info.BroadcastBits))
	}

	trailer := info.Scope
	if info.Class != "" {
		trailer = fmt.Sprintf("Class %s, %s", info.Class, info.Scope)
	}
	cmd.Printf("%-10s %-22s %s\n", "Hosts/Net:", info.Hosts, trailer)
}
