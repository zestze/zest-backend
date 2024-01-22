BEGIN;

CREATE TABLE IF NOT EXISTS users(
    id serial PRIMARY KEY,
    username text UNIQUE NOT NULL,
    password text UNIQUE NOT NULL,
    salt int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS reddit_posts(
    id serial PRIMARY KEY,
    name  text UNIQUE NOT NULL,
    user_id int REFERENCES users(id) ON DELETE CASCADE,
    permalink text UNIQUE NOT NULL,
    subreddit text NOT NULL,
    title text,
    num_comments int,
    upvote_ratio real,
    ups int,
    score int,
    total_awards_received int,
    suggested_sort text,
    created_utc real, -- seeconds since the epoch
    created_at timestamptz NOT NULL DEFAULT now()
);
-- TODO(zeke): opting for no index for now, but could put (user_id, subreddit)

DO $$ BEGIN
    CREATE TYPE metacritic_medium AS ENUM ('switch', 'tv', 'movie', 'pc');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS metacritic_posts(
    id serial PRIMARY KEY,
    title text UNIQUE NOT NULL,
    href text UNIQUE NOT NULL,
    score int NOT NULL,
    description text,
    released date NOT NULL,
    medium metacritic_medium NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

COMMIT;