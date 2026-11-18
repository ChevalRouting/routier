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
| `routier diff [config] [--json] [--exit-code]` | show what would change to converge the live system to the config |
| `routier apply [config]` | render + apply with armed rollback |
| `routier confirm` | confirm a pending apply (disarm rollback) |
| `routier rollback [-s <id>] [-n <offset>]` | roll back to the previous snapshot, or a given one |
| `routier snapshots list` | list snapshots |
| `routier history list` / `routier history show <id>` | apply history and the full log of one run |
| `routier failures list\|show <id>\|export <id>\|rm <id>` | rendered artifacts preserved from failed applies |
| `routier backups [config]` / `routier restore <archive>` | export / import full config |
| `routier edit [config]` | edit the config |
| `routier init [-d dir]` | write a starter config and the default templates |

## Friends

| Command | Purpose |
|---------|---------|
| `routier friends list` | list configured friends |
| `routier friends check <name>` | force a health check and info refresh |
| `routier friends config <name>` | fetch and print a friend's live config |
| `routier friends sync [name]` | push HA (VRRP and conntrackd) config to friends and apply locally |
| `routier friends preview --url --token` | fetch a prospective friend's identity |
| `routier friends add --name --url --token [--fingerprint\|--yes]` | add (TOFU) and pair |
| `routier friends update <name> [--url --token --tls-skip --enabled]` | edit a friend |
| `routier friends remove <name> [--yes]` | remove (cascades tagged sections) |
| `routier friends pair <name>` | re-run the pairing handshake |
| `routier friends interfaces <name>` | list a friend's interfaces/IPs |
| `routier friends wireguard <name> --subnet --local-iface --friend-iface` | derive a tunnel |

Friend commands accept `--config` (config path) and `--identity` (ed25519 key
path, default `/var/lib/routier/identity_ed25519`).

## DNS

| Command | Purpose |
|---------|---------|
| `routier dns` | mode, recursion, listen addresses, upstreams and zones |
| `routier dns stats` | key BIND counters |
| `routier dns zones [name]` | declared records, serial, and whether the zone is answering |
| `routier dns query <name> [type]` | resolve through the local resolver |
| `routier dns flush [name]` | flush the cache, or one name within it |
| `routier dns reload [zone]` | reconfigure named, or reload one zone |

See [dns.md](dns.md).

## DHCP

| Command | Purpose |
|---------|---------|
| `routier dhcp leases` | list active leases |
| `routier dhcp leases clear <ip>` | delete an active lease |
| `routier dhcp reservations` | list host reservations |
| `routier dhcp reservations add <hostname> <mac-or-duid> <ip-or-cidr>` | reserve an address (picked automatically when only a CIDR is given) |
| `routier dhcp reservations del <ip>` | delete a reservation |
| `routier dhcp reservations persist <ip>` | promote an active lease to a reservation |

See [dhcp.md](dhcp.md).

## HA and macros

- `routier vrrp-notify TYPE NAME STATE [PRIORITY]` - reconcile the VRRP transition hooks with the live VIP state (called by keepalived).
- `routier vrrp-sync` - the same reconcile without a keepalived event (run from cron every 30 seconds).
- `routier macros list|show|apply|delete` - manage macros.

## NAT

| Command | Purpose |
|---------|---------|
| `routier nat list [config]` | list NAT shortcut rules |
| `routier nat masquerade [config] --out [--source --family --comment]` | add a masquerade rule |
| `routier nat snat [config] --out --to [--source --family --comment]` | add a source-NAT rule |
| `routier nat dnat [config] --in --dport --to [--proto --comment]` | add an inbound port forward |
| `routier nat rm [config] --index <n>` | remove a rule by its `nat list` index |

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
| `routier anycast run [config]` | run one anycast health check cycle and update BGP routes |
| `routier update [--json]` | refresh package indexes and list available upgrades |
| `routier upgrade [--yes] [--reboot]` | upgrade all packages, syncing the bootloader |
| `routier takeover [--force]` | back up the host network config and hand networking to Routier |
| `routier setup` | disk installer, available on installation media only |
| `routier onboarding status\|complete\|reset` | manage the web setup wizard state |
| `routier cibi join <cibi-url> <code>` | register this router in a cibi fleet manager |
| `routier version` | print version |

Run `routier <command> --help` for the full flags of any command.
