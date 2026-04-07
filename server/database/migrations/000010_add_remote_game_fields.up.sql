-- For games hosted on remote servers, store the remote game URI and WebSocket URL
ALTER TABLE games ADD COLUMN IF NOT EXISTS remote_game_uri TEXT;
ALTER TABLE games ADD COLUMN IF NOT EXISTS remote_ws_url TEXT;
