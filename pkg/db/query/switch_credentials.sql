-- name: UpsertSwitchCredentials :exec
INSERT INTO switch_credentials (name, sealed, updated_at)
VALUES (?, ?, ?)
ON CONFLICT (name) DO UPDATE SET
    sealed = excluded.sealed,
    updated_at = excluded.updated_at;

-- name: GetSwitchCredentials :one
SELECT sealed
FROM switch_credentials
WHERE name = ?;

-- name: ListSwitchCredentialNames :many
SELECT name
FROM switch_credentials
ORDER BY name;

-- name: DeleteSwitchCredentials :execrows
DELETE FROM switch_credentials
WHERE name = ?;
