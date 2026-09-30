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

```yaml
vrfs:
  blue:
    table: 100
```

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

## Tunnels

`tunnels` defines IP tunnels (GRE, IPIP, etc.).

```yaml
tunnels:
  gre0:
    mode: gre
    local: 203.0.113.1
    remote: 203.0.113.2
    addresses: ["10.255.0.1/30"]
    ttl: 64
    mtu: 1476
```

Fields: `mode` (required), `local`, `remote`, `ttl`, `addresses`, `mtu`.

UI: **Network > Tunnels**.
