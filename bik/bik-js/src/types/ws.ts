import type { Position, Color, Clock, GamePhase, GameResult } from "./common.js";

// ---- Client → Host ----

export type ClientMessage =
  | { type: "move";         data: { pos: Position } }
  | { type: "resign";       data: Record<string, never> }
  | { type: "mark_dead";    data: { positions: Position[] } }
  | { type: "score_accept"; data: Record<string, never> }
  | { type: "score_reject"; data: Record<string, never> };

// ---- Host → Clients ----

export interface GameStateData {
  moves: Position[];
  phase: GamePhase;
  nextToPlay: Color;
  clock: Clock;
}

export interface MovePlayed {
  pos: Position;
  moveNumber: number;
  color: Color;
  captures: Position[];
  nextToPlay: Color;
  clock: Clock;
}

export type MoveRejectedReason =
  | "ko"
  | "occupied"
  | "suicide"
  | "not_your_turn"
  | "game_not_active";

export type ServerMessage =
  | { type: "game_state";   data: GameStateData }
  | { type: "move_played";  data: MovePlayed }
  | { type: "move_rejected"; data: { reason: MoveRejectedReason } }
  | { type: "clock";        data: Clock }
  | { type: "phase_change"; data: { phase: GamePhase } }
  | { type: "dead_stones";  data: { positions: Position[] } }
  | { type: "game_over";    data: { result: GameResult; winner: Color; scoreBlack: number | null; scoreWhite: number | null } };
