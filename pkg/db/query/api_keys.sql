-- name: CreateAPIKey :exec
INSERT INTO api_keys (id, name, token_hash, prefix, created_at, created_by)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ListAPIKeys :many
SELECT id, name, prefix, created_at, created_by, revoked_at
FROM api_keys
ORDER BY created_at DESC, name;

-- name: GetActiveAPIKeyByHash :one
SELECT id, name, prefix, created_at, created_by, revoked_at
FROM api_keys
WHERE token_hash = ? AND revoked_at IS NULL;

-- name: RevokeAPIKey :execrows
UPDATE api_keys
SET revoked_at = ?
WHERE id = ? AND revoked_at IS NULL;
