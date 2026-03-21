import type { GameResult, TimeControl } from "./common.js";

export interface BikChallenge {
  type: "BikChallenge";
  boardSize: number;
  timeControl: TimeControl;
  colorAssignment: "black" | "white" | "random";
  komi?: number;     // default 0
  expiresAt: string; // ISO 8601
}

export interface BikGame {
  type: "BikGame";
  id: string;        // full game URI (human-readable page)
  black: string;     // actor URI
  white: string;     // actor URI
  komi?: number;     // default 0
  websocket: string; // wss:// URL
}

export interface BikGameResult {
  type: "BikGameResult";
  winner: string;    // actor URI
  result: GameResult;
  sgf?: string;      // URI
}

export interface CreateChallengeActivity {
  "@context": "https://www.w3.org/ns/activitystreams";
  type: "Create";
  actor: string;
  to: string[];
  cc: string[];
  object: {
    type: "Note";
    id: string;
    attributedTo: string;
    content: string;
    attachment: BikChallenge;
  };
}

export interface AcceptChallengeActivity {
  "@context": "https://www.w3.org/ns/activitystreams";
  type: "Accept";
  actor: string;
  to: string[];  // [challenger actor URI]
  object: string; // URI of the challenge Note
}

export interface UndoChallengeActivity {
  "@context": "https://www.w3.org/ns/activitystreams";
  type: "Undo";
  actor: string;
  to: string[];
  cc: string[];
  object: string; // URI of the challenge Note
}

export interface CreateGameActivity {
  "@context": "https://www.w3.org/ns/activitystreams";
  type: "Create";
  actor: string;
  to: string[];
  cc: string[];  // includes both players and host's followers
  object: {
    type: "Note";
    id: string;
    content: string;
    inReplyTo: string;
    attachment: BikGame;
  };
}

export interface CreateGameResultActivity {
  "@context": "https://www.w3.org/ns/activitystreams";
  type: "Create";
  actor: string;
  to: string[];
  cc: string[];  // includes both players and host's followers
  object: {
    type: "Note";
    id: string;
    content: string;
    inReplyTo: string;
    attachment: BikGameResult;
  };
}
