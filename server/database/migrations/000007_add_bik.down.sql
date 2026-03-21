ALTER TABLE games
    DROP COLUMN IF EXISTS black_actor_uri,
    DROP COLUMN IF EXISTS white_actor_uri,
    DROP COLUMN IF EXISTS bik_challenge_uri;

DROP TABLE IF EXISTS bik_challenges;
