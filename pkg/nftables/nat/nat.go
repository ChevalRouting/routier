package nat

import "github.com/ChevalRouting/routier/pkg/config"

const Tag = "nat"

const (
	chainPostrouting = "postrouting"
	chainPrerouting  = "prerouting"
)

type Kind string

const (
	KindMasquerade Kind = "masquerade"
	KindSNAT       Kind = "snat"
	KindDNAT       Kind = "dnat"
)

type Spec struct {
	Kind    Kind   `json:"kind"`
	Out     string `json:"out,omitempty" validate:"optional"`
	In      string `json:"in,omitempty" validate:"optional"`
	Source  string `json:"source,omitempty" validate:"optional"`
	Family  string `json:"family,omitempty" validate:"optional"`
	Proto   string `json:"proto,omitempty" validate:"optional"`
	DPort   string `json:"dport,omitempty" validate:"optional"`
	To      string `json:"to,omitempty" validate:"optional"`
	Comment string `json:"comment,omitempty" validate:"optional"`
}

func rule(s Spec) (string, config.ManagedRule) {
	r := config.ManagedRule{Tag: Tag, Comment: s.Comment}

	switch s.Kind {
	case KindMasquerade:
		r.Action = "masquerade"
		r.Match = &config.RuleMatch{OIF: s.Out, SAddr: s.Source, AddrFamily: sourceFamily(s)}
		return chainPostrouting, r
	case KindSNAT:
		r.Action = "snat"
		r.ActionTo = s.To
		r.Match = &config.RuleMatch{OIF: s.Out, SAddr: s.Source, AddrFamily: sourceFamily(s)}
		return chainPostrouting, r
	case KindDNAT:
		r.Action = "dnat"
		r.ActionTo = s.To
		r.Match = &config.RuleMatch{IIF: s.In, Protocol: s.Proto, DPort: s.DPort}
		return chainPrerouting, r
	}

	return "", r
}

func sourceFamily(s Spec) string {
	if s.Source == "" {
		return ""
	}

	return s.Family
}

func Rules(specs []Spec) map[string][]config.ManagedRule {
	out := map[string][]config.ManagedRule{}
	for _, s := range specs {
		chain, r := rule(s)
		if chain == "" {
			continue
		}

		out[chain] = append(out[chain], r)
	}

	return out
}

func Parse(nft *config.NftablesConfig) []Spec {
	if nft == nil {
		return nil
	}

	var specs []Spec
	for _, chain := range []string{chainPostrouting, chainPrerouting} {
		ch := nft.Chains[chain]
		if ch == nil {
			continue
		}

		for _, r := range ch.Managed {
			if r.Tag != Tag {
				continue
			}

			if s, ok := specFromRule(r); ok {
				specs = append(specs, s)
			}
		}
	}

	return specs
}

func specFromRule(r config.ManagedRule) (Spec, bool) {
	s := Spec{Comment: r.Comment}
	m := r.Match

	switch r.Action {
	case "masquerade":
		s.Kind = KindMasquerade
		if m != nil {
			s.Out, s.Source, s.Family = m.OIF, m.SAddr, m.AddrFamily
		}
	case "snat":
		s.Kind = KindSNAT
		s.To = r.ActionTo
		if m != nil {
			s.Out, s.Source, s.Family = m.OIF, m.SAddr, m.AddrFamily
		}
	case "dnat":
		s.Kind = KindDNAT
		s.To = r.ActionTo
		if m != nil {
			s.In, s.Proto, s.DPort = m.IIF, m.Protocol, m.DPort
		}
	default:
		return Spec{}, false
	}

	return s, true
}

func Apply(cfg *config.Config, specs []Spec) {
	if cfg.Nftables == nil {
		cfg.Nftables = &config.NftablesConfig{}
	}

	if cfg.Nftables.Chains == nil {
		cfg.Nftables.Chains = map[string]*config.NftChain{}
	}

	generated := Rules(specs)

	for _, chain := range []string{chainPostrouting, chainPrerouting} {
		ch := cfg.Nftables.Chains[chain]

		var kept []config.ManagedRule
		if ch != nil {
			for _, r := range ch.Managed {
				if r.Tag != Tag {
					kept = append(kept, r)
				}
			}
		}

		kept = append(kept, generated[chain]...)

		if len(kept) == 0 {
			if ch != nil {
				ch.Managed = nil
				if chainEmpty(ch) {
					delete(cfg.Nftables.Chains, chain)
				}
			}

			continue
		}

		if ch == nil {
			ch = &config.NftChain{}
			cfg.Nftables.Chains[chain] = ch
		}

		ch.Managed = kept
	}
}

func chainEmpty(ch *config.NftChain) bool {
	return ch.Policy == "" && ch.Rules == "" && len(ch.Files) == 0 && len(ch.Managed) == 0
}
