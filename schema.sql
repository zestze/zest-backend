CREATE TABLE users(
    id serial PRIMARY KEY,
    username text UNIQUE NOT NULL,
    password text UNIQUE NOT NULL,
    salt int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE reddit_posts(
    id serial PRIMARY KEY,
    name  text UNIQUE NOT NULL,
    user_id int REFERENCES users(id) 
        ON DELETE CASCADE
        NOT NULL,
    permalink text NOT NULL,
    subreddit text NOT NULL,
    title text,
    num_comments int,
    upvote_ratio real,
    ups int,
    score int,
    total_awards_received int,
    suggested_sort text,
    link_title text,
    body text,
    created_utc real, -- seeconds since the epoch
    created_at timestamptz NOT NULL DEFAULT now()
);


CREATE TYPE metacritic_medium AS ENUM ('switch', 'tv', 'movie', 'pc');

CREATE TABLE metacritic_posts(
    id serial PRIMARY KEY,
    title text UNIQUE NOT NULL,
    href text UNIQUE NOT NULL,
    score int NOT NULL,
    description text,
    released date NOT NULL,
    medium metacritic_medium NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);


CREATE TABLE spotify_tokens(
    user_id int PRIMARY KEY REFERENCES users(id) 
        ON DELETE CASCADE NOT NULL,
    access_token text NOT NULL,
    token_type text,
    scope text,
    expires_at timestamptz NOT NULL,
    refresh_token text NOT NULL
);

CREATE TABLE spotify_songs(
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

-- TODO(zeke): ignoring artists listed in albums for now
-- TODO(zeke): also ignoring created_at for now!
CREATE TABLE spotify_albums(
    id text PRIMARY KEY,
    name text NOT NULL,
    href text NOT NULL,
    uri text NOT NULL,
    external_url text NOT NULL,
    type text NOT NULL
);

CREATE TABLE spotify_tracks(
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

CREATE TABLE spotify_artists(
    id text PRIMARY KEY,
    name text NOT NULL,
    href text NOT NULL,
    uri text NOT NULL,
    external_url text NOT NULL,
    genres text[],
    popularity int
);

-- association table
CREATE TABLE spotify_credits(
    track_id text references spotify_tracks(id)
        ON DELETE CASCADE
        NOT NULL,
    artist_id text references spotify_artists(id)
        ON DELETE CASCADE
        NOT NULL,
    PRIMARY KEY (track_id, artist_id)
);

CREATE TABLE spotify_played_tracks(
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

CREATE TABLE saved_metacritic_posts(
    post_id int REFERENCES metacritic_posts(id)
        ON DELETE CASCADE
        NOT NULL,
    user_id int REFERENCES users(id)
        ON DELETE CASCADE
        NOT NULL,
    action text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);