-- name: InsertChangelog :exec
INSERT INTO changelog (
	server,
	local_id,
	date,
	new_watched_date,
	new_watched_progress,
	new_watched_position_ticks,
	new_is_favorite
)
VALUES (
	sqlc.arg(server),
	sqlc.arg(local_id),
	sqlc.arg(date),
	sqlc.arg(new_watched_date),
	sqlc.arg(new_watched_progress),
	sqlc.arg(new_watched_position_ticks),
	sqlc.arg(new_is_favorite)
)
