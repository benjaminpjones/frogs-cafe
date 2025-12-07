export interface Player {
  id: number;
  username: string;
  email: string;
  rating: number;
  created_at: string;
  updated_at: string;
}

export interface Game {
  id: number;
  black_player_id: number | null;
  white_player_id: number | null;
  board_size: number;
  status: "waiting" | "active" | "finished";
  winner_id: number | null;
  creator_id: number | null;
  created_at: string;
  updated_at: string;
}

export interface Move {
  id: number;
  game_id: number;
  player_id: number;
  move_number: number;
  x: number;
  y: number;
  created_at: string;
}

export interface WebSocketMessage {
  type: 'move' | 'join' | 'leave' | 'chat';
  data: any;
}

export interface AuthResponse {
  token: string;
  player: Player;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}
