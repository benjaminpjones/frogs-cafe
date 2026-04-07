ALTER TABLE moves ADD CONSTRAINT moves_game_id_move_number_unique UNIQUE (game_id, move_number);
