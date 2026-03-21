import { describe, it, expect } from "vitest";
import { WebSocketServer, WebSocket as WS } from "ws";
import { BikClient } from "../src/client.js";
import type { ServerMessage, ClientMessage } from "../src/types/ws.js";
import type { Color, Clock, Position } from "../src/types/common.js";

const PORT = 9877;
const WS_IMPL = WS as unknown as typeof WebSocket;

/**
 * A minimal BIK game server that tracks turns, broadcasts moves to both
 * players, and handles resignation. Enough to prove two players can
 * complete a game through the protocol.
 */
function startGameServer() {
  const wss = new WebSocketServer({ port: PORT });
  const clients = new Map<WS, { color: Color }>();
  let nextToPlay: Color = "black";
  let moveNumber = 0;

  const clock = (): Clock => ({
    system: "absolute",
    black: 600,
    white: 600,
    activeColor: nextToPlay,
  });

  const broadcast = (msg: ServerMessage) => {
    const payload = JSON.stringify(msg);
    for (const ws of clients.keys()) ws.send(payload);
  };

  wss.on("connection", (ws) => {
    // First connection is black, second is white
    const color: Color = clients.size === 0 ? "black" : "white";
    clients.set(ws, { color });

    ws.send(JSON.stringify({
      type: "game_state",
      data: { moves: [], phase: "active", nextToPlay, clock: clock() },
    } satisfies ServerMessage));

    ws.on("message", (raw) => {
      const msg = JSON.parse(raw.toString()) as ClientMessage;
      const { color: playerColor } = clients.get(ws)!;

      if (msg.type === "move") {
        if (playerColor !== nextToPlay) {
          ws.send(JSON.stringify({
            type: "move_rejected",
            data: { reason: "not_your_turn" },
          } satisfies ServerMessage));
          return;
        }

        moveNumber++;
        nextToPlay = nextToPlay === "black" ? "white" : "black";

        broadcast({
          type: "move_played",
          data: {
            pos: msg.data.pos,
            moveNumber,
            color: playerColor,
            captures: [],
            nextToPlay,
            clock: clock(),
          },
        });
      }

      if (msg.type === "resign") {
        const winner: Color = playerColor === "black" ? "white" : "black";
        broadcast({
          type: "game_over",
          data: { result: `${winner[0].toUpperCase()}+R`, winner, scoreBlack: null, scoreWhite: null },
        });
      }
    });
  });

  return { wss };
}

function makeClient(messages: ServerMessage[]) {
  return new BikClient(`ws://localhost:${PORT}`, "token", {
    onMessage: (msg) => messages.push(msg),
    WebSocketImpl: WS_IMPL,
  });
}

const wait = (ms = 50) => new Promise((r) => setTimeout(r, ms));

describe("two players play a game", () => {
  it("plays several moves then black resigns", async () => {
    const { wss } = startGameServer();

    const blackMsgs: ServerMessage[] = [];
    const whiteMsgs: ServerMessage[] = [];
    const black = makeClient(blackMsgs);
    const white = makeClient(whiteMsgs);

    black.connect();
    await wait();
    white.connect();
    await wait();

    // Both should have received game_state
    expect(blackMsgs[0].type).toBe("game_state");
    expect(whiteMsgs[0].type).toBe("game_state");

    // Black plays D4
    black.send({ type: "move", data: { pos: [3, 3] } });
    await wait();

    // Both see move_played
    const blackMove = blackMsgs.find((m) => m.type === "move_played");
    const whiteMove = whiteMsgs.find((m) => m.type === "move_played");
    expect(blackMove?.type === "move_played" && blackMove.data.pos).toEqual([3, 3]);
    expect(whiteMove?.type === "move_played" && whiteMove.data.pos).toEqual([3, 3]);

    // White plays Q16
    white.send({ type: "move", data: { pos: [15, 2] } });
    await wait();

    // Black resigns
    black.send({ type: "resign", data: {} });
    await wait();

    // Both should see game_over with white winning
    const blackGameOver = blackMsgs.find((m) => m.type === "game_over");
    const whiteGameOver = whiteMsgs.find((m) => m.type === "game_over");
    expect(blackGameOver?.type === "game_over" && blackGameOver.data.result).toBe("W+R");
    expect(whiteGameOver?.type === "game_over" && whiteGameOver.data.winner).toBe("white");

    black.close();
    white.close();
    wss.close();
  });

  it("rejects a move played out of turn", async () => {
    const { wss } = startGameServer();

    const blackMsgs: ServerMessage[] = [];
    const whiteMsgs: ServerMessage[] = [];
    const black = makeClient(blackMsgs);
    const white = makeClient(whiteMsgs);

    black.connect();
    await wait();
    white.connect();
    await wait();

    // White tries to play first (it's black's turn)
    white.send({ type: "move", data: { pos: [3, 3] } });
    await wait();

    const rejected = whiteMsgs.find((m) => m.type === "move_rejected");
    expect(rejected?.type === "move_rejected" && rejected.data.reason).toBe("not_your_turn");

    black.close();
    white.close();
    wss.close();
  });
});
