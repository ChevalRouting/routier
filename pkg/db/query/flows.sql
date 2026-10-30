-- name: InsertFlow :exec
INSERT INTO flows (start_ts, end_ts, in_if, out_if, family, proto, src, dst, src_port, dst_port, packets, bytes, samples)
VALUES (@start_ts, @end_ts, @in_if, @out_if, @family, @proto, @src, @dst, @src_port, @dst_port, @packets, @bytes, @samples);

-- name: DeleteFlowsEndedBefore :exec
DELETE FROM flows WHERE end_ts < @before;

-- name: TrimFlowsToNewest :exec
DELETE FROM flows WHERE id <= (SELECT id FROM flows ORDER BY id DESC LIMIT 1 OFFSET @keep);

-- name: CountFlows :one
SELECT COUNT(*) FROM flows;

-- name: RecentFlows :many
SELECT id, start_ts, end_ts, in_if, out_if, family, proto, src, dst, src_port, dst_port, packets, bytes, samples
FROM flows
WHERE end_ts >= @since AND (CAST(@address AS TEXT) = '' OR src = @address OR dst = @address)
ORDER BY end_ts DESC, id DESC
LIMIT @row_limit;

-- name: TopFlows :many
SELECT
    CAST(CASE @dimension
        WHEN 'src' THEN src
        WHEN 'dst' THEN dst
        WHEN 'service' THEN CAST(proto AS TEXT) || '/' || CAST(dst_port AS TEXT)
        WHEN 'protocol' THEN CAST(proto AS TEXT)
        WHEN 'in_if' THEN in_if
        WHEN 'out_if' THEN out_if
        ELSE src || ' > ' || dst
    END AS TEXT) AS group_key,
    CAST(SUM(bytes) AS INTEGER) AS bytes,
    CAST(SUM(packets) AS INTEGER) AS packets,
    COUNT(*) AS flows
FROM flows
WHERE end_ts >= @since
GROUP BY 1
ORDER BY 2 DESC
LIMIT @row_limit;

-- name: FlowSeries :many
SELECT CAST(MAX(end_ts) AS INTEGER) AS ts, in_if, CAST(SUM(bytes) AS INTEGER) AS bytes, CAST(SUM(packets) AS INTEGER) AS packets
FROM flows
WHERE end_ts >= @since AND @bucket > 0
GROUP BY in_if, end_ts / ?2
ORDER BY ts;

-- name: FlowPaths :many
WITH keyed AS (
    SELECT
        src,
        dst,
        CAST(proto AS TEXT) || '/' || CAST(dst_port AS TEXT) AS service,
        CAST(proto AS TEXT) AS protocol,
        in_if,
        out_if,
        bytes,
        packets
    FROM flows
    WHERE end_ts >= @since
)
SELECT
    CAST(CASE @first WHEN 'src' THEN src WHEN 'dst' THEN dst WHEN 'service' THEN service WHEN 'protocol' THEN protocol WHEN 'in_if' THEN in_if WHEN 'out_if' THEN out_if ELSE '' END AS TEXT) AS first_key,
    CAST(CASE @second WHEN 'src' THEN src WHEN 'dst' THEN dst WHEN 'service' THEN service WHEN 'protocol' THEN protocol WHEN 'in_if' THEN in_if WHEN 'out_if' THEN out_if ELSE '' END AS TEXT) AS second_key,
    CAST(CASE @third WHEN 'src' THEN src WHEN 'dst' THEN dst WHEN 'service' THEN service WHEN 'protocol' THEN protocol WHEN 'in_if' THEN in_if WHEN 'out_if' THEN out_if ELSE '' END AS TEXT) AS third_key,
    CAST(SUM(bytes) AS INTEGER) AS bytes,
    CAST(SUM(packets) AS INTEGER) AS packets
FROM keyed
GROUP BY 1, 2, 3
ORDER BY 4 DESC
LIMIT @row_limit;

-- name: FlowBytesSince :one
SELECT CAST(COALESCE(SUM(bytes), 0) AS INTEGER) FROM flows WHERE end_ts >= @since;

-- name: UpsertFlowName :exec
INSERT INTO flow_names (address, name, resolved_at) VALUES (@address, @name, @resolved_at)
ON CONFLICT(address) DO UPDATE SET name = excluded.name, resolved_at = excluded.resolved_at;

-- name: FlowNames :many
SELECT address, name FROM flow_names
WHERE name != '' AND json_array_length(@addresses) > 0 AND address IN (SELECT value FROM json_each(?1));

-- name: DeleteFlowNamesResolvedBefore :exec
DELETE FROM flow_names WHERE resolved_at < @before;
