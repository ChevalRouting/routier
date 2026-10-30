-- +goose Up
CREATE TABLE IF NOT EXISTS flow_names (
    address TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    resolved_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_flow_names_resolved ON flow_names(resolved_at);

-- +goose Down
DROP TABLE IF EXISTS flow_names;
