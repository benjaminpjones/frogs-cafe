import { describe, it, expect } from "vitest";
import { config, keys } from "../src/setup.js";
import {
  setupGame,
  connectLocalPlayer,
  connectRemotePlayer,
} from "../src/helpers.js";
import type { PlayerConnection } from "../src/helpers.js";

// The local user creates the challenge, so they are black (creator = black in current impl).
// The mock remote user (testbot) accepts, so they are white.

async function setupConnectedGame(): Promise<{
  black: PlayerConnection;
  white: PlayerConnection;
  cleanup: () => void;
}> {
  const game = await setupGame(config);
  const black = await connectLocalPlayer(config, game);
  const white = await connectRemotePlayer(config, keys, game);
  return {
    black,
    white,
    cleanup() {
      black.close();
      white.close();
    },
  };
}

describe("GAMEPLAY", () => {
  it("both players receive game_state on connect", async () => {
    const { black, white, cleanup } = await setupConnectedGame();

    const blackState = black.messages.find((m) => m.type === "game_state");
    const whiteState = white.messages.find((m) => m.type === "game_state");

    expect(blackState).toBeDefined();
    expect(whiteState).toBeDefined();

    if (blackState!.type === "game_state") {
      expect(blackState!.data.phase).toBe("active");
      expect(blackState!.data.moves).toEqual([]);
      expect(blackState!.data.nextToPlay).toBe("black");
    }

    cleanup();
  });

  it("move is broadcast to both players", async () => {
    const { black, white, cleanup } = await setupConnectedGame();

    // Black plays
    black.send({ type: "move", data: { pos: [2, 2] } });

    const blackMsg = await black.waitFor("move_played");
    const whiteMsg = await white.waitFor("move_played");

    expect(blackMsg.type).toBe("move_played");
    expect(whiteMsg.type).toBe("move_played");

    if (blackMsg.type === "move_played") {
      expect(blackMsg.data.pos).toEqual([2, 2]);
      expect(blackMsg.data.moveNumber).toBe(1);
      expect(blackMsg.data.color).toBe("black");
      expect(blackMsg.data.nextToPlay).toBe("white");
    }

    cleanup();
  });

  it("move out of turn is rejected", async () => {
    const { black, white, cleanup } = await setupConnectedGame();

    // White tries to move first (it's black's turn)
    white.send({ type: "move", data: { pos: [3, 3] } });

    const rejection = await white.waitFor("move_rejected");
    expect(rejection.type).toBe("move_rejected");
    if (rejection.type === "move_rejected") {
      expect(rejection.data.reason).toBe("not_your_turn");
    }

    cleanup();
  });

  it("pass move is accepted", async () => {
    const { black, white, cleanup } = await setupConnectedGame();

    black.send({ type: "move", data: { pos: null } });

    const msg = await black.waitFor("move_played");
    expect(msg.type).toBe("move_played");
    if (msg.type === "move_played") {
      expect(msg.data.pos).toBeNull();
      expect(msg.data.color).toBe("black");
    }

    cleanup();
  });

  it("resign ends game for both players", async () => {
    const { black, white, cleanup } = await setupConnectedGame();

    // Play one move then white resigns
    black.send({ type: "move", data: { pos: [2, 2] } });
    await black.waitFor("move_played");

    white.send({ type: "resign", data: {} });

    const blackEnd = await black.waitFor("game_over");
    const whiteEnd = await white.waitFor("game_over");

    expect(blackEnd.type).toBe("game_over");
    expect(whiteEnd.type).toBe("game_over");

    if (blackEnd.type === "game_over") {
      expect(blackEnd.data.winner).toBe("black");
      expect(blackEnd.data.result).toMatch(/^B\+R$/);
    }

    cleanup();
  });

  it("game_state on reconnect includes move history", async () => {
    const game = await setupGame(config);
    const black = await connectLocalPlayer(config, game);
    const white = await connectRemotePlayer(config, keys, game);

    // Play two moves
    black.send({ type: "move", data: { pos: [2, 2] } });
    await white.waitFor("move_played");
    white.send({ type: "move", data: { pos: [6, 6] } });
    await black.waitFor("move_played");

    // Disconnect and reconnect white
    white.close();
    const white2 = await connectRemotePlayer(config, keys, game);

    const state = white2.messages.find((m) => m.type === "game_state");
    expect(state).toBeDefined();
    if (state!.type === "game_state") {
      expect(state!.data.moves.length).toBe(2);
    }

    black.close();
    white2.close();
  });
});
