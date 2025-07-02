CREATE TABLE IF NOT EXISTS movies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server TEXT NOT NULL,
    local_id TEXT NOT NULL,
    name TEXT NOT NULL,
    imdb_id INTEGER,
    tmdb_id INTEGER,
    runtime INTEGER NOT NULL,
    watched_date INTEGER NOT NULL,
    watched_progress FLOAT NOT NULL,
    watched_position_ticks INTEGER NOT NULL,
    is_favorite BOOL NOT NULL,
    last_seen INTEGER NOT NULL,

    UNIQUE (server, local_id)
);

CREATE INDEX IF NOT EXISTS idx_movies_server ON movies(server);
CREATE INDEX IF NOT EXISTS idx_movies_server_watched_date ON movies(server, watched_date);
CREATE INDEX IF NOT EXISTS idx_movies_imdb_id ON movies(imdb_id) WHERE imdb_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_movies_tmdb_id ON movies(tmdb_id) WHERE tmdb_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_movies_watched_date ON movies(watched_date);

CREATE TABLE IF NOT EXISTS episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server TEXT NOT NULL,
    local_id TEXT NOT NULL,
    name TEXT NOT NULL,
    series_name TEXT NOT NULL,
    season_name TEXT NOT NULL,
    imdb_id INTEGER,
    tmdb_id INTEGER,
    tvdb_id INTEGER,
    runtime INTEGER NOT NULL,
    watched_date INTEGER NOT NULL,
    watched_progress REAL NOT NULL,
    watched_position_ticks INTEGER NOT NULL,
    is_favorite BOOL NOT NULL,
    last_seen INTEGER NOT NULL,

    UNIQUE (server, local_id)
);

CREATE INDEX IF NOT EXISTS idx_episodes_server ON episodes(server);
CREATE INDEX IF NOT EXISTS idx_episodes_server_watched_date ON episodes(server, watched_date);
CREATE INDEX IF NOT EXISTS idx_episodes_imdb_id ON episodes(imdb_id) WHERE imdb_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_episodes_tmdb_id ON episodes(tmdb_id) WHERE tmdb_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_episodes_tvdb_id ON episodes(tvdb_id) WHERE tvdb_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_episodes_watched_date ON episodes(watched_date);

CREATE TABLE IF NOT EXISTS changelog (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server TEXT NOT NULL,
    local_id TEXT NOT NULL,
    date INTEGER NOT NULL,
    new_watched_date INTEGER NOT NULL,
    new_watched_progress REAL NOT NULL,
    new_watched_position_ticks INTEGER NOT NULL,
    new_is_favorite BOOL NOT NULL
);

CREATE TABLE IF NOT EXISTS state (
     id INTEGER PRIMARY KEY AUTOINCREMENT,
     server TEXT NOT NULL,
     type TEXT NOT NULL,
     last_sync INTEGER NOT NULL CHECK (last_sync > 0),

     UNIQUE (server, type)
);

CREATE TABLE schema_version
(
    version INTEGER NOT NULL
);
