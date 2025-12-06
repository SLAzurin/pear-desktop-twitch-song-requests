CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
) WITHOUT ROWID;
CREATE TABLE song_requests (
    id INTEGER PRIMARY KEY,
    user_id TEXT NOT NULL,
    song_title TEXT NOT NULL,
    artist_name TEXT,
    video_id TEXT,
    requested_at TEXT DEFAULT CURRENT_TIMESTAMP
);