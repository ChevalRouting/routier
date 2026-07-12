# Monitoring

Routier collects metrics and exposes live status. UI: **Monitor**, with tabs
for traffic, logs, HA status and neighbors; the dashboard summarizes the same
data.

## Stats

A background collector records interface, system, BGP, protocol, neighbor,
and route metrics into the local SQLite database.

- `GET /api/stats` - current snapshot.
- `GET /api/stats/history` - time series (used by the dashboard traffic
  charts).
- `GET /api/stats/neighbors`, `GET /api/stats/processes`.
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
    routes: 60
```

Stats history is kept for 30 days. The `routier-ui` daemon prunes older rows
hourly, and a crontab entry (`routier prune-stats`) does the same so cleanup
still runs when the UI is stopped. The database uses SQLite incremental
auto-vacuum, so space freed by pruning is returned to the filesystem.

## Routing state

- `GET /api/routing/learned` - learned routes (FRR).
- `GET /api/routing/routes` - kernel routes.
- `GET /api/routing/neighbors` - L2/L3 neighbors.
- The **Topology** view (`/topology`) visualizes the routing/interface graph.

## HA status

`GET /api/ha/status` returns VRRP instance roles (MASTER/BACKUP, peers) and
conntrackd state.

## Logs

- `GET /api/logs/stream` - live log stream.
- UI: **Monitor > Logs**.
