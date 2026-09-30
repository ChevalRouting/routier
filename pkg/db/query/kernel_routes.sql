-- name: DeleteKernelRoutes :exec
DELETE FROM kernel_routes;

-- name: InsertKernelRoute :exec
INSERT INTO kernel_routes (dst, gateway, dev, protocol, metric, family)
VALUES (?, ?, ?, ?, ?, ?);

-- name: CountKernelRoutes :one
WITH filter_args(protocols) AS (VALUES (?2))
SELECT COUNT(*) FROM kernel_routes CROSS JOIN filter_args
WHERE (CAST(@default_only AS INTEGER) = 0 OR dst IN ('default', '0.0.0.0/0', '::/0'))
  AND @protocols = @protocols
  AND (@family = '' OR family = @family)
  AND (json_array_length(filter_args.protocols) = 0 OR EXISTS (
      SELECT 1 FROM json_each(filter_args.protocols)
      WHERE LOWER(kernel_routes.protocol) = LOWER(json_each.value)
  ))
  AND (@query = '' OR LOWER(dst) LIKE '%' || LOWER(@query) || '%'
      OR LOWER(gateway) LIKE '%' || LOWER(@query) || '%'
      OR LOWER(dev) LIKE '%' || LOWER(@query) || '%'
      OR LOWER(protocol) LIKE '%' || LOWER(@query) || '%');

-- name: QueryKernelRoutes :many
WITH filter_args(protocols) AS (VALUES (?2))
SELECT dst, gateway, dev, protocol, metric, family FROM kernel_routes CROSS JOIN filter_args
WHERE (CAST(@default_only AS INTEGER) = 0 OR dst IN ('default', '0.0.0.0/0', '::/0'))
  AND @protocols = @protocols
  AND (@family = '' OR family = @family)
  AND (json_array_length(filter_args.protocols) = 0 OR EXISTS (
      SELECT 1 FROM json_each(filter_args.protocols)
      WHERE LOWER(kernel_routes.protocol) = LOWER(json_each.value)
  ))
  AND (@query = '' OR LOWER(dst) LIKE '%' || LOWER(@query) || '%'
      OR LOWER(gateway) LIKE '%' || LOWER(@query) || '%'
      OR LOWER(dev) LIKE '%' || LOWER(@query) || '%'
      OR LOWER(protocol) LIKE '%' || LOWER(@query) || '%')
ORDER BY family, dst
LIMIT @limit_count OFFSET @offset_count;
