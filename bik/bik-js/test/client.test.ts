import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { WebSocketServer, WebSocket as WS } from "ws";
import { BikClient } from "../src/client.js";
import type { ServerMessage, ClientMessage } from "../src/types/ws.js";

const PORT = 9876;
const GAME_URL = `ws://localhost:${PORT}`;
const TOKEN = "test-token";

function mockServer(): {
  wss: WebSocketServer;
  send: (msg: ServerMessage) => void;
  received: ClientMessage[];
  waitForMessage: () => Promise<ClientMessage>;
} {
  const wss = new WebSocketServer({ port: PORT });
  const received: ClientMessage[] = [];
  let conn: WS | null = null;
  const pending: Array<(msg: ClientMessage) => void> = [];

  wss.on("connection", (ws) => {
    conn = ws;

    // Send game_state on connect
    const gameState: ServerMessage = {
      type: "game_state",
      data: {
        moves: [],
        phase: "active",
        nextToPlay: "black",
        clock: { system: "absolute", black: 600, white: 600, activeColor: "black" },
      },
    };
    ws.send(JSON.stringify(gameState));

    ws.on("message", (raw) => {
      const msg = JSON.parse(raw.toString()) as ClientMessage;
      received.push(msg);
      pending.shift()?.(msg);
    });
  });

  return {
    wss,
    send: (msg) => conn?.send(JSON.stringify(msg)),
    received,
    waitForMessage: () =>
      new Promise((resolve) => pending.push(resolve)),
  };
}

describe("BikClient", () => {
  let server: ReturnType<typeof mockServer>;

  beforeEach(() => {
    server = mockServer();
  });

  afterEach(() => {
    server.wss.close();
  });

  it("connects and receives game_state", async () => {
    const messages: ServerMessage[] = [];
    const client = new BikClient(GAME_URL, TOKEN, {
      onMessage: (msg) => messages.push(msg),
      WebSocketImpl: WS as unknown as typeof WebSocket,
    });

    client.connect();
    await new Promise((r) => setTimeout(r, 50));

    expect(messages[0].type).toBe("game_state");
    client.close();
  });

  it("sends a move", async () => {
    const client = new BikClient(GAME_URL, TOKEN, {
      WebSocketImpl: WS as unknown as typeof WebSocket,
    });
    client.connect();
    await new Promise((r) => setTimeout(r, 50));

    client.send({ type: "move", data: { pos: [3, 3] } });
    const received = await server.waitForMessage();

    expect(received).toEqual({ type: "move", data: { pos: [3, 3] } });
    client.close();
  });

  it("sends a pass", async () => {
    const client = new BikClient(GAME_URL, TOKEN, {
      WebSocketImpl: WS as unknown as typeof WebSocket,
    });
    client.connect();
    await new Promise((r) => setTimeout(r, 50));

    client.send({ type: "move", data: { pos: null } });
    const received = await server.waitForMessage();

    expect(received).toEqual({ type: "move", data: { pos: null } });
    client.close();
  });

  it("receives move_played and updates state", async () => {
    const messages: ServerMessage[] = [];
    const client = new BikClient(GAME_URL, TOKEN, {
      onMessage: (msg) => messages.push(msg),
      WebSocketImpl: WS as unknown as typeof WebSocket,
    });
    client.connect();
    await new Promise((r) => setTimeout(r, 50));

    server.send({
      type: "move_played",
      data: {
        pos: [3, 3],
        moveNumber: 1,
        color: "black",
        captures: [],
        nextToPlay: "white",
        clock: { system: "absolute", black: 595, white: 600, activeColor: "white" },
      },
    });
    await new Promise((r) => setTimeout(r, 50));

    const movePlayed = messages.find((m) => m.type === "move_played");
    expect(movePlayed).toBeDefined();
    expect(movePlayed?.type === "move_played" && movePlayed.data.pos).toEqual([3, 3]);
    client.close();
  });

  it("receives game_over", async () => {
    const messages: ServerMessage[] = [];
    const client = new BikClient(GAME_URL, TOKEN, {
      onMessage: (msg) => messages.push(msg),
      WebSocketImpl: WS as unknown as typeof WebSocket,
    });
    client.connect();
    await new Promise((r) => setTimeout(r, 50));

    server.send({
      type: "game_over",
      data: { result: "B+R", winner: "black", scoreBlack: null, scoreWhite: null },
    });
    await new Promise((r) => setTimeout(r, 50));

    const gameOver = messages.find((m) => m.type === "game_over");
    expect(gameOver?.type === "game_over" && gameOver.data.result).toBe("B+R");
    client.close();
  });
});
