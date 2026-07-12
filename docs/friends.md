# Friends

Friends are mutually-authenticated peer routers. A friend relationship is the
substrate for cooperative features: **liveness** monitoring and **derived
WireGuard** tunnels.

Everything here is available from the web UI (**System > Friends**), the HTTP
API (`/api/friends/...`), and the CLI (`routier friends ...`).

## Identity

Each node has an **ed25519 host identity** key, generated once and stored as a
host secret (alongside the web DB, e.g. `/var/lib/routier/identity_ed25519`).
It is used only for authentication and fingerprinting, never for WireGuard
crypto. A node advertises its hostname + public key; the SHA256 fingerprint
(`SHA256:...`) is what an operator verifies.

## Configuration

```yaml
friends:
  - name: edge-b
    hostname: edge-b
    url: https://10.0.0.2:8080
    token: <shared secret>        # bearer used in both directions for this pair
    tls_skip_verify: true
    enabled: true
    identity:
      fingerprint: SHA256:AbCd...  # pinned at add time (TOFU)
```

- `token` - one shared bearer token per pair, used in both directions.
- `identity.fingerprint` - the pinned ed25519 fingerprint (TOFU).

The ed25519 identity key is a host secret and is never in this YAML.

## Adding a friend (TOFU)

Adding is trust-on-first-use:

1. Contact the prospective friend and fetch its identity.
2. Confirm the fingerprint matches the remote node.
3. Save it (the fingerprint is pinned) and pair so the remote pins you too.

```
routier friends preview --url https://10.0.0.2:8080 --token <shared>
routier friends add --name edge-b --url https://10.0.0.2:8080 --token <shared> \
    --fingerprint SHA256:AbCd...        # or --yes to trust on first contact
```

API: `POST /api/friends/preview` (fetch identity, saves nothing) then
`POST /api/friends` (verifies the confirmed fingerprint, self-check, persists,
best-effort pair). UI: the add dialog shows the fingerprint for confirmation.

Pairing (`POST /api/friends/pair`) authenticates with the shared token and a
signature over the caller's fingerprint, so it succeeds when the receiver
already holds that token. A single confirmed add then pins both sides;
otherwise add on both sides with the same token.

## Liveness

A background poller hits each enabled friend's `/api/friends/hello` roughly
every 10s with a random **challenge nonce**; the friend signs it with its
identity key. Routier verifies the signature against the pinned fingerprint,
so a stolen token alone cannot impersonate a node. The poller records
reachability, RTT, last-seen, version, and hostname (in memory).

```
routier friends list
```

API: `GET /api/friends` (list) and `GET /api/friends/status` (live snapshot).
UI: friend cards with a status dot, latency, and an identity-verified badge;
the dashboard shows a "Friends" card (`alive / total`).

## Derived WireGuard

Derive a link-only tunnel to a friend instead of hand-writing it:

```
routier friends interfaces edge-b      # list the friend's interfaces/IPs
routier friends wireguard edge-b \
    --subnet 169.254.50.0/31 \
    --local-iface wan --friend-iface wan
```

The derivation mints a fresh keypair per side + one shared PSK, addresses the
two endpoints from the subnet, picks a shared random port (overridable per
side), writes the local interface tagged `friend: edge-b`, and pushes the
counterpart to the friend. It is **link-only** (`table: off`, `allowed_ips` =
the peer tunnel address) and installs no routes; routing over the tunnel is a
separate, manual concern. Endpoint addresses are picked automatically when an
interface has one IP and prompted for when it has several.

API: `GET /api/friends/{name}/interfaces` and
`POST /api/friends/{name}/wireguard`. UI: "Create WireGuard" on the friend
card.

## Removing a friend (cascade)

Removing a friend offers to delete the config sections derived from it
(tagged `friend: <name>`, e.g. derived WireGuard interfaces).

```
routier friends remove edge-b           # prints what would be removed
routier friends remove edge-b --yes     # removes the friend + tagged sections
```

API: `DELETE /api/friends/{name}` returns the tagged sections without
deleting; `DELETE /api/friends/{name}?confirm=true` performs the cascade. The
UI always prompts first.

## Security summary

- Per-pair shared bearer token, constant-time compared, **scoped** to the
  friend-facing endpoints + the config push path (not full admin).
- ed25519 identity pinned via TOFU; per-poll challenge-response proves key
  possession.
- TLS (optionally `tls_skip_verify` for self-signed); WireGuard material is
  sent inline over this authenticated, encrypted channel.

## Friend cache

What Routier learns from each friend (liveness and the friend's interfaces)
is cached in memory and written through to `friends_cache.json` in the runtime
state dir. This is runtime/derived state, never config. The poller refreshes
it, and when a friend is unreachable the interface endpoints and WireGuard
derivation fall back to the cache (interface responses set
`X-Friend-Stale: true`), so config can still be (re)generated while a peer is
down. The cache entry is dropped when the friend is removed.

## Config overlay (friend interpolation)

Config string values can reference a friend's data with a template
expression:

```yaml
routing:
  static:
    - destination: 10.9.0.0/24
      via: '{{ friend "edge-b" "interfaces.wan.address" }}'
```

Supported paths: `hostname`, `fingerprint`, `version`,
`interfaces.<iface>.address` (first address) and
`interfaces.<iface>.addresses` (comma-joined).

The stored config keeps the **expression**, which stays declarative and
stateless; the value is resolved at validate/apply time from the friend cache
above, so it resolves even while the friend is briefly offline. Validation
and render run on the resolved config; the file on disk keeps
`{{ friend ... }}`. A reference to a friend that has never been contacted (no
cached data) fails with a clear error.

Both the API/daemon apply path and `routier apply` resolve interpolation: the
API uses the live in-memory cache, the CLI reads the persisted
`friends_cache.json` (override with `--friends-cache`).
