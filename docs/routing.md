# Routing

The `routing` section covers static routes and the dynamic protocols rendered
into FRR (BGP, OSPF, OSPF6), plus BFD, policy-based routing, anycast, and IPv6
router advertisements.

## Static routes

```yaml
routing:
  static:
    - destination: 10.0.0.0/8
      via: 192.168.1.254
      metric: 100
    - destination: 2001:db8:1::/48
      dev: lan
    - destination: 192.0.2.1/32
      dev: wan
      scope: link
    - destination: ::/0
      via: fe80::1
      dev: wan
      src: 2001:db8::10
```

Fields per route: `destination` (required), `via`, `dev`, `src`, `metric`,
`scope`.

`src` is the preferred source address for traffic the router itself sends over
the route. It must be in the same address family as the destination and
assigned to one of the router's interfaces, otherwise the kernel rejects the
route. Forwarded traffic keeps its original source.

`scope` is one of `global`, `link`, `host`, or `site`. Use `link` for a
directly-reachable destination that has no gateway (`dev` only).

`type` turns a route into a kernel `blackhole`, `unreachable`, or `prohibit`
route. A typed route takes no `via`, `dev`, `src`, or `scope`. A `blackhole`
silently drops what it matches; `unreachable` and `prohibit` answer with an
ICMP error. Covering the private ranges keeps traffic for networks the router
does not know from following the default route, while any connected, static,
or BGP route that is more specific still wins:

```yaml
routing:
  static:
    - { destination: 10.0.0.0/8, type: blackhole }
    - { destination: 100.64.0.0/10, type: blackhole }
    - { destination: 172.16.0.0/12, type: blackhole }
    - { destination: 192.168.0.0/16, type: blackhole }
```

## BGP

```yaml
routing:
  bgp:
    asn: 65001
    router_id: 192.0.2.1
    allow_inbound: [wan]
    neighbors:
      - { ... }
    address_families: { ... }
    prefix_lists: { ... }
    route_maps: { ... }
```

Key fields: `asn` (required), `router_id`, `neighbors`, `address_families`,
`prefix_lists`, `route_maps`, and the toggles `no_ebgp_requires_policy`,
`no_default_ipv4_unicast`, `no_import_check`.

### Neighbor policy wrappers

In the default BGP instance, every activated neighbor and address family is
bound to two Routier route-maps, `ROUTIER-IN-<neighbor>-<family>` and
`ROUTIER-OUT-<neighbor>-<family>`, instead of the user's map. The neighbor
part replaces every character other than letters, digits and `-` with `_`, so
`192.0.2.1` in `ipv4-unicast` gives `ROUTIER-OUT-192_0_2_1-ipv4-unicast`. Each
wrapper is built in fixed slots:

| Sequence | Entry |
|----------|-------|
| 100 to 999 | Routier hooks that run before the user's rules |
| 1000 | `call` the user's map: the neighbor's `route_map_in`/`route_map_out`, else the address family's |
| 2000 to 8999 | Routier hooks that run after the user's rules |
| 9999 | the final entry |

- When the user has a map, a route the user's map denies is dropped, and the
  final `permit` is only reached by routes the user's map permits.
- When the user has no map for that direction, the wrapper is a single
  `deny 9999`: nothing is accepted or advertised without a user rule, for eBGP
  and iBGP alike.
- Every hook map ends with a catch-all permit, so a hook only changes the
  routes it matches. The VRRP BGP prepend (see [ha.md](ha.md)) is the first
  hook, running before the user's outbound rules.
- BGP instances in VRFs keep binding the user's maps directly.

Moving neighbors onto the wrappers rebinds their route-maps in one FRR reload.
FRR applies the change with a soft refresh, without resetting the sessions.

`allow_inbound` lists the interfaces on which BGP (TCP 179) is accepted by the
firewall. Acceptance is scoped to those interfaces and their subnets, so
enabling BGP does not open port 179 anywhere else (see
[firewall.md](firewall.md)).

## OSPF / OSPFv3

```yaml
routing:
  ospf:
    router_id: 192.0.2.1
    areas: [ ... ]
    passive_interfaces: [lan]
    redistribute: [connected, static]
    default_information_originate: true
    allow_inbound: [core]
```

`ospf6` is the IPv6 counterpart. Like BGP, `allow_inbound` scopes firewall
acceptance to the named interfaces.

## BFD

`routing.bfd` defines BFD profiles that BGP/OSPF neighbors reference for fast
failure detection. Enabling BFD adds the `bfdd` daemon to FRR; Routier
restarts FRR when the daemon set changes, so a newly enabled daemon actually
starts.

## Policy-based routing (PBR)

```yaml
routing:
  pbr:
    nexthop_groups: { ... }
    maps: { ... }
    policies: { ... }
```

- `nexthop_groups` - named groups of nexthops.
- `maps` - ordered match/action entries. Each entry matches on `match_src`
  and `match_dst`, and takes exactly one action: `set_nexthop`,
  `set_nexthop_group`, or `set_vrf`.
- `policies` - bind a map to an interface, by logical name. The map applies
  to traffic arriving on that interface; traffic the router sends itself, and
  traffic for the router's own addresses, is not affected.

Entries are evaluated by ascending `seq`, and the first match wins.
`set_vrf: unchanged` sends the matched traffic back to the interface's own
routing table, which is how a map excludes destinations from a broader rule.
`set_vrf` also accepts a VRF declared in `vrfs`.

A nexthop does not have to be directly connected: FRR resolves it through
the routing table. Pointing it at a remote router's loopback learned over
BGP makes the rule follow that router: when the BGP route goes away, the
rule's table empties and traffic falls back to the main table.

The following map sends one host's internet traffic out through the router
whose loopback is `198.51.100.11`, while every private destination keeps normal
routing:

```yaml
routing:
  pbr:
    maps:
      EGRESS:
        - { seq: 10, match_dst: 10.0.0.0/8, set_vrf: unchanged }
        - { seq: 11, match_dst: 100.64.0.0/10, set_vrf: unchanged }
        - { seq: 12, match_dst: 172.16.0.0/12, set_vrf: unchanged }
        - { seq: 13, match_dst: 192.168.0.0/16, set_vrf: unchanged }
        - { seq: 100, match_src: 192.0.2.56/32, set_nexthop: 198.51.100.11 }
    policies:
      public: EGRESS
```

The exit router must forward and NAT the steered traffic, and the router
receiving the replies must accept them on the tunnel: strict reverse-path
filtering (`rp_filter=1`) on the tunnel drops them, because its own route
back to the internet points elsewhere. The kernel applies the higher of the
`all` and per-interface values, and Routier creates tunnels after it applies
sysctls, so set loose mode through `all`:

```yaml
sysctl:
  net.ipv4.conf.all.rp_filter: "2"
```

UI: **Network > Routing**, with tabs for static, BGP, OSPF, PBR, and the rest.
The dashboard BGP card deep-links to the BGP neighbors tab.

## EVPN

BGP supports the `l2vpn-evpn` address family globally and inside a VRF.
Neighbors are activated through their existing `address_families` map. Typed
family options cover VNI advertisement, default-gateway and SVI advertisement,
IPv4/IPv6 unicast advertisement, and import/export route targets:

```yaml
routing:
  bgp:
    asn: 65001
    router_id: 192.0.2.1
    neighbors:
      - address: 192.0.2.2
        remote_asn: 65002
        address_families:
          l2vpn-evpn: {}
    address_families:
      l2vpn-evpn:
        advertise_all_vni: true
        advertise: [ipv4-unicast]
        route_target_import: [65001:100]
        route_target_export: [65001:100]
```

For a VRF with an `l3vni` (see [interfaces.md](interfaces.md#vrfs)), a BGP
instance in that VRF advertises its prefixes as EVPN type-5 routes; the
global instance keeps the neighbors:

```yaml
routing:
  vrfs:
    blue:
      bgp:
        asn: 65001
        router_id: 192.0.2.1
        address_families:
          ipv4-unicast:
            redistribute: [connected]
          l2vpn-evpn:
            advertise: [ipv4-unicast]
```

`import_vrf` leaks routes between BGP instances (`default` is the global one);
`import_vrf_route_map` filters what is imported, for example only a default
route into a VRF:

```yaml
routing:
  bgp:
    prefix_lists:
      DEFAULT: [{ seq: 10, action: permit, prefix: 0.0.0.0/0 }]
    route_maps:
      ONLY-DEFAULT: [{ seq: 10, action: permit, match: { ip address prefix-list: DEFAULT } }]
    address_families:
      ipv4-unicast: { networks: [0.0.0.0/0], import_vrf: [blue] }
  vrfs:
    blue:
      bgp:
        address_families:
          ipv4-unicast: { import_vrf: [default], import_vrf_route_map: ONLY-DEFAULT }
```

## Anycast and RADVD

- `routing.anycast` - advertise anycast prefixes (works with BGP).
- `routing.radvd` - IPv6 router advertisements per interface.

VRRP failover and conntrackd live under the `ha:` section (see
[high availability](ha.md)).
