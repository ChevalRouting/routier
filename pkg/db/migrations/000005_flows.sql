-- +goose Up
CREATE TABLE IF NOT EXISTS flows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    start_ts INTEGER NOT NULL,
    end_ts INTEGER NOT NULL,
    in_if TEXT NOT NULL DEFAULT '',
    out_if TEXT NOT NULL DEFAULT '',
    family INTEGER NOT NULL,
    proto INTEGER NOT NULL,
    src TEXT NOT NULL,
    dst TEXT NOT NULL,
    src_port INTEGER NOT NULL DEFAULT 0,
    dst_port INTEGER NOT NULL DEFAULT 0,
    packets INTEGER NOT NULL,
    bytes INTEGER NOT NULL,
    samples INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_flows_end ON flows(end_ts);

-- +goose Down
DROP TABLE IF EXISTS flows;
