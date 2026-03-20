import { describe, it, expect, beforeEach, afterEach } from "vitest";
import * as http from "http";
import {
  buildCreateChallenge,
  buildAcceptChallenge,
  buildUndoChallenge,
  buildCreateGame,
  buildCreateGameResult,
  postActivity,
} from "../src/ap.js";
import type { CreateChallengeActivity, AcceptChallengeActivity, UndoChallengeActivity, CreateGameActivity } from "../src/types/ap.js";

const SERVER_A = "https://server-a.example";
const SERVER_B = "https://server-b.example";
const ALICE = `${SERVER_A}/users/alice`;
const BOB = `${SERVER_B}/users/bob`;
const CHALLENGE_URI = `${SERVER_A}/challenges/abc123`;
const GAME_URI = `${SERVER_A}/games/xyz789`;

const inbox = (port: number) => `http://localhost:${port}/inbox`;

/** Simple mock HTTP inbox that records received activities */
async function startInbox(port: number, respondWith: (activity: object) => object) {
  const received: object[] = [];

  const server = http.createServer((req, res) => {
    let body = "";
    req.on("data", (chunk) => (body += chunk));
    req.on("end", () => {
      const activity = JSON.parse(body) as object;
      received.push(activity);
      const reply = respondWith(activity);
      res.writeHead(200, { "Content-Type": "application/activity+json" });
      res.end(JSON.stringify(reply));
    });
  });

  await new Promise<void>((resolve) => server.listen(port, resolve));
  return { server, received };
}

describe("AP activity builders", () => {
  it("builds a valid CreateChallenge activity", () => {
    const activity = buildCreateChallenge(
      ALICE,
      CHALLENGE_URI,
      {
        type: "BikChallenge",
        boardSize: 19,
        timeControl: { system: "byoyomi", mainTime: 600, periods: 5, periodTime: 30 },
        colorPreference: "any",
        expiresAt: "2024-03-10T15:30:00Z",
      },
      "Looking for a game! 19x19, 10min + 5x30s byo-yomi",
    );

    expect(activity["@context"]).toBe("https://www.w3.org/ns/activitystreams");
    expect(activity.type).toBe("Create");
    expect(activity.actor).toBe(ALICE);
    expect(activity.object.attachment.type).toBe("BikChallenge");
    expect(activity.object.attachment.boardSize).toBe(19);
  });

  it("builds a valid AcceptChallenge activity", () => {
    const activity = buildAcceptChallenge(BOB, CHALLENGE_URI);
    expect(activity.type).toBe("Accept");
    expect(activity.actor).toBe(BOB);
    expect(activity.object).toBe(CHALLENGE_URI);
  });

  it("builds a valid UndoChallenge activity", () => {
    const activity = buildUndoChallenge(ALICE, CHALLENGE_URI);
    expect(activity.type).toBe("Undo");
    expect(activity.object).toBe(CHALLENGE_URI);
  });
});

describe("challenge/accept flow", () => {
  async function withInbox(port: number, respondWith: (activity: object) => object) {
    const srv = await startInbox(port, respondWith);
    return {
      ...srv,
      url: inbox(port),
      [Symbol.asyncDispose]: async () => {
        srv.server.closeAllConnections();
        await new Promise<void>((resolve) => srv.server.close(() => resolve()));
      },
    };
  }

  it("server B accepts a challenge and server A creates the game", async () => {
    await using srv = await withInbox(9878, () =>
      buildCreateGame(
        ALICE, GAME_URI, CHALLENGE_URI,
        { type: "BikGame", id: "xyz789", black: ALICE, white: BOB,
          url: `${SERVER_A}/games/xyz789`, websocket: `wss://server-a.example/ws/games/xyz789` },
        "@alice vs @bob@server-b.example — Game started!",
      ),
    );

    const accept = buildAcceptChallenge(BOB, CHALLENGE_URI);
    const res = await postActivity(srv.url, accept);
    const game = await res.json() as CreateGameActivity;

    expect(res.status).toBe(200);
    expect(game.type).toBe("Create");
    expect(game.object.attachment.type).toBe("BikGame");
    expect(game.object.attachment.websocket).toBe("wss://server-a.example/ws/games/xyz789");
    expect(game.object.attachment.black).toBe(ALICE);
    expect(game.object.attachment.white).toBe(BOB);

    expect(srv.received).toHaveLength(1);
    const received = srv.received[0] as AcceptChallengeActivity;
    expect(received.type).toBe("Accept");
    expect(received.object).toBe(CHALLENGE_URI);
  });

  it("server A withdraws the challenge after it is accepted", async () => {
    const activities: object[] = [];
    await using srv = await withInbox(9879, (a) => { activities.push(a); return { ok: true }; });

    await postActivity(srv.url, buildAcceptChallenge(BOB, CHALLENGE_URI));
    await postActivity(srv.url, buildUndoChallenge(ALICE, CHALLENGE_URI));

    expect(activities).toHaveLength(2);
    expect((activities[0] as AcceptChallengeActivity).type).toBe("Accept");
    expect((activities[1] as UndoChallengeActivity).type).toBe("Undo");
    expect((activities[1] as UndoChallengeActivity).object).toBe(CHALLENGE_URI);
  });

  it("server A announces the game result", async () => {
    await using srv = await withInbox(9880, () => ({ ok: true }));

    const result = buildCreateGameResult(
      ALICE, `${GAME_URI}/result`, GAME_URI,
      { type: "BikGameResult", winner: ALICE, result: "B+R", sgf: `${GAME_URI}.sgf` },
      "@alice (B) defeated @bob@server-b.example (W) by resignation.",
    );
    const res = await postActivity(srv.url, result);

    expect(res.status).toBe(200);
    const received = srv.received[0] as CreateGameActivity;
    expect(received.object.attachment.type).toBe("BikGameResult");
    expect((received.object.attachment as { result: string }).result).toBe("B+R");
  });
});
