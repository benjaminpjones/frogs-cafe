-- Allow moves to be made by remote (federated) players who have no local player record
ALTER TABLE moves
    ALTER COLUMN player_id DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS actor_uri TEXT;
