-- name: GetMovieWithGreatestWatchedDate :many
-- Get movies with greatest watched_date among identical movies, excluding specified server
WITH episode_groups AS (
    -- Step 1: Normalize all movie data and create matching keys
    -- This CTE standardizes data types and creates a unique identifier for grouping identical movies
    SELECT
        CAST(id AS INTEGER) as id,
        CAST(server AS TEXT) as server,
        CAST(local_id AS TEXT) as local_id,
        CAST(name AS TEXT) as name,
        CAST(imdb_id AS INTEGER) as imdb_id,
        CAST(tmdb_id AS INTEGER) as tmdb_id,
        CAST(runtime AS INTEGER) as runtime,
        CAST(watched_date AS INTEGER) as watched_date,
        CAST(watched_progress AS REAL) as watched_progress,
        CAST(watched_position_ticks AS INTEGER) as watched_position_ticks,
        CAST(is_favorite AS BOOL) as is_favorite,
        -- Create a unique matching key to identify the same movie across different servers
        -- Priority: IMDB ID > TMDB ID > Name+Runtime combination
        CASE
            WHEN imdb_id IS NOT NULL AND imdb_id != '' THEN CONCAT('imdb_', imdb_id)
            WHEN tmdb_id IS NOT NULL AND tmdb_id != '' THEN CONCAT('tmdb_', tmdb_id)
            ELSE CONCAT('name_', name, '_', CAST(runtime AS VARCHAR))
            END as match_key
    FROM movies
),
     local_movies AS (
         -- Step 2: Get watch status from the local server (the one we're syncing TO)
         -- This represents the current state of movies on the target server
         SELECT
             match_key,
             local_id,
             name,
             watched_date as local_watched_date,
             watched_progress as local_watched_progress
         FROM episode_groups
         WHERE server = sqlc.arg(server)
     ),
     max_remote_movies AS (
         -- Step 3: Find the most recent watch date for each movie on remote servers
         -- This identifies which remote server has the most up-to-date watch progress
         SELECT
             match_key,
             MAX(watched_date) as max_remote_watched_date
         FROM episode_groups
         WHERE server != sqlc.arg(server)
    AND watched_date > 0  -- Only consider movies that have been watched
GROUP BY match_key
    ),
    best_remote_movies AS (
-- Step 4: Get the complete record for the movie with the highest watch date on remote servers
-- Using window functions to get all details from the "winning" remote server
SELECT DISTINCT
    eg.match_key,
    FIRST_VALUE(eg.local_id) OVER (PARTITION BY eg.match_key ORDER BY eg.watched_date DESC) as remote_local_id,
    FIRST_VALUE(eg.name) OVER (PARTITION BY eg.match_key ORDER BY eg.watched_date DESC) as remote_name,
    FIRST_VALUE(eg.watched_date) OVER (PARTITION BY eg.match_key ORDER BY eg.watched_date DESC) as remote_watched_date,
    FIRST_VALUE(eg.watched_progress) OVER (PARTITION BY eg.match_key ORDER BY eg.watched_date DESC) as remote_watched_progress,
    FIRST_VALUE(eg.watched_position_ticks) OVER (PARTITION BY eg.match_key ORDER BY eg.watched_date DESC) as watched_position_ticks,
    FIRST_VALUE(eg.is_favorite) OVER (PARTITION BY eg.match_key ORDER BY eg.watched_date DESC) as is_favorite
FROM episode_groups eg
    INNER JOIN max_remote_movies mre ON eg.match_key = mre.match_key
    AND eg.watched_date = mre.max_remote_watched_date
WHERE eg.server != sqlc.arg(server)
    )
-- Step 5: Final result - Return movies that need their watch status updated
-- Only return movies where remote watch progress is newer than local watch progress
SELECT
    CAST(le.local_id AS TEXT) as local_id, -- ! Use local_id from local_episodes !
    CAST(bre.remote_name AS TEXT) as name,
    CAST(bre.remote_watched_date AS INTEGER) as watched_date,
    CAST(bre.remote_watched_progress AS REAL) as watched_progress,
    CAST(bre.watched_position_ticks AS INTEGER) as watched_position_ticks,
    CAST(bre.is_favorite AS BOOL) as is_favorite
FROM best_remote_movies bre
         INNER JOIN local_movies le ON bre.match_key = le.match_key
WHERE bre.remote_watched_date > COALESCE(le.local_watched_date, 0)
  AND bre.remote_watched_date > 0;

-- name: InsertMovie :exec
INSERT INTO
    movies (
        server,
        name,
        local_id,
        imdb_id,
        tmdb_id,
        runtime,
        watched_date,
        watched_progress,
        watched_position_ticks,
        is_favorite
    )
VALUES (
        sqlc.arg(server),
        sqlc.arg(name),
        sqlc.arg(local_id),
        sqlc.arg(imdb_id),
        sqlc.arg(tmdb_id),
        sqlc.arg(runtime),
        sqlc.arg(watched_date),
        sqlc.arg(watched_progress),
        sqlc.arg(watched_position_ticks),
        sqlc.arg(is_favorite)
)
ON CONFLICT(server, local_id) DO UPDATE SET
        name = excluded.name,
        imdb_id = excluded.imdb_id,
        tmdb_id = excluded.tmdb_id,
        runtime = excluded.runtime,
        watched_date = excluded.watched_date,
        watched_progress  = excluded.watched_progress,
        watched_position_ticks  = excluded.watched_position_ticks,
        is_favorite = excluded.is_favorite;
