# HTTP API

Routier exposes two HTTP APIs, both served by the same daemon and protected
by the same bearer-token authentication: a versioned REST API for automation,
and the web API that backs the UI.

## REST API v1 (`/api/v1`)

The versioned, resource-oriented API for automation. Interactive
documentation is served by the appliance itself at `/api/v1/docs` (embedded
Swagger UI, works offline), with the OpenAPI document at
`/api/v1/docs/openapi.json`.

### Session model

All configuration editing happens inside a **session**, a server-side copy of
the committed config:

```
POST   /api/v1/sessions                 create a session (snapshot of the config)
GET    /api/v1/sessions                 list your sessions
GET    /api/v1/sessions/{id}            session status
DELETE /api/v1/sessions/{id}            discard
GET    /api/v1/sessions/{id}/diff       YAML diff against the committed config
POST   /api/v1/sessions/{id}/validate   full validation
POST   /api/v1/sessions/{id}/apply      validate, apply, arm watchdog, delete session
```

Automation that works with complete YAML documents can read the committed
canonical configuration with `GET /api/v1/config`, then use
`GET/PUT /api/v1/sessions/{id}/config` to read or replace a session. These
responses include the canonical YAML and its SHA-256 digest. Replacing a
session never changes the committed configuration; validate, inspect the diff
and apply explicitly.

Every config section is a resource under the session. Objects support
`GET`/`PUT` (`/hostname`, `/dns`, `/routing`, `/ssh`, `/sysctl`, ...), keyed
maps additionally expose per-item routes (`/interfaces`,
`/interfaces/{name}`, same for `vpn/wireguard`, `users`, `services`,
`vrfs`, `switches`), and ordered lists are replaced wholesale (`/nftables`,
`/boot_modules`).

Writes are validated incrementally: a mutation is rejected only if it
introduces validation errors that were not already present. This gives
referential integrity (you cannot delete a VRF an interface still references)
without blocking unrelated edits. The full resolve-and-validate pass still
runs at apply.

The committed config is always the single YAML file on disk; only a
confirmed apply writes it. After `POST /api/v1/sessions/{id}/apply`, confirm
with `POST /api/apply/confirm` once the node is reachable; until then the file
still holds the previous config, and the watchdog rolls the apply back after
its timeout. While an apply waits for confirmation, a new apply returns
`409 Conflict`.

## Web/UI API (`/api`)

The API the React frontend uses. It follows a per-user staging model
(`GET/PUT /api/config/{section}` edit a staging copy, `POST /api/config/apply`
validates and applies, promoting the staging copy on confirm) and adds operational endpoints: stats
history, monitor streams, DHCP lease operations, DNS operations (resolver
overview and statistics, zone inspection and per-zone reload, cache flush,
resolving a name through the local resolver, and a live query-log stream),
friend management, backup export/import, apply log, snapshots and rollback
confirmation.

The first-run wizard stages its result with `PUT /api/setup/network`
(hostname, an optional uplink on a port by DHCP or static address, and local
networks); the uplink becomes the interface `wan`, local networks are
masqueraded behind it, and the default firewall rules are added. The wizard
then applies the staging copy like any other change.

`GET /api/config` answers with the caller's staging copy when it has one.
`source=running` returns the running configuration instead, which lets a
controller plan against what is applied while another of its jobs stages.

It also exposes stateless IP utilities under `/api/tools` (`subnet`, `reverse`,
`range`); see [tools.md](tools.md).

Managed switches have their own operational routes; see [switches.md](switches.md#api).

These endpoints back the UI and the CLI; for automation, prefer the v1 API.

## Authentication

`POST /api/login` with username and password returns a bearer token; send it
as `Authorization: Bearer <token>` on every request. Sessions and tokens are
stored server-side in the appliance database.

VPN configuration is available at `/api/config/vpn` and session `/vpn`.
The section editors use `vpn.wireguard`; tunnels are interface entries with
`type: tunnel`. The `tunnels` and `vpn.tunnels` aliases edit these interface
entries for existing clients. Old WireGuard section and
session paths remain aliases for compatibility.
