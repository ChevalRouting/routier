# Monitoring

Routier collects metrics and exposes live status. In the UI, the
**Monitoring** sidebar group has two pages: **System** (resources, processes,
logs) and **Network** (topology, traffic, flows, BGP, neighbors, probes). The
dashboard summarizes the same data.

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

## Live traffic

**Monitoring > Network > Live Traffic** shows the throughput of one interface
as it happens, in the manner of `nload`. Incoming and outgoing traffic each get
a bar graph of the last 60 seconds, with the current, average, minimum and
maximum rate over that window, the current packet rate, and the total bytes
the kernel has counted since the interface was created.

`GET /api/stats/live?iface=<name>` streams one sample per second as server-sent
events. Each sample carries the byte and packet rates computed from the
interface counters in `/sys/class/net/<name>/statistics`, the counter totals,
and the operational state. Nothing is recorded into the stats database.

## Connectivity probes

Scheduled ICMP checks record reachability and average round-trip time for a
target. Configure them under **Monitoring > Network > Probes**; the dashboard shows their
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
**Monitoring > Network > Neighbors** section shows these alongside the ARP/NDP tables, and
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

## Traffic flows (sFlow)

Routier samples forwarded and incoming packets, exports them as sFlow v5 to
any number of collectors, such as an [akvorado](https://github.com/akvorado/akvorado)
inlet, and keeps a bounded history of the resulting flows in the local SQLite
database. **Monitoring > Network > Flows** draws that history as a Sankey
diagram (source, service and destination, or the interfaces traffic crosses)
and lists top talkers, services and per-interface traffic.

Flow sampling is off by default:

```yaml
monitoring:
  flows:
    enabled: true
    sampling: 1000
    interfaces: [wan, lan]
    source: 192.0.2.1
    collectors:
      - address: 198.51.100.10
        port: 6343
    retention:
      hours: 72
      max_flows: 200000
```

- `sampling` samples one packet in N (default 1000). Lower values are more
  accurate and cost more CPU. Each sample carries the rate, so collectors scale
  bytes and packets back up.
- `interfaces` lists configured interfaces whose ingress traffic is sampled.
  When omitted, every configured interface and WireGuard tunnel is sampled.
- `source` is the address datagrams are sent from and announced as the sFlow
  agent. When omitted, the kernel picks the address of the route towards each
  collector.
- `collectors` receive the sFlow datagrams over UDP (default port 6343). The
  list can be empty to keep flows only locally.
- `retention` bounds the local history: flows older than `hours` (default 72)
  are deleted, and only the newest `max_flows` (default 200000) are kept.

Sampling is a managed nftables rule in its own `routier_flows` table. It logs
one packet in N, before the filter chains, to netlink group 6343; the
`routier-ui` daemon reads those packets, sends each sample as an sFlow flow
sample with the input and output interface indexes, and sends interface counter
samples every 20 seconds. It also groups samples into flows by
interface, protocol, addresses and ports, closing a flow after 15 seconds
without packets or 60 seconds after it started, and stores each flow with its
estimated bytes and packets.

The API:

- `GET /api/flows/status` - agent state, sampling, collectors and counters.
- `GET /api/flows?minutes=60&address=192.0.2.10` - recent flows.
- `GET /api/flows/top?dimension=src&minutes=60` - totals by `src`, `dst`,
  `service`, `protocol`, `in_if`, `out_if` or `conversation`.
- `GET /api/flows/series?minutes=60` - bytes per input interface over time.
- `GET /api/flows/sankey?dimensions=src,service,dst&minutes=60` - the largest
  paths across two or three of the `top` dimensions, except `conversation`,
  with the total bytes of the window.
- `GET /api/flows/metadata` - this router's interfaces for akvorado.

### Sending flows to akvorado

Add the akvorado inlet as a collector; its default configuration already
listens for sFlow on UDP 6343. Akvorado drops flows from interfaces it cannot
name, so let it read interface names and speeds from Routier: create an API
key in Routier and add a static provider source to the outlet configuration
(`outlet.yaml` in the docker setup, without the `outlet:` level):

```yaml
outlet:
  metadata:
    providers:
      - type: static
        exporter-sources:
          routier-rt1:
            url: https://rt1.example.net/api/flows/metadata
            headers:
              Authorization: Bearer rtr_api_...
            interval: 10m
            transform: .result.exporters[]
```

The metadata lists the router once per address it sends sFlow from, with
every kernel interface index, the device name, the Routier interface name as
the description, and the link speed in Mbps.

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

When a local DNS server is configured (see [dns.md](dns.md)), the **DNS** page
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
- UI: **Monitoring > System > Logs**.

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
