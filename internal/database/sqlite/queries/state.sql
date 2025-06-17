-- name: UpsertState :exec
INSERT INTO state (
    server,
    type,
    last_sync
)
VALUES (
    sqlc.arg(server),
    sqlc.arg(type),
    sqlc.arg(last_sync)
)

ON CONFLICT(server, type) DO UPDATE SET last_sync = excluded.last_sync;

-- name: GetLastCheck :one
SELECT
    last_sync
FROM state
WHERE
    server = sqlc.arg(server) AND
    type = sqlc.arg(type);
