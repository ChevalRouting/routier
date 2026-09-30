# High availability

The `ha:` section groups the features that keep a pair (or cluster) of routers
serving through a failure: VRRP for virtual-IP failover and conntrackd for
connection-state replication.

```yaml
ha:
  vrrp:
    - id: 51
      interface: lan
      vips: ["10.50.0.100/32"]
      priority: 200
  conntrackd:
    interface: lan
    address: 10.0.0.1
    peer_ips: ["10.0.0.3"]
```

API: `GET/PUT /api/v1/sessions/{id}/ha`. UI: **Network > HA**.

## VRRP

`ha.vrrp` is a list of VRRP instances, each keepalived-managed. Every instance
names the interface that hosts its virtual addresses:

```yaml
ha:
  vrrp:
    - id: 51
      interface: lan
      vips: ["10.50.0.100/32"]
      priority: 200
```

Fields:

- `id` - virtual router id, 1-255, unique across all instances (required).
- `interface` - the interface the VIPs live on, by logical name (required).
- `vips` - virtual IP addresses this instance owns (required).
- `transport` - advertise VRRP on a different interface, for example a dedicated
  sync link, while the VIPs stay on `interface`.
- `priority` - election priority, 1-254; highest wins (default 100).
- `password` - simple auth password, 8 characters or fewer.
- `track_interfaces` - interfaces whose link state affects the instance.
- `virtual_routes` - routes installed while MASTER.
- `switchover` - bring WireGuard down on BACKUP, up on MASTER (see
  [architecture](architecture.md)).
- `allow_inbound` - open the VRRP protocol on the instance's interface in the
  firewall.

The `vips(<name>)` selector (see [DNS](dns.md)) resolves to the VIPs whose
`interface` matches `<name>`.

## Conntrackd

`ha.conntrackd` replicates connection tracking state to the peer so established
flows survive a failover:

```yaml
ha:
  conntrackd:
    interface: lan
    address: 10.0.0.1
    peer_ips: ["10.0.0.3"]
    port: 3780
    allow_inbound: true
```

Fields:

- `interface` - interface carrying the replication traffic (required).
- `address` - this router's address on that interface (required).
- `peer_ips` - the peer addresses to replicate to.
- `port` - UDP port for replication (default 3780).
- `allow_inbound` - open the replication port on `interface` in the firewall.

## Friend-driven HA

When routers are paired as [friends](friends.md), the `vrrp` and `conntrackd`
sync sections push the matching `ha` config to the peer and apply it on both
sides. Live roles are at `GET /api/ha/status` (see [monitoring](monitoring.md)).
