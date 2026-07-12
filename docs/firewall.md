# Firewall (nftables)

Routier **owns** a dual-stack `table inet routier` and its base chains. You
contribute rules into those chains; Routier owns the table, the chains, their
type/hook/priority, and the mandatory rules, and it automatically allows the
traffic that enabled features need. This split means your rules never have to
account for Routier's own plumbing.

## Model

```yaml
nftables:
  defines: |
    define my_set = { 10.0.0.0/8, 2001:db8::/48 }
  chains:
    input:
      policy: drop
      rules: |
        ct state established,related accept
        ip saddr $lan_network tcp dport 22 accept
    forward:
      rules: |
        ct state established,related accept
        ct state invalid drop
  include:
    - /etc/routier/extra.nft
```

- `defines` - raw nft `define` lines emitted before the ruleset.
- `chains.<name>` - one of the owned chains: `input`, `forward`, `output`
  (filter) and `prerouting`, `postrouting` (nat).
  - `policy` - override the chain's default policy (`accept`/`drop`).
  - `rules` - a raw nft rule block, preserved verbatim (stored as a YAML block
    scalar so indentation survives), appended after Routier's own rules.
  - `files` - paths to files of raw rule lines appended to the chain.
  - `managed` - an optional structured (UI-friendly) form of individual rules.
- `include` - complete nft files rendered as-is, with their own tables and
  chains.

A note on rule semantics: `accept` is terminal only within its chain, and only
`drop` is globally terminal. This is why Routier owns the base chains and
emits its rules before yours.

## Variables

Routier emits `define`s you can reference with `$` in your rules:

- `$me` / `$me6` - every IPv4 / IPv6 address on the router.
- `$<iface>_address` / `$<iface>_network` (and `*6`) - per-interface host and
  network addresses.
- interface-membership sets used by the auto-allow rules.

`GET /api/config/nftables/vars` returns the rendered variable names and
values, and the UI editor autocompletes them when you type `$`.

## Automatic allows

Enabling a feature opens its traffic automatically, with no manual rule
needed:

- WireGuard interfaces with `allow_inbound: true` (their listen port),
- VRRP and conntrackd when configured,
- BGP/OSPF/OSPF6 via their `allow_inbound` interface lists, scoped to those
  interfaces and their subnets.

## Defaults and editing

Default rules ship in the package as first-boot config; once you edit the
ruleset, Routier keeps your rules and only re-applies the automatic allows.
The **Firewall** UI edits each chain in its own tab, with a Defines tab
alongside, and keeps in-progress edits across a refresh.
