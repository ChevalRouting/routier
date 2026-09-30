# Monitoring

Routier collects metrics and exposes live status. UI: **Monitor**, with tabs
for traffic, logs, HA status, DNS and neighbors; the dashboard summarizes the
same data.

## Stats

A background collector records interface, system, BGP, protocol, neighbor,
LLDP/CDP, and route metrics into the local SQLite database.

- `GET /api/stats` - current snapshot.
- `GET /api/stats/history` - time series (used by the dashboard traffic
  charts).
- `GET /api/stats/neighbors`, `GET /api/stats/lldp`, `GET /api/stats/processes`.
- CLI: `routier collect-stats --db ... --config ...` (run periodically).

Collection intervals are configurable, in seconds:

```yaml
monitoring:
  collection:
    iface: 5
    system: 10
    bgp: 30
    proto: 30
    neighbors: 30
    lldp: 300
    routes: 60
```

Stats history is kept for 30 days. The `routier-ui` daemon prunes older rows
hourly, and a crontab entry (`routier prune-stats`) does the same so cleanup
still runs when the UI is stopped. The database uses SQLite incremental
auto-vacuum, so space freed by pruning is returned to the filesystem.

## Connectivity probes

Scheduled ICMP checks record reachability and average round-trip time for a
target. Configure them under **Monitor > Probes**; the dashboard shows their
latest status and history read-only.

```yaml
monitoring:
  probes:
    - name: uplink
      target: 1.1.1.1
      interval: 300
      timeout: 5000
```

`interval` is in seconds and `timeout` in milliseconds. Probe samples are
recorded into the stats database and served by `GET /api/stats/history` under
the `probes` series.

## Link-layer discovery (LLDP/CDP)

Routier can run the `lldpd` daemon to discover directly connected switches and
routers over LLDP (802.1AB) and Cisco's CDP. Each neighbor reports the local
interface, remote chassis name and management IP, remote port, and VLAN. The
**Monitor > Neighbors** tab shows these alongside the ARP/NDP tables, and
`GET /api/routing/lldp` returns the live view.

Discovery is off by default and enabled per configuration:

```yaml
monitoring:
  lldp:
    enabled: true
    cdp: true
    transmit: false
    interfaces:
      - eth1
      - eth2
```

- `cdp` also receives (and, with `transmit`, sends) CDP frames in addition to
  LLDP.
- `transmit` advertises this router to its neighbors. When false, `lldpd` runs
  in receive-only mode and sends nothing. When enabled, the router advertises a
  system description of `Routier` and uses its configured hostname as the
  system name, rather than the default kernel `Linux` values.
- `interfaces` restricts discovery to the listed interfaces. When omitted,
  `lldpd` runs on all interfaces.

Discovered neighbors are also recorded into the stats database on the `lldp`
collection interval and served by `GET /api/stats/lldp`.

## Routing state

- `GET /api/routing/learned` - learned routes (FRR).
- `GET /api/routing/routes` - kernel routes.
- `GET /api/routing/neighbors` - L2/L3 neighbors (ARP/NDP).
- `GET /api/routing/lldp` - LLDP/CDP link-layer neighbors.
- The **Topology** view (`/topology`) visualizes the routing/interface graph.

## HA status

`GET /api/ha/status` returns VRRP instance roles (MASTER/BACKUP, peers) and
conntrackd state.

## DNS

When a local DNS server is configured (see [dns.md](dns.md)), **Monitor > DNS**
exposes the resolver's live state:

- `GET /api/dns` - resolver state (running, listen addresses, upstreams),
  forward zones, and the authoritative zones with their per-zone serial and
  record count.
- `GET /api/dns/stats` - key statistics: queries, cache hit ratio, SERVFAIL
  rate, request-list depth.
- `GET /api/dns/queries/stream` - live query log (requires
  `dns.server.log_queries`).

All of it is read live from the daemon's control socket; unlike interface and
routing metrics, DNS statistics are not recorded into the stats database, so
there is no DNS history.

## Logs

- `GET /api/logs/stream` - live log stream.
- UI: **Monitor > Logs**.

## Interface and bond status

Under **Monitor → System**, click an interface in the Network Interfaces stats
table to open its live status in a modal. Close it to select another interface
from the stats table. The modal shows kernel link state, flags,
MTU, master/parent relationships, counters, and type-specific attributes such
as VLAN IDs, bridge settings, and tunnel parameters. It uses the equivalent of
`ip -j -d -s link show dev <name>` and includes raw link details. All interface
types in the stats table are supported. Details refresh every five seconds
while open; a failed refresh displays an error alongside the last snapshot.

Bond interface details also show member link failures and LACP aggregator and churn state from
`/proc/net/bonding`. A bond is degraded if a member link is down, LACP churn is
reported, or a member is outside the active aggregator. Historical failure
counts alone do not mark a currently healthy bond as degraded.

`GET /api/system/interfaces/{name}` returns link and optional bond details.
`GET /api/system/bonds` returns all bond status through the authenticated
API. Systems without bonding information return no bonds.

## Live DNS queries

Open the DNS page's query view after enabling query logging in Resolver
settings. The view retains up to 500 queries and supports filtering, expanding
query details, pausing, and clearing the table. The first connection reads the
last 50 named log lines, then follows the log across rotation. Resuming and
reconnecting follow new lines only; queries during a disconnection are not
backfilled. Connection errors are displayed and retried every three seconds.

`GET /api/dns/queries/stream` emits parsed queries as server-sent events. Pass
`history=0` to skip the initial log lines. Comment frames establish the
connection immediately and provide a heartbeat every 15 seconds.
