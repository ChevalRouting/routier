# IP tools

Address-calculation utilities that do not touch the config: an ipcalc-style
subnet breakdown, reverse-DNS name resolution, and range-to-CIDR conversion.
They are available in the web UI (**Tools > IP Tools**), the HTTP API
(`/api/tools/...`), and the CLI (`routier ipcalc ...`). All three share the same
`pkg/iptools` engine, so the results match.

## Subnet calculator

Given an address or CIDR, it reports the network, netmask, wildcard, host range,
broadcast, host count, and (for IPv4) the classful class and address scope, with
the binary form of each row split at the prefix boundary. A bare address is
treated as a host route (`/32` or `/128`).

IPv4 host counts follow ipcalc: a `/31` has two usable hosts and no broadcast, a
`/32` is a host route with one host, and every other prefix reports
`2^hostbits - 2`. IPv6 has no broadcast or classful class, and every address in
the block counts as usable.

```
routier ipcalc subnet 100.64.12.192/27
routier ipcalc subnet 2001:db8::/64
```

API: `GET /api/tools/subnet?cidr=100.64.12.192/27`.

## Reverse DNS

Translates an address into its reverse-DNS (PTR) name: `in-addr.arpa` for IPv4,
`ip6.arpa` for IPv6. When the input carries a prefix aligned to an octet (IPv4)
or nibble (IPv6) boundary, the delegation zone is returned as well.

```
routier ipcalc reverse 100.64.12.192
routier ipcalc reverse 2001:db8::/32
```

API: `GET /api/tools/reverse?ip=100.64.12.192`.

## Range to CIDR

Splits an inclusive address range into the minimal set of aligned CIDR blocks
that exactly covers it. Start and end must be the same address family.

```
routier ipcalc range 192.0.2.5 192.0.2.20
```

API: `GET /api/tools/range?start=192.0.2.5&end=192.0.2.20`.
