CREATE TABLE IF NOT EXISTS spotify_songs(
    user_id int REFERENCES users(id)
        ON DELETE CASCADE
        NOT NULL,
    played_at timestamptz NOT NULL,
    track_id text NOT NULL,
    track_name text NOT NULL,
    track_blob json,
    album_id text NOT NULL,
    album_name text NOT NULL,
    artist_id text NOT NULL,
    artist_name text NOT NULL,
    context_blob json,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, played_at)
);