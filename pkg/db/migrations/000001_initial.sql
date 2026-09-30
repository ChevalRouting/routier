-- +goose Up
CREATE TABLE IF NOT EXISTS ui_layout (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS kernel_routes (
    dst TEXT NOT NULL DEFAULT '',
    gateway TEXT NOT NULL DEFAULT '',
    dev TEXT NOT NULL DEFAULT '',
    protocol TEXT NOT NULL DEFAULT '',
    metric INTEGER NOT NULL DEFAULT 0,
    family TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_kernel_routes ON kernel_routes(family, dst);

CREATE TABLE IF NOT EXISTS iface_stats (
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

CREATE INDEX IF NOT EXISTS idx_iface_stats ON iface_stats(iface, ts DESC);

CREATE INDEX IF NOT EXISTS idx_iface_stats_ts ON iface_stats(ts DESC);

CREATE TABLE IF NOT EXISTS system_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ts INTEGER NOT NULL,
    cpu_pct REAL NOT NULL DEFAULT 0,
    mem_used INTEGER NOT NULL DEFAULT 0,
    mem_total INTEGER NOT NULL DEFAULT 0,
    load1 REAL NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_system_stats ON system_stats(ts DESC);

CREATE TABLE IF NOT EXISTS bgp_peer_stats (
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

CREATE INDEX IF NOT EXISTS idx_bgp_peer_stats ON bgp_peer_stats(peer, ts DESC);

CREATE INDEX IF NOT EXISTS idx_bgp_peer_stats_ts ON bgp_peer_stats(ts DESC);

CREATE TABLE IF NOT EXISTS proto_stats (
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

CREATE INDEX IF NOT EXISTS idx_proto_stats ON proto_stats(ts DESC);

CREATE TABLE IF NOT EXISTS neighbor_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ts INTEGER NOT NULL,
    ip TEXT NOT NULL,
    mac TEXT NOT NULL DEFAULT '',
    dev TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL DEFAULT '',
    family TEXT NOT NULL DEFAULT 'ipv4',
    hostname TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_neighbor_stats ON neighbor_stats(ts DESC);

CREATE TABLE IF NOT EXISTS probe_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ts INTEGER NOT NULL,
    name TEXT NOT NULL,
    target TEXT NOT NULL,
    reachable INTEGER NOT NULL,
    rtt_avg_ms REAL
);

CREATE INDEX IF NOT EXISTS idx_probe_stats_name_ts ON probe_stats(name, ts DESC);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    base_dir TEXT NOT NULL DEFAULT '',
    config_yaml TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS macros (
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

CREATE TABLE IF NOT EXISTS announcements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message TEXT NOT NULL,
    level TEXT NOT NULL DEFAULT 'info',
    enabled INTEGER NOT NULL DEFAULT 1,
    dismissible INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- +goose Down
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

