import type { GameResult, TimeControl } from "./common.js";

export interface BikChallenge {
  type: "BikChallenge";
  boardSize: number;
  timeControl: TimeControl;
  colorPreference: "black" | "white" | "any";
  expiresAt: string; // ISO 8601
}

export interface BikGame {
  type: "BikGame";
  id: string;
  black: string;     // actor URI
  white: string;     // actor URI
  url: string;       // human-readable game page
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
  object: string; // URI of the challenge Note
}

export interface UndoChallengeActivity {
  "@context": "https://www.w3.org/ns/activitystreams";
  type: "Undo";
  actor: string;
  object: string; // URI of the challenge Note
}

export interface CreateGameActivity {
  "@context": "https://www.w3.org/ns/activitystreams";
  type: "Create";
  actor: string;
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
  object: {
    type: "Note";
    id: string;
    content: string;
    inReplyTo: string;
    attachment: BikGameResult;
  };
}
