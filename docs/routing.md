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
```

Fields per route: `destination` (required), `via`, `dev`, `metric`.

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
- `maps` - ordered match/action entries.
- `policies` - bind a map to an inbound match.

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

## Anycast and RADVD

- `routing.anycast` - advertise anycast prefixes (works with BGP).
- `routing.radvd` - IPv6 router advertisements per interface.

VRRP failover and conntrackd live under the `ha:` section (see
[high availability](ha.md)).
