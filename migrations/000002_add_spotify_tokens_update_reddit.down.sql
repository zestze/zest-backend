BEGIN;

DROP TABLE IF EXISTS spotify_tokens;

-- TODO(zeke): not idempotent! but can't check if constraint exists simply
ALTER TABLE IF EXISTS reddit_posts ADD CONSTRAINT reddit_posts_permalink_key UNIQUE (permalink);

COMMIT;