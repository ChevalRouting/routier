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

Every config section is a resource under the session. Objects support
`GET`/`PUT` (`/hostname`, `/dns`, `/routing`, `/ssh`, `/sysctl`, ...), keyed
maps additionally expose per-item routes (`/interfaces`,
`/interfaces/{name}`, same for `tunnels`, `wireguard`, `users`, `services`,
`vrfs`), and ordered lists are replaced wholesale (`/nftables`,
`/boot_modules`).

Writes are validated incrementally: a mutation is rejected only if it
introduces validation errors that were not already present. This gives
referential integrity (you cannot delete a VRF an interface still references)
without blocking unrelated edits. The full resolve-and-validate pass still
runs at apply.

The committed config is always the single YAML file on disk; only a
successful apply writes it.

## Web/UI API (`/api`)

The API the React frontend uses. It follows a per-user staging model
(`GET/PUT /api/config/{section}` edit a staging copy, `POST /api/config/apply`
validates, applies and promotes) and adds operational endpoints: stats
history, monitor streams, DHCP lease operations, friend management, backup
export/import, apply log, snapshots and rollback confirmation.

It also exposes stateless IP utilities under `/api/tools` (`subnet`, `reverse`,
`range`); see [tools.md](tools.md).

These endpoints back the UI and the CLI; for automation, prefer the v1 API.

## Authentication

`POST /api/login` with username and password returns a bearer token; send it
as `Authorization: Bearer <token>` on every request. Sessions and tokens are
stored server-side in the appliance database.
