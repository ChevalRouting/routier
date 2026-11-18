# Configuration

Routier is driven by one declarative YAML file (`/etc/routier/config.yml` by
default). Every run loads the config, validates it, renders artifacts, and
applies them; there is no other source of truth.

## Top-level sections

| Key | Purpose |
|-----|---------|
| `version` | config schema version (`vX.Y.Z`), required |
| `hostname` | system hostname |
| `vrfs` | VRF definitions |
| `interfaces` | physical/virtual interfaces, addresses, VLANs, bridges |
| `vpn` | WireGuard interfaces |
| `routing` | static routes, BGP, OSPF, BFD, PBR |
| `ha` | high availability: VRRP and conntrackd |
| `dhcp` | Kea DHCP subnets, reservations, DDNS and HA ([dhcp.md](dhcp.md)) |
| `nftables` | firewall rules in the routier-owned table |
| `sysctl` | kernel sysctl keys |
| `users` | local users |
| `dns` | resolver configuration and the local DNS server |
| `services` | managed services and their config templates |
| `logging` | logging configuration |
| `friends` | mutually-authenticated peer routers |
| `ssh` | SSH server settings |
| `boot_modules` | kernel modules to load at boot |
| `gai` | `gai.conf` address-selection policy |
| `monitoring` | stats collection, probes, LLDP, flows and the Prometheus exporter ([monitoring.md](monitoring.md)) |
| `switches` | managed switches: port VLANs and bonds |
| `web` | web UI certificate: self-signed, files, or ACME ([web-tls.md](web-tls.md)) |

Each section is documented in its own page (see the [index](README.md)).

## Automatic firewall rules

By default, Routier adds service allow rules when their `allow_inbound` settings
are enabled. To suppress all of these automatic rules in advanced YAML, set:

```yaml
nftables:
  disable_auto_rules: true
```

This overrides service `allow_inbound` settings for WireGuard, DNS, BGP, OSPF,
OSPF6, VRRP, conntrackd, Kea HA and the Prometheus exporter. User rules, includes, defines, and chain policies
still apply; add any required service allow rules yourself. Omit the option or
set it to `false` to keep automatic rule creation enabled.

## Versioning

The config carries a `version` (`vX.Y.Z`). Routier compares the **major**
version with its own `CurrentVersion` (`v5.0.0`) and rejects a major mismatch.

## Validation

`routier validate [config]` loads and validates without applying. The same
validation runs before every apply, so a failing config is never applied.

## Staging, apply, and rollback

The web UI and API edit a **per-user staging copy** of the config; nothing
takes effect until applied. The CLI edits the live file directly.

- `routier plan [config]` - show what would change.
- `routier apply [config]` - render + apply with a snapshot and an **armed
  rollback**: if you do not confirm within the timeout, the previous state is
  restored automatically. This protects against locking yourself out of a
  remote box.
- `routier confirm` - confirm a pending apply (disarm the rollback).
- `routier rollback` - manually roll back to the previous snapshot.
- `routier snapshots` / `routier history` - list snapshots and apply history.
- `routier backups` / `routier restore` - export/import the full config.
  A backup also carries the files the config references, such as nftables
  includes and WireGuard key files. A restore writes the config to the
  router's own config path and extracts only files under its directory
  (`/etc/routier`); files elsewhere in the archive are skipped with a warning
  and must be copied by hand.

API equivalents: `GET /api/config`, `GET /api/config/{section}`,
`PUT /api/config/{section}`, `GET /api/config/diff`, `POST /api/config/apply`,
`GET /api/apply/pending`, `POST /api/apply/confirm`, `GET /api/snapshots`,
`POST /api/snapshots/{id}/restore`, `GET/POST /api/backup/...`.

## Macros

Reusable, parameterized config fragments that can be previewed (diff) and
applied. See [macros.md](macros.md).
