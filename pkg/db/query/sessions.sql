-- name: InsertSession :exec
INSERT INTO sessions (id, username, created_at, base_dir, config_yaml)
VALUES (?, ?, ?, ?, ?);

-- name: LoadSession :one
SELECT id, username, created_at, base_dir, config_yaml FROM sessions WHERE id = ?;

-- name: UpdateSession :execrows
UPDATE sessions SET config_yaml = ? WHERE id = ?;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = ?;

-- name: ListUserSessions :many
SELECT id, username, created_at FROM sessions
WHERE username = ? ORDER BY created_at DESC;

