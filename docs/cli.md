# CLI reference

`routier` is the command-line surface. Every web UI / API capability has a
CLI equivalent. Most commands take an optional `[config]` path argument
(default `/etc/routier/config.yml`); database-backed commands take `--db`
(default `/var/lib/routier/web.db`).

## Config lifecycle

| Command | Purpose |
|---------|---------|
| `routier validate [config]` | validate without applying |
| `routier plan [config]` | show what would change |
| `routier apply [config]` | render + apply with armed rollback |
| `routier confirm` | confirm a pending apply (disarm rollback) |
| `routier rollback` | roll back to the previous snapshot |
| `routier migrate [config]` | migrate config file(s) to the current schema (`--dry-run`) |
| `routier snapshots` | list snapshots |
| `routier history` | apply history |
| `routier backups [config]` / `routier restore <archive>` | export / import full config |
| `routier edit` | edit the config |

## Friends

| Command | Purpose |
|---------|---------|
| `routier friends list` | list configured friends |
| `routier friends preview --url --token` | fetch a prospective friend's identity |
| `routier friends add --name --url --token [--fingerprint\|--yes]` | add (TOFU) and pair |
| `routier friends update <name> [--url --token --tls-skip --enabled]` | edit a friend |
| `routier friends remove <name> [--yes]` | remove (cascades tagged sections) |
| `routier friends pair <name>` | re-run the pairing handshake |
| `routier friends interfaces <name>` | list a friend's interfaces/IPs |
| `routier friends wireguard <name> --subnet --local-iface --friend-iface` | derive a tunnel |

Friend commands accept `--config` (config path) and `--identity` (ed25519 key
path, default `/var/lib/routier/identity_ed25519`).

## HA and macros

- `routier switchover` / `routier activate` - VRRP BACKUP/MASTER hooks.
- `routier macros list|show|apply|delete` - manage macros.

## IP tools

- `routier ipcalc subnet <address|cidr>` - ipcalc-style subnet breakdown.
- `routier ipcalc reverse <address|cidr>` - reverse-DNS (PTR) name and zone.
- `routier ipcalc range <start> <end>` - split an address range into CIDRs.

See [tools.md](tools.md).

## Operations

| Command | Purpose |
|---------|---------|
| `routier collect-stats --db --config` | collect metrics (run periodically) |
| `routier prune-stats --db` | delete stats history past the 30-day window and reclaim space |
| `routier watchdog` | apply-confirmation watchdog |
| `routier boot` / `routier firstboot` | boot-time setup |
| `routier anycast` | anycast helpers |
| `routier version` | print version |

Run `routier <command> --help` for the full flags of any command.
