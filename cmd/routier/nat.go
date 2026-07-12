package main

import (
	"fmt"
	"os"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/nat"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newNatCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nat",
		Short: "NAT shortcuts (masquerade / snat / dnat) as tagged nftables rules",
	}

	cmd.AddCommand(
		newNatListCommand(),
		newNatMasqueradeCommand(),
		newNatSNATCommand(),
		newNatDNATCommand(),
		newNatRemoveCommand(),
	)

	return cmd
}

func loadForNat(args []string) (*config.Config, string, error) {
	path := configArg(args)
	cfg, err := config.Load(path)
	if err != nil {
		return nil, "", err
	}

	return cfg, path, nil
}

func saveNat(cfg *config.Config, path string, specs []nat.Spec) error {
	nat.Apply(cfg, specs)

	if errs := config.Validate(cfg, false); len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errs[0])
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

func newNatListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list [config]",
		Short: "list NAT shortcut rules",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadForNat(args)
			if err != nil {
				return err
			}

			for i, s := range nat.Parse(cfg.Nftables) {
				fmt.Printf("%d\t%s\t%+v\n", i, s.Kind, s)
			}

			return nil
		},
	}
}

func newNatMasqueradeCommand() *cobra.Command {
	var out, source, family, comment string

	cmd := &cobra.Command{
		Use:   "masquerade [config]",
		Short: "add a masquerade rule",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadForNat(args)
			if err != nil {
				return err
			}

			specs := append(nat.Parse(cfg.Nftables), nat.Spec{
				Kind: nat.KindMasquerade, Out: out, Source: source, Family: family, Comment: comment,
			})

			return saveNat(cfg, path, specs)
		},
	}

	cmd.Flags().StringVar(&out, "out", "", "outbound interface variable or name (e.g. $wan_interfaces)")
	cmd.Flags().StringVar(&source, "source", "", "restrict to this source address/variable")
	cmd.Flags().StringVar(&family, "family", "", "source address family: ip or ip6")
	cmd.Flags().StringVar(&comment, "comment", "", "optional comment")
	cmd.MarkFlagRequired("out")

	return cmd
}

func newNatSNATCommand() *cobra.Command {
	var out, to, source, family, comment string

	cmd := &cobra.Command{
		Use:   "snat [config]",
		Short: "add a source-NAT rule",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadForNat(args)
			if err != nil {
				return err
			}

			specs := append(nat.Parse(cfg.Nftables), nat.Spec{
				Kind: nat.KindSNAT, Out: out, To: to, Source: source, Family: family, Comment: comment,
			})

			return saveNat(cfg, path, specs)
		},
	}

	cmd.Flags().StringVar(&out, "out", "", "outbound interface variable or name")
	cmd.Flags().StringVar(&to, "to", "", "source address to rewrite to")
	cmd.Flags().StringVar(&source, "source", "", "restrict to this source address/variable")
	cmd.Flags().StringVar(&family, "family", "", "source address family: ip or ip6")
	cmd.Flags().StringVar(&comment, "comment", "", "optional comment")
	cmd.MarkFlagRequired("out")
	cmd.MarkFlagRequired("to")

	return cmd
}

func newNatDNATCommand() *cobra.Command {
	var in, proto, dport, to, comment string

	cmd := &cobra.Command{
		Use:   "dnat [config]",
		Short: "add an inbound port-forward (DNAT) rule",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadForNat(args)
			if err != nil {
				return err
			}

			specs := append(nat.Parse(cfg.Nftables), nat.Spec{
				Kind: nat.KindDNAT, In: in, Proto: proto, DPort: dport, To: to, Comment: comment,
			})

			return saveNat(cfg, path, specs)
		},
	}

	cmd.Flags().StringVar(&in, "in", "", "inbound interface variable or name")
	cmd.Flags().StringVar(&proto, "proto", "tcp", "protocol: tcp or udp")
	cmd.Flags().StringVar(&dport, "dport", "", "external destination port(s)")
	cmd.Flags().StringVar(&to, "to", "", "internal target address[:port]")
	cmd.Flags().StringVar(&comment, "comment", "", "optional comment")
	cmd.MarkFlagRequired("dport")
	cmd.MarkFlagRequired("to")

	return cmd
}

func newNatRemoveCommand() *cobra.Command {
	var index int

	cmd := &cobra.Command{
		Use:   "rm [config]",
		Short: "remove a NAT rule by its list index",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadForNat(args)
			if err != nil {
				return err
			}

			specs := nat.Parse(cfg.Nftables)
			if index < 0 || index >= len(specs) {
				return fmt.Errorf("index %d out of range (%d rules)", index, len(specs))
			}

			specs = append(specs[:index], specs[index+1:]...)
			return saveNat(cfg, path, specs)
		},
	}

	cmd.Flags().IntVar(&index, "index", -1, "index from `nat list`")
	cmd.MarkFlagRequired("index")

	return cmd
}
