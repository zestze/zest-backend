BEGIN;

-- TODO(zeke): ignoring artists listed in albums for now
-- TODO(zeke): also ignoring created_at for now!
CREATE TABLE IF NOT EXISTS spotify_albums(
    id text PRIMARY KEY,
    name text NOT NULL,
    href text NOT NULL,
    uri text NOT NULL,
    external_url text NOT NULL,
    type text NOT NULL
);

CREATE TABLE IF NOT EXISTS spotify_tracks(
    id text PRIMARY KEY,
    name text NOT NULL,
    href text NOT NULL,
    uri text NOT NULL,
    external_url text NOT NULL,
    album_id text REFERENCES spotify_albums(id)
        ON DELETE CASCADE
        NOT NULL,
    duration_ms int NOT NULL,
    explicit boolean,
    popularity int
);

CREATE TABLE IF NOT EXISTS spotify_artists(
    id text PRIMARY KEY,
    name text NOT NULL,
    href text NOT NULL,
    uri text NOT NULL,
    external_url text NOT NULL,
    genres text[],
    popularity int
);

-- association table
CREATE TABLE IF NOT EXISTS spotify_credits(
    track_id text references spotify_tracks(id)
        ON DELETE CASCADE
        NOT NULL,
    artist_id text references spotify_artists(id)
        ON DELETE CASCADE
        NOT NULL,
    PRIMARY KEY (track_id, artist_id)
);

CREATE TABLE IF NOT EXISTS spotify_played_tracks(
    user_id int REFERENCES users(id)
        ON DELETE CASCADE
        NOT NULL,
    played_at timestamptz NOT NULL,
    track_id text REFERENCES spotify_tracks(id)
        ON DELETE CASCADE
        NOT NULL,
    context_blob json,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, played_at)
);

COMMIT;