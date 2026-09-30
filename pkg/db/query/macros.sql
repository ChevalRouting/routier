-- name: InsertMacro :exec
INSERT INTO macros (id, name, description, base_yaml, mod_yaml, sections, created_at, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: LoadMacro :one
SELECT id, name, description, base_yaml, mod_yaml, sections, created_at, created_by, apply_count, applied_at
FROM macros WHERE id = ?;

-- name: LoadMacroByName :one
SELECT id, name, description, base_yaml, mod_yaml, sections, created_at, created_by, apply_count, applied_at
FROM macros WHERE name = ?;

-- name: ListMacros :many
SELECT id, name, description, base_yaml, mod_yaml, sections, created_at, created_by, apply_count, applied_at
FROM macros ORDER BY created_at DESC;

-- name: DeleteMacro :execrows
DELETE FROM macros WHERE id = ?;

-- name: MarkMacroApplied :exec
UPDATE macros SET apply_count = apply_count + 1, applied_at = ? WHERE id = ?;

