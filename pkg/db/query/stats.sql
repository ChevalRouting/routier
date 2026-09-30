-- name: LastIfaceCounters :many
SELECT iface, ts, rx_bytes, tx_bytes, rx_pkts, tx_pkts FROM iface_stats
WHERE ts = (SELECT MAX(ts) FROM iface_stats);

-- name: InsertIfaceStat :exec
INSERT INTO iface_stats
(ts, iface, rx_bytes, tx_bytes, rx_pkts, tx_pkts, rx_errs, tx_errs, rx_bps, tx_bps, rx_pps, tx_pps, operstate)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: IfaceHistory :many
SELECT CAST(MAX(ts) AS INTEGER) AS ts, iface,
       rx_bytes, tx_bytes, rx_pkts, tx_pkts, rx_errs, tx_errs,
       AVG(rx_bps) AS rx_bps, AVG(tx_bps) AS tx_bps,
       AVG(rx_pps) AS rx_pps, AVG(tx_pps) AS tx_pps,
       operstate
FROM iface_stats
WHERE ts >= @cutoff AND @bucket > 0 AND (@iface = '' OR iface = @iface)
GROUP BY iface, ts / ?2
ORDER BY iface, ts ASC;

-- name: IfaceTotals :many
SELECT CAST(MAX(ts) AS INTEGER) AS ts, AVG(rx) AS rx, AVG(tx) AS tx, AVG(rxp) AS rxp, AVG(txp) AS txp FROM (
    SELECT ts, SUM(rx_bps) AS rx, SUM(tx_bps) AS tx, SUM(rx_pps) AS rxp, SUM(tx_pps) AS txp
    FROM iface_stats
    WHERE ts >= @cutoff AND @bucket > 0
      AND (json_array_length(@ifaces) = 0 OR iface IN (SELECT value FROM json_each(?3)))
    GROUP BY ts
)
GROUP BY ts / CAST(@bucket AS INTEGER)
ORDER BY ts ASC;

-- name: InsertSystemStat :exec
INSERT INTO system_stats (ts, cpu_pct, mem_used, mem_total, load1) VALUES (?, ?, ?, ?, ?);

-- name: SystemHistory :many
SELECT CAST(MAX(ts) AS INTEGER) AS ts, AVG(cpu_pct) AS cpu_pct, CAST(AVG(mem_used) AS INTEGER) AS mem_used,
       CAST(MAX(mem_total) AS INTEGER) AS mem_total, AVG(load1) AS load1
FROM system_stats WHERE ts >= @cutoff AND @bucket > 0
GROUP BY ts / ?2 ORDER BY ts ASC;

-- name: InsertBGPStat :exec
INSERT INTO bgp_peer_stats (ts, peer, remote_asn, state, uptime, msg_rcvd, msg_sent, prefixes)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: BGPHistory :many
SELECT CAST(MAX(ts) AS INTEGER) AS ts, peer, state, uptime, msg_rcvd, msg_sent, prefixes
FROM bgp_peer_stats WHERE ts >= @cutoff AND @bucket > 0
GROUP BY peer, ts / ?2 ORDER BY peer, ts ASC;

-- name: InsertProtoStat :exec
INSERT INTO proto_stats
(ts, tcp_active_opens, tcp_passive_opens, tcp_attempt_fails, tcp_estab_resets,
 tcp_curr_estab, tcp_in_segs, tcp_out_segs, tcp_retrans_segs,
 udp_in_datagrams, udp_out_datagrams, udp_in_errors, udp_no_ports,
 icmp_in_msgs, icmp_out_msgs)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ProtoHistory :many
SELECT CAST(MAX(ts) AS INTEGER) AS ts, tcp_active_opens, tcp_passive_opens, tcp_attempt_fails,
       tcp_estab_resets, tcp_curr_estab, tcp_in_segs, tcp_out_segs, tcp_retrans_segs,
       udp_in_datagrams, udp_out_datagrams, udp_in_errors, udp_no_ports,
       icmp_in_msgs, icmp_out_msgs
FROM proto_stats WHERE ts >= @cutoff AND @bucket > 0
GROUP BY ts / ?2 ORDER BY ts ASC;

-- name: InsertNeighborStat :exec
INSERT INTO neighbor_stats (ts, ip, mac, dev, state, family, hostname) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: LatestNeighbors :many
SELECT ip, mac, dev, state, family, hostname FROM neighbor_stats
WHERE ts = ? ORDER BY ip ASC;

-- name: InsertLLDPNeighbor :exec
INSERT INTO lldp_neighbors (ts, local_iface, protocol, chassis_id, chassis_name, sys_descr, mgmt_ip, port_id, port_descr, capabilities, vlan, age)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: LatestLLDPNeighbors :many
SELECT local_iface, protocol, chassis_id, chassis_name, sys_descr, mgmt_ip, port_id, port_descr, capabilities, vlan, age
FROM lldp_neighbors WHERE ts = ? ORDER BY local_iface ASC;

-- name: LastLLDPTS :one
SELECT CAST(COALESCE(MAX(ts), 0) AS INTEGER) FROM lldp_neighbors;

-- name: PruneLLDPNeighbors :exec
DELETE FROM lldp_neighbors WHERE ts < ?;

-- name: LastProbeTS :one
SELECT CAST(COALESCE(MAX(ts), 0) AS INTEGER) FROM probe_stats WHERE name = ?;

-- name: LastAnyProbeTS :one
SELECT CAST(COALESCE(MAX(ts), 0) AS INTEGER) FROM probe_stats;

-- name: InsertProbeStat :exec
INSERT INTO probe_stats (ts, name, target, reachable, rtt_avg_ms) VALUES (?, ?, ?, ?, ?);

-- name: ProbeHistory :many
SELECT CAST(MAX(ts) AS INTEGER) AS ts, name, target, reachable, rtt_avg_ms
FROM probe_stats WHERE ts >= @cutoff AND @bucket > 0
GROUP BY name, ts / ?2 ORDER BY ts ASC;

-- name: LastIfaceTS :one
SELECT CAST(COALESCE(MAX(ts), 0) AS INTEGER) FROM iface_stats;

-- name: LastBGPTS :one
SELECT CAST(COALESCE(MAX(ts), 0) AS INTEGER) FROM bgp_peer_stats;

-- name: LastProtoTS :one
SELECT CAST(COALESCE(MAX(ts), 0) AS INTEGER) FROM proto_stats;

-- name: LastSystemTS :one
SELECT CAST(COALESCE(MAX(ts), 0) AS INTEGER) FROM system_stats;

-- name: LastNeighborTS :one
SELECT CAST(COALESCE(MAX(ts), 0) AS INTEGER) FROM neighbor_stats;

-- name: PruneIfaceStats :exec
DELETE FROM iface_stats WHERE ts < ?;

-- name: PruneBGPStats :exec
DELETE FROM bgp_peer_stats WHERE ts < ?;

-- name: PruneProtoStats :exec
DELETE FROM proto_stats WHERE ts < ?;

-- name: PruneSystemStats :exec
DELETE FROM system_stats WHERE ts < ?;

-- name: PruneNeighborStats :exec
DELETE FROM neighbor_stats WHERE ts < ?;

-- name: PruneProbeStats :exec
DELETE FROM probe_stats WHERE ts < ?;
