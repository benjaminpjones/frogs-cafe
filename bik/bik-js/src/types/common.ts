export type Position = [number, number] | null;

export type Color = "black" | "white";

export type GamePhase = "active" | "scoring" | "finished";

/** SGF-style result: B+R, W+3.5, B+T, etc. */
export type GameResult = string;

export interface ByoyomiPlayerClock {
  main: number;       // seconds remaining
  periods: number;
  periodTime: number; // seconds per period
}

export type Clock =
  | { system: "byoyomi"; black: ByoyomiPlayerClock; white: ByoyomiPlayerClock; activeColor: Color }
  | { system: "fischer"; black: number; white: number; increment: number; activeColor: Color }
  | { system: "absolute"; black: number; white: number; activeColor: Color };

export type TimeControl =
  | { system: "byoyomi"; mainTime: number; periods: number; periodTime: number }
  | { system: "fischer"; mainTime: number; increment: number }
  | { system: "absolute"; mainTime: number };
