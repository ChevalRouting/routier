# Architecture

Routier turns one declarative YAML document into a running router. Whether a
change comes from the CLI, the web UI, or the REST API, it flows through the
same pipeline.

## The pipeline

```
config.yml ─▶ load + validate ─▶ resolve ─▶ render ─▶ apply ─▶ verify/rollback
```

1. **Load and validate** (`pkg/config`). The YAML is parsed into typed Go
   structs and validated: referential integrity (a subnet's interface must
   exist, a route map referenced by a neighbor must be defined), value checks
   (IPs, CIDRs, ASNs, ports), and cross-section rules.
2. **Resolve** (`pkg/config`, `pkg/friends`). Interface selectors
   (`prefix[idx]`, `mac(...)`) are resolved to real device names, and
   friend-published variables are interpolated into the config.
3. **Render** (`pkg/render`). The config becomes concrete artifacts: FRR
   configuration, an nftables ruleset, wg-quick files, Kea DHCP server
   configs, the BIND configuration and its zone files, keepalived, radvd,
   lldpd, conntrackd, sshd, sysctl and more. Rendering is pure: no system state is
   touched, which is what makes `routier plan` safe to run anywhere.
4. **Apply** (`pkg/apply`, `pkg/managers`, `pkg/svc`). Artifacts are written
   and services reloaded under a global apply lock (an in-process mutex plus a
   file lock, so the CLI and the daemon cannot race each other).
5. **Verify or roll back**. Interactive applies arm a watchdog; if the
   operator does not confirm within the timeout, the previous state is
   restored.

## Apply, in order

`managers.ApplyConfig` performs the steps below. The ordering is deliberate:

1. Hostname, users and `/etc/resolv.conf` are applied first.
2. Every artifact whose content changed is written; before the first write, a
   **snapshot** of the previous file contents (plus the last-applied config)
   is saved under `/var/lib/routier/snapshots/`.
3. **Netlink reconcile** creates and configures links, VLANs, bridges, VRFs,
   addresses and routes. It runs before sysctl so per-interface sysctl keys
   have an interface to attach to.
4. **sysctl and nftables** load next, before any routing daemon starts, so
   traffic is never forwarded through a box whose firewall is not in place.
   The ruleset is validated with `nft -c` before it is loaded.
5. **FRR** is validated (`vtysh -C`), then reloaded in place when possible,
   and restarted when the daemon set changed.
6. keepalived, radvd, lldpd, conntrackd, sshd and WireGuard interfaces follow.
   Managed WireGuard interfaces that left the config are torn down with
   `wg-quick down` so their PostDown hooks run.
7. **BIND** comes next, over its rndc control channel: a per-zone reload when
   only zone data changed, `rndc reconfig` when the configuration changed, and
   a service reload or restart only as a fallback. Zone files are validated
   with `named-checkzone` before they are written and the rendered `named.conf`
   with `named-checkconf` after, since it references the zone files by absolute
   path. See [dns.md](dns.md).
8. **Kea DHCP** is the last service reloaded, via `config-reload` on its
   control socket so active leases survive; a failed reload falls back to a
   restart. Validation failures abort the apply like any other service. It runs
   after BIND so the resolver is answering before DHCP starts handing out
   its address to clients.
9. Services that are no longer needed are stopped and removed from the boot
   runlevel; newly needed ones are started and enabled.

Any fatal error rolls the snapshot back: previous
file contents are restored, affected services reloaded again, and netlink is
reconciled against the **previous** config. That previous config is read from
the restored `/var/lib/routier/last-applied.yml`, not from the current
`config.yml`, which may already hold the new (broken) config.

## Watchdog

An apply from the web UI or API arms a pending state with a timeout. The
`routier watchdog` cron job rolls back automatically if the operator never
confirms, so it is safe to apply a config that might cut off remote access: if
it does, the previous state is restored after the timeout. Confirming
(`routier confirm` / `POST /api/apply/confirm`) clears the pending state and
commits.

## Staging and promotion

The web UI edits a **per-user staging copy** of the config; changes take
effect only when applied. Apply validates the staged config, applies
it, and only then promotes it to `/etc/routier/config.yml`. A failed apply
leaves the committed file untouched. The staging copy survives until the
apply is confirmed: if the watchdog rolls back instead, the committed file
reverts, the staged changes show up as pending again, and the apply can be
retried. The REST API v1 uses sessions with the
same property: the committed config is always the single YAML file, and only a
successful apply writes it.

## Boot

At boot, `routier boot` renders and reloads everything, with no watchdog. When
a switchover VRRP instance is configured, WireGuard initialization is skipped
so a standby node does not bring tunnels up before it becomes master;
keepalived raises the tunnels on the MASTER transition instead.

## Repository layout

| Path | Contents |
|------|----------|
| `cmd/routier` | CLI and daemon entry point |
| `cmd/routier-ui` | Web UI server entry point |
| `pkg/config` | Config types, loading, validation, selectors |
| `pkg/render` | Template rendering of all artifacts |
| `pkg/apply` | File writing, snapshots, rollback |
| `pkg/managers` | Apply orchestration, apply lock, pending/watchdog, rollback, boot, friend sync |
| `pkg/svc` | Service control, reload ordering, service reconciliation |
| `pkg/netlink` | Link/address/route reconciliation |
| `pkg/api` | HTTP APIs (web/UI and REST v1), embedded web app |
| `pkg/friends` | Peer router protocol (poll, pairing, variables) |
| `pkg/kea` | Kea DHCP control-socket client |
| `pkg/bind` | BIND rndc client and runtime inspection |
| `pkg/db` | SQLite storage, embedded migrations and the sqlc-backed persistence facade |
| `web/` | React/TypeScript frontend |
| `tests/` | Integration tests with a mocked service runner |
