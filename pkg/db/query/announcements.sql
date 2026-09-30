-- name: ListAnnouncements :many
SELECT id, message, level, enabled, dismissible, created_at, updated_at
FROM announcements ORDER BY id DESC;

-- name: ListEnabledAnnouncements :many
SELECT id, message, level, enabled, dismissible, created_at, updated_at
FROM announcements WHERE enabled = 1 ORDER BY id DESC;

-- name: CreateAnnouncement :execresult
INSERT INTO announcements (message, level, enabled, dismissible, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateAnnouncement :execrows
UPDATE announcements
SET message = ?, level = ?, enabled = ?, dismissible = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteAnnouncement :execrows
DELETE FROM announcements WHERE id = ?;

