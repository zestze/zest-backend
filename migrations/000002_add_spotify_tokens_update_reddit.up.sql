BEGIN;

CREATE TABLE IF NOT EXISTS spotify_tokens(
    user_id int PRIMARY KEY REFERENCES users(id) 
        ON DELETE CASCADE NOT NULL,
    access_token text NOT NULL,
    token_type text,
    scope text,
    expires_at timestamptz NOT NULL,
    refresh_token text NOT NULL
);

-- default naming scheme for this unique constraint is
-- `<table>_<column>_<key> 
-- hence: reddit_posts_permalink_key
ALTER TABLE IF EXISTS reddit_posts 
    DROP CONSTRAINT IF EXISTS reddit_posts_permalink_key;
DROP INDEX IF EXISTS reddit_posts_permalink_key;

-- add columns in reddit_posts!
ALTER TABLE IF EXISTS reddit_posts 
    ADD COLUMN IF NOT EXISTS link_title text,
    ADD COLUMN IF NOT EXISTS body text;

COMMIT;