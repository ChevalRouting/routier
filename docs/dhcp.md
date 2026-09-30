# DHCP (Kea)

Routier manages ISC Kea DHCP servers declaratively: subnets, pools and host
reservations live in the `dhcp:` section of the config, and the apply pipeline
renders the Kea configuration and reloads the servers. Leases are runtime
state and are queried live, never stored in the config.

## Configuration

```yaml
dhcp:
  enabled: true
  subnets4:
    - subnet: 192.168.10.0/24
      interface: lan            # logical interface or VLAN name
      pools:
        - 192.168.10.100-192.168.10.200
      gateway: 192.168.10.1
      dns: [192.168.10.1]
      reservations:
        - hw_address: aa:bb:cc:dd:ee:ff
          ip_address: 192.168.10.5
          hostname: printer
  subnets6:
    - subnet: 2001:db8:10::/64
      interface: lan
```

- The address family is inferred from the subnet CIDR; `subnets4` renders
  `kea-dhcp4.conf`, `subnets6` renders `kea-dhcp6.conf`, each only when it has
  subnets.
- `interface` accepts a Routier logical name (resolved to the real device), a
  literal device, or `*`.
- Reservations are declarative: they are part of the config, versioned and
  applied like everything else.

## Dynamic DNS (leases become DNS records)

When `dhcp.ddns` is enabled, Kea's DDNS daemon (`kea-dhcp-ddns`, "D2") turns
leases into DNS records: as clients get and release addresses, forward
(`A`/`AAAA`) and reverse (`PTR`) records appear and disappear in the local BIND.

```yaml
dhcp:
  enabled: true
  subnets4:
    - subnet: 192.168.10.0/24
      interface: lan
      pools:
        - 192.168.10.100-192.168.10.200
  ddns:
    enabled: true
    domain: lan.example.com     # forward zone and qualifying suffix
    ttl: 3600
    reverse: true               # PTR updates (default true)

dns:
  server:
    enabled: true               # DDNS requires the local BIND
```

- `dhcp.ddns.enabled` requires `dns.server.enabled: true`: D2 sends the updates
  to the BIND that Routier runs.
- The **forward zone** (`domain`) and the **reverse zones** derived from each
  subnet CIDR (`in-addr.arpa` / `ip6.arpa`) are created and owned automatically
  as dynamic primaries. Do not declare them under `dns.server.zones`; Routier
  writes a minimal `SOA`+`NS` bootstrap file once and lets D2 own the records
  from then on.
- Reverse zones are derived only for octet-aligned IPv4 prefixes (`/8`, `/16`,
  `/24`) and nibble-aligned IPv6 prefixes (a multiple of 4). Other prefixes are
  skipped for PTR, and forward records still work.
- Kea and BIND authenticate updates with a shared **TSIG key**. Routier
  generates it on first apply and writes it back into the config as
  `dhcp.ddns.key`, so both `kea-dhcp-ddns.conf` and `named.conf` render the same
  secret. If your config lives in a directory of split files, set `dhcp.ddns.key`
  explicitly (the auto-persist writes to a single config file).
- Per subnet you can override with `ddns: false` (opt a subnet out) or
  `ddns_domain:` (a different qualifying suffix).

## How Routier talks to Kea

Kea 3.0 deprecated the standalone Control Agent, so each local DHCP server
exposes its own control socket. Routier talks to those unix sockets directly,
which also avoids binding a loopback port the appliance may not have. Only
base commands plus the stock `lease_cmds` hook are used, so a standard Kea
build is enough; the premium `subnet_cmds`/`host_cmds` hooks are not needed.

Applying DHCP changes uses `config-reload`, which re-reads the rendered config
without dropping active leases. A failed reload falls back to a restart, and a
config that fails validation (`kea-dhcp4 -t`) aborts the apply and rolls back.

## Runtime operations

The Monitor page (and the matching endpoints under `/api/dhcp/`) exposes:

- **Leases**: a live list per subnet, with search (an exact IP is resolved by
  Kea, anything else is matched server-side), reserved vs dynamic marking,
  and pagination.
- **Reserve from lease**: stages a reservation for a leased host, using a free
  address outside every dynamic pool, applies it, then clears the lease so the
  client picks up its reserved address on the next request. IPv4 free
  addresses are found by a deterministic scan that jumps pool ranges; IPv6 by
  random selection with retry, since a /64 cannot be scanned.
- **Clear lease**: deletes an active lease.
- **Statistics and live lease events**: key server statistics plus a streamed
  tail of the Kea lease logs.

The same operations exist on the CLI (`routier dhcp ...`), which resolves the
control endpoint the same way the API does, so both drive the same server.
