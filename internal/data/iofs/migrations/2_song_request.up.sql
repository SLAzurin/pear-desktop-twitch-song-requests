DROP TABLE IF EXISTS song_requests;

CREATE TABLE song_requests (
    video_id TEXT PRIMARY KEY,
    song_title TEXT NOT NULL,
    artist_name TEXT NOT NULL,
    image_url TEXT NOT NULL
) WITHOUT ROWID;

CREATE TABLE song_request_requesters (
    video_id TEXT NOT NULL,
    twitch_username TEXT NOT NULL,
    requested_at TEXT NOT NULL DEFAULT datetime,
    FOREIGN KEY(video_id) REFERENCES song_requests(video_id)
);
