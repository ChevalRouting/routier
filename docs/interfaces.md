# Interfaces, VLANs, bridges, VRFs

## Interfaces

`interfaces` maps a logical name to an interface definition. The physical
device is chosen by a `select` expression (by name, index, or MAC), so the
same config remains portable across hardware: `wan` stays `wan` even when the
NIC becomes `eth2` on the next box.

```yaml
interfaces:
  wan:
    select: name=eth0
    addresses: [dhcp]
    mtu: 1500
  lan:
    select: name=eth1
    addresses: ["192.168.1.1/24", "2001:db8::1/64"]
```

Fields:

- `select` - how to match the physical NIC (required).
- `addresses` - list of CIDRs, or the keywords `dhcp`, `dhcp4`, `dhcp6`,
  `slaac`. Static CIDRs feed the firewall `$me`/`$<iface>_network` variables.
- `dhcp_options` - extra DHCP client options.
- `mtu` - interface MTU.
- `vrf` - place the interface in a named VRF.
- `vlan` - tag the interface as a VLAN subinterface (see below).
- `bridge` - turn the interface into a bridge (see below).
- `bond` - aggregate several NICs into a bond (see below).

VRRP failover groups live under `routing.vrrp` and reference an interface by
name (see [routing](routing.md)).

API: `GET/PUT /api/v1/sessions/{id}/interfaces`, plus per-item routes for
get/put/delete. UI: **Network > Interfaces**.

## VLANs

A VLAN is its own interface with `type: vlan`. `select` names the parent
interface (by its logical name) and `vlan.id` is the tag. It carries its own
`addresses`, `mtu`, and `vrf` like any other interface, and its logical name
becomes the device name:

```yaml
interfaces:
  lan:
    select: name=eth1
  servers:
    type: vlan
    select: lan
    addresses: ["10.0.0.1/24"]
    vlan:
      id: 100
```

Fields:

- `select` - parent interface, referenced by its logical name (required).
- `vlan.id` - VLAN tag, 1-4094 (required).

## Bridges

`bridge` turns an interface into a Linux bridge with `members` and optional
`stp`.

When the `br_netfilter` module is loaded (for example on a host that also runs
Docker), Linux sends bridged frames through the IP firewall, where the forward
policy drops them. With a `bridge`, `vxlan-bridge` or `vnet` configured, Routier
sets `net.bridge.bridge-nf-call-iptables`, `-ip6tables` and `-arptables` to 0,
unless `sysctl` sets them.

## Bonds

A bond aggregates several physical NICs into one logical interface for
redundancy or throughput. Declare it as a virtual interface with `type: bond`
and list the member interfaces by their logical names:

```yaml
interfaces:
  eth0:
    select: name=eth0
  eth1:
    select: name=eth1
  bond0:
    type: bond
    addresses: ["10.0.0.1/24"]
    bond:
      mode: 802.3ad
      members: [eth0, eth1]
      miimon: 100
      xmit_hash_policy: layer3+4
      lacp_rate: fast
```

Fields:

- `members` - member interfaces, referenced by logical name (required).
- `mode` - `balance-rr` (default), `active-backup`, `balance-xor`,
  `broadcast`, `802.3ad` (LACP), `balance-tlb`, or `balance-alb`.
- `miimon` - link monitoring interval in milliseconds.
- `xmit_hash_policy` - transmit hash for `balance-xor` and `802.3ad`:
  `layer2`, `layer2+3`, `layer3+4`, `encap2+3`, `encap3+4`, or `vlan+srcmac`.
- `lacp_rate` - `slow` or `fast`, `802.3ad` only.
- `updelay` / `downdelay` - milliseconds to wait before enabling or disabling a
  member after a link change; both require `miimon`.
- `min_links` - minimum active members for the bond to carry traffic
  (`802.3ad` only).
- `primary` - preferred member for `active-backup`, `balance-tlb`, and
  `balance-alb`.

## VRFs

`vrfs` maps a VRF name to a routing `table` number. Interfaces reference a
VRF via their `vrf` field, and routing protocols can run per-VRF
(`routing.vrfs`).

Routier keeps VRFs apart: each VRF table ends with an `unreachable default`
(IPv4 and IPv6, metric 4278198272), and the `local` rule moves from priority 0
to 32765, after the VRF rule, so a host in a VRF cannot reach the router's
addresses in other tables. When Kea serves an interface that is in a VRF,
Routier sets `net.ipv4.ip_nonlocal_bind` and `net.ipv6.ip_nonlocal_bind` to 1
(unless `sysctl` sets them) so Kea can bind that interface's address.

```yaml
vrfs:
  blue:
    table: 100
```

A VRF shared across EVPN sites gets an `l3vni`: the VNI that carries its
routed traffic between VTEPs, the `vxlan-bridge` it uses, and an internal
transport VLAN on that bridge (unique per bridge, like VNet VLANs):

```yaml
vrfs:
  blue:
    table: 100
    l3vni:
      vni: 5000
      transport: fabric
      vlan: 3999
```

Routier creates the SVI (a VLAN device on the transport bridge, in the VRF),
maps the VLAN to the VNI on the bridge's VXLAN device, and declares the VNI in
FRR (`vrf blue` / `vni 5000`). Advertise the VRF's prefixes with a BGP instance
in the VRF (see [routing.md](routing.md#evpn)).

## VXLAN

A VXLAN is declared as a virtual interface and can be used anywhere an interface
name is accepted, including as a bridge member:

```yaml
interfaces:
  vxlan100:
    type: vxlan
    mtu: 1450
    vxlan:
      vni: 100
      local: 192.0.2.1
      vtep: underlay
      port: 4789
      learning: false
  br100:
    type: bridge
    bridge:
      members: [vxlan100, tenant100]
```

`vni` is required and must be between 1 and 16777215. `local` selects the
source VTEP address. `vtep` selects the underlay interface by logical or device
name. Use either `remote` for a unicast flood endpoint or `group` for a multicast
group. The UDP port defaults to 4789 and MAC learning defaults to true; EVPN
fabrics normally set `learning: false`.

## VXLAN bridges and VNets

Use `type: vxlan-bridge` for a shared EVPN transport and `type: vnet` for
networks that need an IP address or local bridge members:

```yaml
interfaces:
  tenant_port:
    select: eth1
    mtu: 1450
  fabric:
    type: vxlan-bridge
    mtu: 1450
    vxlan:
      local: 192.0.2.10
      port: 4789
  servers:
    type: vnet
    select: fabric
    mtu: 1450
    vnet:
      vlan: 200
      vni: 10200
    addresses: ["10.20.0.1/24"]
    bridge:
      members: [tenant_port]
```

`fabric` is a VLAN-aware Linux bridge with an automatically named SVD.
`servers` is an untagged Linux bridge. Routier creates a managed veth pair
between them, with its transport-side port in VLAN 200. This makes local
MACs visible in the transport bridge's VLAN FDB for FRR EVPN advertisement.
VM TAP interfaces can attach directly to `servers` without a VLAN tag.
Apply preserves externally attached, unmanaged TAP interfaces.

VNet `select` must reference a `vxlan-bridge`. `vnet.vlan` is an internal
transport VLAN (1-4094); `vnet.vni` is the VXLAN identifier (1-16777215).
VLANs must be unique per transport and VNIs must be unique across VNets.
A VNet supports normal `addresses`, `vrf`, `mtu`, and optional
`bridge.members`. Its MTU inherits the transport MTU if omitted and cannot
exceed it. A VNet VRF controls its local IP interface; this feature does not
automatically configure an EVPN L3VNI or inter-VNI routing.

The transport requires an explicit overlay `mtu` and a locally assigned,
reachable `vxlan.local` address.
Optional `vxlan.vtep` selects the underlay interface. Routier forces external
metadata mode and disables SVD MAC/VTEP learning; remote endpoints are
installed by FRR, with no static peers. Enable `advertise_all_vni` in the
BGP EVPN configuration. Addresses and VRFs belong on VNets, not the transport.
Each transport claims its UDP port, so do not share that port with another
VXLAN device. Generated devices are removed when their configuration is
removed, and obsolete VLAN-to-VNI mappings are reconciled on apply.

### Anycast gateway

When a VNet is stretched over several sites, `vnet.gateway` gives every site
the same gateway address and MAC, so hosts use their local routier as
gateway wherever they are:

```yaml
  servers:
    type: vnet
    select: fabric
    vnet:
      vlan: 200
      vni: 10200
      gateway:
        ips: [10.20.0.1, "fd20::1"]
    addresses: ["10.20.0.2/24", "fd20::2/64"]
```

Each routier keeps its own address in `addresses` (use a different one per
site) and puts the gateway IPs on a generated macvlan of the VNet bridge, as
host addresses (`/32`, `/128`, without duplicate address detection). The MAC is
`02:cb` followed by the VNI and `01` unless `gateway.mac` sets another
unicast address; use the same one on every site. A `bridge routier_anycast`
nftables table drops frames from or to that MAC and ARP requests for the
gateway at the transport port, so the gateway never crosses the fabric and EVPN
never advertises it. The VNet bridge and the gateway answer ARP only for their
own addresses (`arp_ignore=1`), and the gateway uses loose reverse path
filtering because replies leave through the bridge. Hand the gateway address
to hosts as their router, for example in the DHCP subnet `gateway`.

For an underlay MTU of 1500, use an overlay MTU of 1450 with IPv4 VTEPs or
1430 with IPv6 VTEPs. IPv6 VTEPs also require compatible FRR versions on all
EVPN participants. Set attached physical and guest interfaces accordingly.

## Tunnels

Interfaces with `type: tunnel` define IP tunnels (GRE, IPIP, etc.).

```yaml
interfaces:
  gre-office:
    type: tunnel
    mode: gre
    local: 203.0.113.1
    remote: 203.0.113.2
    addresses: ["10.255.0.1/30"]
    ttl: 64
    mtu: 1476
```

Fields: `mode` (required), `local`, `remote`, `ttl`, `addresses`, `mtu`.

UI: **Network > Interfaces**.
