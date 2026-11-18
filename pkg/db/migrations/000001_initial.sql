-- +goose Up
CREATE TABLE ui_layout (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE kernel_routes (
    dst TEXT NOT NULL DEFAULT '',
    gateway TEXT NOT NULL DEFAULT '',
    dev TEXT NOT NULL DEFAULT '',
    protocol TEXT NOT NULL DEFAULT '',
    metric INTEGER NOT NULL DEFAULT 0,
    family TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_kernel_routes ON kernel_routes(family, dst);

CREATE TABLE iface_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ts INTEGER NOT NULL,
    iface TEXT NOT NULL,
    rx_bytes INTEGER NOT NULL DEFAULT 0,
    tx_bytes INTEGER NOT NULL DEFAULT 0,
    rx_pkts INTEGER NOT NULL DEFAULT 0,
    tx_pkts INTEGER NOT NULL DEFAULT 0,
    rx_errs INTEGER NOT NULL DEFAULT 0,
    tx_errs INTEGER NOT NULL DEFAULT 0,
    rx_bps REAL,
    tx_bps REAL,
    rx_pps REAL,
    tx_pps REAL,
    operstate TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_iface_stats ON iface_stats(iface, ts DESC);

CREATE INDEX idx_iface_stats_ts ON iface_stats(ts DESC);

CREATE TABLE system_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ts INTEGER NOT NULL,
    cpu_pct REAL NOT NULL DEFAULT 0,
    mem_used INTEGER NOT NULL DEFAULT 0,
    mem_total INTEGER NOT NULL DEFAULT 0,
    load1 REAL NOT NULL DEFAULT 0
);

CREATE INDEX idx_system_stats ON system_stats(ts DESC);

CREATE TABLE bgp_peer_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ts INTEGER NOT NULL,
    peer TEXT NOT NULL,
    remote_asn INTEGER NOT NULL DEFAULT 0,
    state TEXT NOT NULL DEFAULT '',
    uptime TEXT NOT NULL DEFAULT '',
    msg_rcvd INTEGER NOT NULL DEFAULT 0,
    msg_sent INTEGER NOT NULL DEFAULT 0,
    prefixes INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_bgp_peer_stats ON bgp_peer_stats(peer, ts DESC);

CREATE INDEX idx_bgp_peer_stats_ts ON bgp_peer_stats(ts DESC);

CREATE TABLE proto_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ts INTEGER NOT NULL,
    tcp_active_opens INTEGER NOT NULL DEFAULT 0,
    tcp_passive_opens INTEGER NOT NULL DEFAULT 0,
    tcp_attempt_fails INTEGER NOT NULL DEFAULT 0,
    tcp_estab_resets INTEGER NOT NULL DEFAULT 0,
    tcp_curr_estab INTEGER NOT NULL DEFAULT 0,
    tcp_in_segs INTEGER NOT NULL DEFAULT 0,
    tcp_out_segs INTEGER NOT NULL DEFAULT 0,
    tcp_retrans_segs INTEGER NOT NULL DEFAULT 0,
    udp_in_datagrams INTEGER NOT NULL DEFAULT 0,
    udp_out_datagrams INTEGER NOT NULL DEFAULT 0,
    udp_in_errors INTEGER NOT NULL DEFAULT 0,
    udp_no_ports INTEGER NOT NULL DEFAULT 0,
    icmp_in_msgs INTEGER NOT NULL DEFAULT 0,
    icmp_out_msgs INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_proto_stats ON proto_stats(ts DESC);

CREATE TABLE neighbor_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ts INTEGER NOT NULL,
    ip TEXT NOT NULL,
    mac TEXT NOT NULL DEFAULT '',
    dev TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL DEFAULT '',
    family TEXT NOT NULL DEFAULT 'ipv4',
    hostname TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_neighbor_stats ON neighbor_stats(ts DESC);

CREATE TABLE probe_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ts INTEGER NOT NULL,
    name TEXT NOT NULL,
    target TEXT NOT NULL,
    reachable INTEGER NOT NULL,
    rtt_avg_ms REAL
);

CREATE INDEX idx_probe_stats_name_ts ON probe_stats(name, ts DESC);

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    base_dir TEXT NOT NULL DEFAULT '',
    config_yaml TEXT NOT NULL
);

CREATE TABLE macros (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    base_yaml TEXT NOT NULL,
    mod_yaml TEXT NOT NULL,
    sections TEXT NOT NULL DEFAULT '[]',
    created_at INTEGER NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    apply_count INTEGER NOT NULL DEFAULT 0,
    applied_at INTEGER
);

CREATE TABLE announcements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message TEXT NOT NULL,
    level TEXT NOT NULL DEFAULT 'info',
    enabled INTEGER NOT NULL DEFAULT 1,
    dismissible INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    token_hash BLOB NOT NULL UNIQUE,
    prefix TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    created_by TEXT NOT NULL,
    revoked_at INTEGER
);

CREATE UNIQUE INDEX idx_api_keys_active_name ON api_keys(name) WHERE revoked_at IS NULL;

CREATE TABLE lldp_neighbors (
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

CREATE INDEX idx_lldp_neighbors ON lldp_neighbors(ts DESC);

CREATE TABLE switch_credentials (
    name TEXT PRIMARY KEY,
    sealed BLOB NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE flows (
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

CREATE INDEX idx_flows_end ON flows(end_ts);

CREATE TABLE flow_names (
    address TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    resolved_at INTEGER NOT NULL
);

CREATE INDEX idx_flow_names_resolved ON flow_names(resolved_at);

-- +goose Down
DROP TABLE IF EXISTS flow_names;
DROP TABLE IF EXISTS flows;
DROP TABLE IF EXISTS switch_credentials;
DROP TABLE IF EXISTS lldp_neighbors;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS announcements;
DROP TABLE IF EXISTS macros;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS probe_stats;
DROP TABLE IF EXISTS neighbor_stats;
DROP TABLE IF EXISTS proto_stats;
DROP TABLE IF EXISTS bgp_peer_stats;
DROP TABLE IF EXISTS system_stats;
DROP TABLE IF EXISTS iface_stats;
DROP TABLE IF EXISTS kernel_routes;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS ui_layout;
