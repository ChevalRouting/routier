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
- `vlans` - VLAN subinterfaces (see below).
- `bridge` - turn the interface into a bridge (see below).
- `vrrp` - VRRP instances on this interface (see below).

API: `GET/PUT /api/v1/sessions/{id}/interfaces`, plus per-item routes for
get/put/delete. UI: **Network > Interfaces**.

## VLANs

`vlans` under an interface defines tagged subinterfaces, each with its own
addresses and MTU.

## Bridges

`bridge` turns an interface into a Linux bridge with `members` and optional
`stp`.

## VRFs

`vrfs` maps a VRF name to a routing `table` number. Interfaces reference a
VRF via their `vrf` field, and routing protocols can run per-VRF
(`routing.vrfs`).

```yaml
vrfs:
  blue:
    table: 100
```

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
