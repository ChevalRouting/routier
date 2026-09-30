# DNS (BIND)

Routier manages a BIND 9 name server declaratively. One daemon covers both
roles a router needs: a **recursive/forwarding resolver** for its clients, and
an **authoritative server** for local zones. Zones, records, forwarding and
ACLs live in the `dns.server` section of the config; the apply pipeline renders
`named.conf` plus one zone file per zone, validates them, and reloads `named`
over its control channel.

## Overview

The `dns` section has two independent halves:

| Key | Scope |
|-----|-------|
| `dns.nameservers`, `dns.search` | the **router's own** resolver, rendered into `/etc/resolv.conf` |
| `dns.server` | the DNS server the router **runs for its clients** |

They are separate on purpose. `nameservers`/`search` only ever touch
`/etc/resolv.conf` (see [system.md](system.md)); they do not configure BIND, and
they are not upstreams for it. A router that resolves through its own server
points `nameservers` at a loopback or listen address it serves:

```yaml
dns:
  nameservers: [127.0.0.1]
  search: [42.school]
  server:
    enabled: true
    ...
```

`dns.server` is optional and additive: omit it (or set `enabled: false`) and
nothing under `/etc/bind/` is rendered, the service is stopped and removed from
the boot runlevel, and no firewall rules are emitted. No config migration is
involved — adding `dns.server` to an existing config does not change the schema
version.

## Modes

The two roles are independent. `dns.server.mode` selects which of them this
server performs:

| Mode | Recursion for clients | Serves `zones` | Use for |
|------|----------------------|----------------|---------|
| `forwarder` | yes | no | a resolver for the network, no local zones |
| `authoritative` | **no** | yes | an internal zone server only |
| `both` | yes | yes | the common appliance case |

`mode` is optional. When omitted it is inferred: zones with no
`upstreams`/`forward` is `authoritative`, zones alongside them is `both`, and
anything else is `forwarder`. Setting it explicitly is worth doing anyway — it
turns a silent misconfiguration into a validation error, so
`mode: authoritative` with an upstream still configured is rejected rather than
quietly recursing.

The mode is not cosmetic. `forwarder` and `both` render `recursion yes` plus
`allow-recursion { routier_clients; }`; `authoritative` renders `recursion no`,
so a query for anything outside the local zones is answered **REFUSED** and no
root `forwarders` block is emitted. Without that distinction an authoritative
server would still recurse from the root hints for any name outside its zones —
an open resolver for every client in `allow_from`, which is both a wider
service than intended and a DNS amplification risk.

```yaml
dns:
  server:
    enabled: true
    mode: authoritative
    listen: [iface(lan)]
    allow_from: [10.0.0.0/24]
    zones:
      - name: home.arpa
        ...
```

## Forwarding resolver

The minimum viable resolver is a set of listen addresses, an ACL, and
upstreams. This is the full configuration of a campus router that forwards one
internal domain and its reverse space to a campus resolver and everything else
to a public one:

```yaml
dns:
  nameservers: [127.0.0.1]
  search: [42.school]
  server:
    enabled: true
    listen:
      - 127.0.0.1
      - iface(region)         # 10.170.32.2
      - vips(region)          # 10.170.32.1  (VRRP VIP)
      - iface(workstations)   # 10.170.48.2
      - vips(workstations)    # 10.170.48.1  (VRRP VIP)
      - iface(external)       # 10.170.33.2
      - iface(dum0)           # 10.255.0.53
    allow_from:
      - 10.170.32.0/24
      - 10.170.48.0/20
      - 10.170.33.0/24
      - 127.0.0.1/32
      - 10.255.0.53/32
    allow_inbound: [region, workstations, external]
    upstreams: [1.1.1.1]
    forward:
      - domain: 10.in-addr.arpa
        servers: [10.255.0.54]
      - domain: 42.school
        servers: [10.255.0.54]
    cache:
      disabled: true
```

Server fields:

| Field | Meaning |
|-------|---------|
| `listen` | addresses to answer on (see below); rendered as `listen-on` / `listen-on-v6` |
| `port` | listen port, 53 by default; the firewall auto-allow follows it |
| `allow_from` | the client ACL, rendered as `acl routier_clients` and used by `allow-query`/`allow-recursion` |
| `allow_inbound` | interfaces on which the firewall accepts DNS (see [Firewall](#firewall)) |
| `upstreams` | root forwarders; omit for full recursion from the root hints |
| `forward` | per-domain conditional forwarding |
| `cache` | cache sizing and TTL clamps |
| `dnssec` | enable DNSSEC validation (off by default) |
| `log_queries` | log every query to the named log, for the live query stream |
| `zones` | authoritative zones |
| `views` | split-horizon views |
| `extra` | raw lines appended inside `options` |

`allow_from` is **mandatory** when the server is enabled. BIND would otherwise
apply its own defaults, and an empty ACL means a resolver nobody can use.

`cache.disabled: true` renders `max-cache-ttl 0` and `max-ncache-ttl 0` rather
than a zero-sized cache: every answer becomes non-cacheable while the cache
structures stay allocated, which is the well-behaved way to express "do not
cache" in BIND.

### Listen address references

A `listen` entry is one of:

- a literal IP — `127.0.0.1`, `0.0.0.0`
- `iface(<name>)` — the static host addresses of that interface or VLAN
- `vips(<name>)` — the VRRP virtual addresses associated with that interface

The reference forms exist because interfaces are selected by MAC or by index
(see [interfaces.md](interfaces.md)), so `wan` stays `wan` even when the NIC
becomes `eth2` on the next box. Literal IPs in the DNS config would silently go
stale in exactly the situation the rest of the config is designed to survive.

Resolution is pure — it reads the config, never the live system — so
`routier plan` stays safe to run anywhere. Addresses are sorted and
de-duplicated, and split into `listen-on` / `listen-on-v6` by family. An
interface whose only address is `dhcp` or `slaac` cannot be a listen target
(the address is unknowable at render time) and is rejected by validation; use a
literal or `0.0.0.0` there.

## VRRP virtual addresses

Listening on a VRRP VIP is a normal thing for an HA pair to want: clients point
at the VIP, and whichever node holds it answers. But the VIP does not exist on
the BACKUP node, and BIND binds its `listen-on` addresses when it starts or
reconfigures.

Routier handles this by emitting

```
net.ipv4.ip_nonlocal_bind = 1
net.ipv6.ip_nonlocal_bind = 1
```

into the rendered sysctl whenever any `listen` entry is a `vips(...)` reference
— unless you have already set `net.ipv4.ip_nonlocal_bind` yourself, in which
case your value wins. With it, `named` binds the VIP at startup whether or not
the address is currently present, and no keepalived transition hook is needed.

Note this is a **global** kernel toggle, not a per-socket one; it allows any
process on the box to bind a non-local address. If you would rather not enable
it, the alternative is `listen: [0.0.0.0]` with the firewall restricting who can
reach port 53, at the cost of the per-address precision.

## Authoritative zones

```yaml
dns:
  server:
    enabled: true
    mode: both
    listen: [iface(lan)]
    allow_from: [10.0.0.0/24]
    allow_inbound: [lan]
    upstreams: [1.1.1.1, 9.9.9.9]
    cache: { size: 64m, max_ttl: 3600 }
    zones:
      - name: home.arpa
        ttl: 300
        nameservers: [ns1.home.arpa.]
        soa:
          primary: ns1.home.arpa.
          email: hostmaster@home.arpa
          refresh: 3600
          retry: 600
          expire: 604800
          minimum: 300
        records:
          - { name: "@",   type: A,     value: 10.0.0.1 }
          - { name: ns1,   type: A,     value: 10.0.0.1 }
          - { name: nas,   type: A,     value: 10.0.0.10 }
          - { name: nas,   type: AAAA,  value: "2001:db8::10" }
          - { name: files, type: CNAME, value: nas.home.arpa. }
          - { name: "@",   type: MX,    value: mail.home.arpa., priority: 10 }
          - { name: "@",   type: TXT,   value: "managed by routier" }
      - name: 0.0.10.in-addr.arpa
        nameservers: [ns1.home.arpa.]
        soa: { primary: ns1.home.arpa., email: hostmaster@home.arpa }
        records:
          - { name: "1",  type: PTR, value: gw.home.arpa. }
          - { name: "10", type: PTR, value: nas.home.arpa. }
```

A zone carries either `records` (a primary zone, rendered to a zone file) or
`primaries` (a secondary zone, transferred in from those addresses) — never
both. Supported record types are `A`, `AAAA`, `CNAME`, `MX`, `TXT`, `SRV`,
`PTR`, `NS`, `CAA`, `SSHFP` and `TLSA`; `priority` applies to `MX` and `SRV`
only. `soa.email` is written in either form — `hostmaster@home.arpa` or
`hostmaster.home.arpa.` — and converted to zone-file form on render.

### Nameservers need address records

Every name in `nameservers` that falls **inside** the zone must also have an
`A` or `AAAA` record in that zone. This is not a style preference:
`named-checkzone` treats missing NS glue as an **error**, not a warning, so the
zone would be rejected at apply time. Validation catches it first, with a
message naming the nameserver. A nameserver outside the zone needs no glue and
is accepted as-is.

### Serials are content-hashed, never timestamps

If `soa.serial` is set, it is used verbatim. Otherwise the serial is an FNV-32
hash of the rendered zone content, so it changes exactly when the zone changes
and never otherwise.

This matters because `routier plan` and `GET /api/config/render-diff` diff
rendered output, and the apply step skips writing files whose content is
unchanged. A timestamp serial would make every plan show a phantom diff and
every apply rewrite every zone file — and, because each zone is its own render
artifact, reload every zone that did not actually change.

### Reverse zones

Reverse zones are ordinary zones whose name is the `in-addr.arpa` /
`ip6.arpa` form. `routier ipcalc reverse <cidr>` computes the zone name for a
prefix (see [tools.md](tools.md)).

### Zones fed by DHCP

When `dhcp.ddns` is enabled, Kea drives a forward zone and the matching reverse
zones through dynamic updates: leases become `A`/`AAAA`/`PTR` records. Routier
creates and owns those zones automatically as dynamic primaries with an
`allow-update` TSIG key, so do not also declare them under `dns.server.zones`.
See [dhcp.md](dhcp.md) for the configuration.

## Views (split-horizon)

`views` answers the same name differently depending on who asks:

```yaml
dns:
  server:
    enabled: true
    listen: [iface(lan), iface(dmz)]
    allow_from: [10.0.0.0/24, 192.0.2.0/24]
    views:
      - name: internal
        match_from: [10.0.0.0/24]
        recursion: true
        upstreams: [1.1.1.1]
        zones:
          - name: corp.example
            nameservers: [ns1.corp.example.]
            soa: { primary: ns1.corp.example., email: hostmaster@corp.example }
            records:
              - { name: "@",  type: A, value: 10.0.0.1 }
              - { name: ns1,  type: A, value: 10.0.0.1 }
              - { name: wiki, type: A, value: 10.0.0.20 }
      - name: external
        match_from: [any]
        recursion: false
        zones:
          - name: corp.example
            nameservers: [ns1.corp.example.]
            soa: { primary: ns1.corp.example., email: hostmaster@corp.example }
            records:
              - { name: "@",  type: A, value: 192.0.2.1 }
              - { name: ns1,  type: A, value: 192.0.2.1 }
```

View fields: `name`, `match_from` (client ACL, `any` when omitted), `recursion`
(overrides the server-level mode for this view), `upstreams`, `forward`,
`zones`, `extra`.

Two BIND rules to know:

- **Views are all-or-nothing.** If any view exists, every zone must live inside
  one; BIND rejects a config that mixes top-level zones with views. Routier
  renders top-level `zones` when there are no views, and view-scoped zones when
  there are, so do not mix them.
- **Order matters.** The first view whose `match_from` matches the client wins,
  so put the most specific view first and the catch-all last.

Zone files are rendered per zone regardless of which view holds them, so two
views serving the same zone name need two differently-named zones or distinct
records; a single zone name rendered twice would collide on its file path.

## Private and internal domains

BIND ships defaults built for a public recursive resolver, and two of them
break an internal, unsigned domain. For every `forward[].domain` and every
`zones[].name` that does not set `dnssec: true`, Routier emits the countermeasure:

```
validate-except {
    "10.in-addr.arpa";
    "42.school";
};
disable-empty-zone "10.in-addr.arpa";
```

Each addresses a different failure:

1. **`validate-except` — SERVFAIL from DNSSEC validation.** When validation is
   on, an internal server serving an unsigned zone under a signed parent yields
   a bogus answer, and bogus answers are returned as SERVFAIL rather than as
   data. `validate-except` marks the domain as an island of trust.
   `validate-except` takes a **list**; BIND rejects repeated statements, so all
   exempt domains are emitted in one block.
2. **`disable-empty-zone` — NXDOMAIN from a built-in empty zone.** With
   `empty-zones-enable` (BIND's default), BIND serves empty zones for the
   RFC 1918 reverse spaces, `home.arpa`, `resolver.arpa`, `service.arpa` and a
   long tail of others. A built-in empty zone **shadows** a `forward` or
   `primary` zone of the same name: the query never reaches your server and
   every lookup returns NXDOMAIN with no upstream traffic on the wire.

`disable-empty-zone` is emitted only for names that actually have a built-in
empty zone — the `in-addr.arpa`, `ip6.arpa`, `home.arpa`, `resolver.arpa`,
`service.arpa` and `empty.arpa` families. Emitting it for an ordinary domain
like `42.school` would be a no-op that BIND warns about on every apply, so it
is left out.

Note that `home.arpa` — the conventional choice for an internal zone, and the
example above — **is** on the empty-zone list. An internal `home.arpa` without
`disable-empty-zone` silently returns NXDOMAIN for everything.

Set `dnssec: true` on a `forward` or `zones` entry when that domain is genuinely
DNSSEC-signed and you want it validated. Routier then omits it from
`validate-except`. Setting it on an unsigned internal domain reintroduces
failure mode 1 in full.

`dns.server.dnssec` (the server-level flag) controls whether validation runs at
all — `dnssec-validation auto` when set, `no` when not. It is off by default.

## How Routier talks to named

BIND is driven over its **rndc control channel** on `127.0.0.1:953`, keyed by
`/etc/bind/rndc.key`. The key is generated with `rndc-confgen` on first apply if
it does not already exist, and is owned by `named` with mode `0640`. Unlike
Kea's unix control socket, this is a loopback TCP port; it is bound to
`127.0.0.1` and its `allow` list is `127.0.0.1` only.

Validation happens in two places, for a reason:

- **Zone files, before they are written**, with `named-checkzone <origin>
  <file>`. Zone files are self-contained, so they validate in a staging
  directory and a bad zone aborts the apply before anything reaches disk.
- **`named.conf`, after it is written**, with `named-checkconf`. The config
  references its zone files by absolute path, so it can only be validated once
  those files exist — the same reason Kea's config is validated in place. A
  non-zero exit aborts the apply and the snapshot rollback fires.

Both gates key off the **exit code**. Both tools emit warnings on healthy input
(zone hygiene notes, unused-`nodefault` notices), and a warning must not fail an
apply.

Reload is as narrow as the change allows:

1. Only zone data changed → `rndc reload <zone>` for each changed zone. Nothing
   else is touched, and no cache is dropped. Because each zone is its own render
   artifact, a one-record edit reloads exactly one zone.
2. `named.conf` changed → `rndc reconfig`, which loads new and changed zones
   without disturbing the rest.
3. `rndc` unreachable → `rc-service named reload`, then `restart` as a last
   resort.
4. `named` not running → `rc-service named start`.

Zone files for zones that have left the config are removed on apply, so a
deleted zone does not linger on disk.

Ordering: `named` is reloaded **before** Kea, so the resolver is answering
before DHCP starts handing its address out to clients. See
[architecture.md](architecture.md).

## Firewall

DNS is opt-in per interface through `allow_inbound`:

```yaml
dns:
  server:
    allow_inbound: [lan, servers]
```

which emits, for each listed interface and each address family present in
`allow_from`:

```
iifname $lan_interfaces ip saddr $dns_allow_from meta l4proto { tcp, udp } th dport 53 accept comment "routier: dns"
```

The rule uses the configured `port`, so a non-default `dns.server.port` is
followed automatically. Unlike the BGP and OSPF auto-allows, the DNS rule is
scoped by the resolver's own `allow_from` ACL rather than by the interface's
subnet: a resolver's clients are routed, not necessarily on-link, so subnet
scoping would drop legitimate queries. Firewall and daemon therefore always
agree on who may query.

`allow_inbound` accepts an interface, VLAN, WireGuard or tunnel name. A name
matching none of those emits no rule and logs a warning rather than failing the
render.

Two variables are exported for use in your own rules:

- `$dns_allow_from` / `$dns_allow_from6` — the client ACL, split by family.
- `$dns_listen` / `$dns_listen6` — the resolved listen addresses.

See [firewall.md](firewall.md).

## Runtime operations

| Endpoint | Purpose |
|----------|---------|
| `GET /api/dns` | mode, recursion, port, listen addresses, upstreams, zones |
| `GET /api/dns/stats` | service state plus key BIND counters |
| `GET /api/dns/zones` | declared records with the live serial per zone |
| `GET /api/dns/zones/{name}` | one zone |
| `POST /api/dns/zones/{name}/reload` | `rndc reload <zone>` |
| `POST /api/dns/cache/flush` | flush the cache, or one name |
| `POST /api/dns/query` | resolve a name through the local resolver |
| `GET /api/dns/queries/stream` | live query log (needs `log_queries: true`) |

`GET /api/dns/stats` returns 200 with `running: false` when `named` is not
running, rather than an error, so the UI can show "stopped" instead of a failure.

The same operations exist on the CLI:

```
routier dns                       overview (mode, listen, zones, state)
routier dns stats                 key statistics
routier dns zones [name]          declared records, serial, live-answer check
routier dns query <name> [type]   resolve through the local resolver
routier dns flush [name]          flush the cache or one name
routier dns reload [zone]         reconfigure, or reload one zone
```

Zone views report both what the config **declares** and whether the running
server actually **answers** for the zone, with its live SOA serial. A zone that
is declared but unanswered is the signature of a zone that failed to load, and
is worth checking against the named log.

## Migrating from VyOS `service dns forwarding`

| VyOS | Routier |
|------|---------|
| `listen-address <ip>` | `dns.server.listen` — prefer `iface(...)` / `vips(...)` |
| `allow-from <cidr>` | `dns.server.allow_from` |
| `name-server <ip>` | `dns.server.upstreams` |
| `domain <d> { name-server <ip> }` | `dns.server.forward[]` |
| `cache-size 0` | `dns.server.cache.disabled: true` |
| `cache-size <n>` | `dns.server.cache.size` |
| `negative-ttl <n>` | `dns.server.cache.max_negative_ttl` |
| `dnssec <mode>` | `dns.server.dnssec` (plus per-domain `dnssec: true`) |
| `ignore-hosts-file` | no key needed — BIND never reads `/etc/hosts` |
| `no-serve-rfc1918` | inverted: Routier emits `disable-empty-zone` for the RFC 1918 reverse spaces you actually forward or serve |
| `port <n>` | `dns.server.port` |
| `system` | set `dns.server.upstreams` explicitly |
| — | `dns.server.zones` has no VyOS equivalent; VyOS forwards only |

## Not supported

- **DNSSEC signing of local zones.** The schema accepts `dnssec: true` on a zone
  and renders `dnssec-policy default` with `inline-signing`, but Alpine's `bind`
  and `bind-tools` packages do not ship `dnssec-keygen`/`dnssec-signzone`, so key
  management is not yet wired up. Treat signing as untested.
- **Zone transfers out.** `allow-transfer { none; }` is rendered
  unconditionally; Routier does not model secondaries of its own zones. BIND is
  capable of it, so this is a schema gap rather than a daemon limitation.
- **Dynamic UPDATE.** `allow-update { none; }` is rendered unconditionally.
  Routier owns the zone files, and an out-of-band update would be overwritten on
  the next apply.
- **Response policy zones / blocklists.** Not modelled.
- **DNS statistics history.** Statistics are live only; they are not recorded
  into the stats database the way interface and BGP metrics are.
