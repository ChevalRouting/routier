-- +goose Up
CREATE TABLE IF NOT EXISTS lldp_neighbors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ts INTEGER NOT NULL,
    local_iface TEXT NOT NULL DEFAULT '',
    protocol TEXT NOT NULL DEFAULT '',
    chassis_id TEXT NOT NULL DEFAULT '',
    chassis_name TEXT NOT NULL DEFAULT '',
    sys_descr TEXT NOT NULL DEFAULT '',
    mgmt_ip TEXT NOT NULL DEFAULT '',
    port_id TEXT NOT NULL DEFAULT '',
    port_descr TEXT NOT NULL DEFAULT '',
    capabilities TEXT NOT NULL DEFAULT '',
    vlan TEXT NOT NULL DEFAULT '',
    age TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_lldp_neighbors ON lldp_neighbors(ts DESC);

-- +goose Down
DROP TABLE IF EXISTS lldp_neighbors;
