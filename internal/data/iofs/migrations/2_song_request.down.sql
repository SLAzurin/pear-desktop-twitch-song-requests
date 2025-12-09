DROP TABLE IF EXISTS song_request_requesters;
DROP TABLE IF EXISTS song_requests;
CREATE TABLE IF NOT EXISTS song_requests (
    id INTEGER PRIMARY KEY,
    user_id TEXT NOT NULL,
    song_title TEXT NOT NULL,
    artist_name TEXT,
    video_id TEXT,
    requested_at TEXT DEFAULT CURRENT_TIMESTAMP
);