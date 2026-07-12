# Configuration

Routier is driven by one declarative YAML document. It can be a single file
(`/etc/routier/config.yml` by default) or a directory of `*.yml` / `*.yaml`
files merged together. Every run loads the config, validates it, renders
artifacts, and applies them; there is no other source of truth.

## Top-level sections

| Key | Purpose |
|-----|---------|
| `version` | config schema version (`vX.Y.Z`), required |
| `hostname` | system hostname |
| `vrfs` | VRF definitions |
| `interfaces` | physical/virtual interfaces, addresses, VLANs, bridges, VRRP |
| `tunnels` | IP tunnels |
| `routing` | static routes, BGP, OSPF, BFD, PBR |
| `wireguard` | WireGuard interfaces (incl. friend-derived) |
| `nftables` | firewall rules in the routier-owned table |
| `sysctl` | kernel sysctl keys |
| `users` | local users |
| `dns` | resolver configuration |
| `services` | managed services and their config templates |
| `logging` | logging configuration |
| `friends` | mutually-authenticated peer routers |
| `ssh` | SSH server settings |
| `boot_modules` | kernel modules to load at boot |
| `gai` | `gai.conf` address-selection policy |
| `monitoring` | stats collection intervals |

Each section is documented in its own page (see the [index](README.md)).

## Versioning and migrations

The config carries a `version` (`vX.Y.Z`). Routier compares the **major**
version with its own `CurrentVersion` and rejects a major mismatch.

Breaking schema changes bump `CurrentVersion` and ship a **migration** that
upgrades older documents. Migrations run automatically when a config is
loaded, so an older file keeps working without intervention. To rewrite files
on disk to the current schema:

```
routier migrate [config]        # file or directory
routier migrate --dry-run       # show what would change
```

For example, the v1 -> v2 migration converts firewall chain rules from a list
of lines into a single block string (preserving indentation).

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

API equivalents: `GET /api/config`, `GET /api/config/{section}`,
`PUT /api/config/{section}`, `GET /api/config/diff`, `POST /api/config/apply`,
`GET /api/apply/pending`, `POST /api/apply/confirm`, `GET /api/snapshots`,
`POST /api/snapshots/{id}/restore`, `GET/POST /api/backup/...`.

## Macros

Reusable, parameterized config fragments that can be previewed (diff) and
applied. See [macros.md](macros.md).
