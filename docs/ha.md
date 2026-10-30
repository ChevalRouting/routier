# High availability

The `ha:` section groups the features that keep a pair (or cluster) of routers
serving through a failure: VRRP for virtual-IP failover and conntrackd for
connection-state replication. DHCP failover lives in the DHCP section, see
[DHCP high availability](dhcp.md#high-availability).

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
- `track_interfaces` - interfaces whose link state affects the instance, by
  logical or device name; the config keeps the names as written.
- `virtual_routes` - routes installed while MASTER.
- `allow_inbound` - open the VRRP protocol on the instance's interface in the
  firewall.
- `bgp_prepend` - prepend the local AS on outbound BGP routes while the
  instance is not MASTER (see below).

The `vips(<name>)` selector (see [DNS](dns.md)) resolves to the VIPs whose
`interface` matches `<name>`.

### BGP prepend while not master

A backup router that keeps advertising the same routes as the master attracts
part of the return traffic. `bgp_prepend` makes it advertise a longer AS path
until it becomes MASTER, so peers prefer the master:

```yaml
ha:
  vrrp:
    - id: 51
      interface: lan
      vips: ["10.50.0.100/32"]
      bgp_prepend:
        count: 3                  # default 3, at most 10
        neighbors: [192.0.2.1]    # default: every eBGP neighbor
        prefixes: [203.0.113.0/24, 2001:db8:100::/48]  # default: every route
```

- Routier renders a `ROUTIER-VRRP-<id>` route-map that prepends the local AS
  `count` times. It runs as a hook in the outbound policy wrapper of each
  covered neighbor, before the user's rules (see
  [neighbor policy wrappers](routing.md#neighbor-policy-wrappers)). A neighbor
  without an outbound user map advertises nothing, prepend or not.
- The rendered `frr.conf` always carries the prepend, so a router that does not
  know its state advertises the longer path. When the instance is MASTER,
  Routier removes the `set as-path prepend` line from the running FRR config.
  The prepend applies to the default BGP instance only.
- `prefixes` limits the prepend to routes inside those CIDRs, the CIDR itself
  and any more-specific route. Routier renders them as the
  `ROUTIER-VRRP-<id>-V4` and `-V6` prefix-lists, matched by the prepending
  entries of `ROUTIER-VRRP-<id>`. A final plain permit lets every other route
  through unchanged.
- `neighbors` must name configured `routing.bgp.neighbors`. Without it, every
  neighbor whose `remote_asn` differs from the local ASN is covered.
- Requires `routing.bgp`.

### Transition hooks

Transition hooks reconcile toward the live VRRP state instead of replaying
keepalived events. An instance is MASTER on this router when one of its VIPs
is assigned to a local interface, and BACKUP otherwise. Each hook compares
that state with what is actually running and changes only what differs, so
hooks must be idempotent. The hooks live in `pkg/svc/vrrphooks.go`; a feature
that reacts to VRRP state adds one entry to the hook list. The BGP prepend is
the first hook: it reads the `ROUTIER-VRRP-<id>` entries from the bgpd running
config and adds or removes `set as-path prepend` where they disagree.

The reconcile is a one-shot job. `routier vrrp-sync` runs it from the package
crontab every 30 seconds, which converges after an FRR restart outside an
apply (`frr.conf` always carries the prepend), a notify that failed because
FRR was not ready, or overlapping notifies that finished out of order. The
same job also runs when keepalived calls `routier vrrp-notify` on a state
change (the reported state is only logged) and after every apply that reloads
FRR or keepalived, so a failover usually takes effect without waiting for the
next tick.

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
