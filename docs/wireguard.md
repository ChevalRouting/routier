# WireGuard

`wireguard` maps an interface name to a WireGuard definition, rendered to a
`wg-quick` config.

```yaml
wireguard:
  wg0:
    listen_port: 51820
    private_key: <base64>
    addresses: ["10.10.0.1/24"]
    table: "off"
    allow_inbound: true
    peers:
      - name: laptop
        public_key: <base64>
        preshared_key: <base64>
        endpoint: peer.example.com:51820
        allowed_ips: ["10.10.0.2/32"]
        keepalive: 25
```

Interface fields: `listen_port`, `private_key` or `private_key_file` (one
required), `addresses`, `table`, `mtu`,
`pre_up`/`post_up`/`pre_down`/`post_down` hooks, `peers`, and `allow_inbound`
(open the listen port in the firewall input chain). `friend` is a provenance
tag set on friend-derived interfaces (below).

Peer fields: `public_key` (required), `private_key`, `preshared_key` or
`preshared_key_file`, `endpoint` (`host:port`), `allowed_ips`, `keepalive`,
`name`.

Key material can be generated via `POST /api/wireguard/keygen`, and
`POST /api/wireguard/pubkey` derives a public key from a private one. UI:
**Network > WireGuard**.

## Friend-derived interfaces

A WireGuard tunnel between two Routier nodes can be derived from a
[friend](friends.md) instead of being hand-written. The derivation:

- mints a **fresh keypair per side plus one shared preshared key** (the
  friend's ed25519 identity is never used for tunnel crypto),
- addresses the two endpoints from a subnet you supply,
- is **link-only**: `table: off` and `allowed_ips` limited to the peer's
  tunnel address, so no routes are installed,
- tags the interface with `friend: <name>` and pushes the counterpart to the
  friend.

Derived interfaces appear in the WireGuard list badged `derived . <friend>`
and remain ordinary, editable WireGuard config. Removing the friend offers to
cascade-delete them. See [friends.md](friends.md#derived-wireguard).
