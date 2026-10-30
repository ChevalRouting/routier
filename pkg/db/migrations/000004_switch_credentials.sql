-- +goose Up
CREATE TABLE switch_credentials (
    name TEXT PRIMARY KEY,
    sealed BLOB NOT NULL,
    updated_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS switch_credentials;
