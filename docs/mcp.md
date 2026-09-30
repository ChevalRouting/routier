# MCP server

`routier-mcp` gives an AI client generic access to one or more Routier
instances. It exposes configuration sessions, apply lifecycle operations,
statistics and WireGuard key utilities. Network workflows remain the AI's
responsibility: the server does not contain specialized topology tools.

Run it on the AI client's host or another management system. It is not shipped
inside the Routier appliance APK or ISO.

## Instance registry

Create a local registry and keep it readable only by its owner:

```yaml
instances:
  edge-a:
    url: https://10.0.0.11:8080
    token: <routier-api-token>
    tls:
      skip_verify: false
  edge-b:
    url: https://10.0.0.12:8080
    token: <routier-api-token>
    tls:
      skip_verify: true
```

The registry embeds the Routier API tokens used by the MCP. Protect it as a
secret, for example with `chmod 600 routier-mcp.yml`. Tokens are never accepted
as tool arguments or returned to the MCP client. `token_env` may be used instead
of `token` when external secret injection is preferred; do not set both.

Run the stdio server with:

```bash
routier-mcp --config /path/to/routier-mcp.yml
```

Configure the AI client to launch that command as a local stdio MCP server.

## Tools

Discovery:

- `instances_list`
- `instance_status`

Canonical configuration sessions:

- `config_get`
- `config_session_create`
- `config_session_get`
- `config_session_put_yaml`
- `config_session_diff`
- `config_session_validate`
- `config_session_apply`
- `config_session_discard`

Apply lifecycle:

- `apply_pending`
- `apply_confirm`
- `apply_rollback`

Monitoring:

- `stats_current`
- `stats_history`
- `stats_neighbors`
- `stats_processes`

WireGuard utilities:

- `wireguard_keypair_generate`
- `wireguard_public_key_derive`

Every instance operation requires the registry name in `instance`.

## Safe configuration workflow

An AI should read the committed YAML, create a session, replace the session
YAML, validate it and inspect its diff before applying. Applying arms Routier's
normal rollback watchdog. Confirmation is a separate operation and should
happen only after the AI verifies that the instance remains healthy.

For a multi-instance change, create and validate every session before applying
any of them. Apply each instance with its watchdog armed, verify all instances,
then confirm them. If one fails, restore already-applied instances or leave
their watchdogs unconfirmed so they roll back.
