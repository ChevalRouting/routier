package main

import (
	"fmt"

	"github.com/ChevalRouting/routier/pkg/net/ipcalc"
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
		RunE:  newIpcalcSubnetCommandHandler,
	}
}

func newIpcalcReverseCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reverse <address|cidr>",
		Short: "show the reverse-DNS (PTR) name and delegation zone",
		Args:  cobra.ExactArgs(1),
		RunE:  newIpcalcReverseCommandHandler,
	}
}

func newIpcalcRangeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "range <start> <end>",
		Short: "split an inclusive address range into CIDR blocks",
		Args:  cobra.ExactArgs(2),
		RunE:  newIpcalcRangeCommandHandler,
	}
}

func printSubnet(cmd *cobra.Command, info *ipcalc.SubnetInfo) {
	is4 := info.Family == "v4"
	bin := func(b string) string { return printSubnetCallback(info, is4, b) }

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

func newIpcalcSubnetCommandHandler(cmd *cobra.Command, args []string) error {
	info, err := ipcalc.Subnet(args[0])
	if err != nil {
		return err
	}

	printSubnet(cmd, info)
	return nil
}

func newIpcalcReverseCommandHandler(cmd *cobra.Command, args []string) error {
	out, err := ipcalc.Reverse(args[0])
	if err != nil {
		return err
	}

	cmd.Println(out.Name)
	if out.Zone != "" {
		cmd.Printf("zone: %s\n", out.Zone)
	}

	return nil
}

func newIpcalcRangeCommandHandler(cmd *cobra.Command, args []string) error {
	out, err := ipcalc.Range(args[0], args[1])
	if err != nil {
		return err
	}

	for _, c := range out.CIDRs {
		cmd.Println(c)
	}

	return nil
}

func printSubnetCallback(info *ipcalc.SubnetInfo, is4 bool, b string) string {
	if b == "" {
		return ""
	}

	return ipcalc.FormatBinary(b, info.Prefix, is4)
}
