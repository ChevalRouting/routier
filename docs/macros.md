# Macros

Macros are saved, reusable config fragments that can be previewed and applied
on demand. They are useful for repetitive changes: adding a standard
interface, a tenant, or a firewall block.

A macro stores a set of config sections. Applying it merges those sections
into the current config, and you can review the diff first.

- `routier macros list`
- `routier macros show <name-or-id>`
- `routier macros apply <name-or-id>`
- `routier macros delete <name-or-id>`

API: `GET/POST /api/macros`, `GET /api/macros/{id}`,
`DELETE /api/macros/{id}`, `GET /api/macros/{id}/diff`,
`POST /api/macros/{id}/apply`. UI: **Tools > Macros**.

Macros are stored in the local database (`--db`, default
`/var/lib/routier/web.db`).
