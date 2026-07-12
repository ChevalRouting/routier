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
