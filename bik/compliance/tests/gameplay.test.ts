import { describe, it, expect, afterEach } from "vitest";
import { config, keys } from "../src/setup.js";
import {
  setupGame,
  connectLocalPlayer,
  connectRemotePlayer,
} from "../src/helpers.js";
import type { PlayerConnection } from "../src/helpers.js";

const openConnections: PlayerConnection[] = [];

afterEach(() => {
  for (const conn of openConnections) {
    conn.close();
  }
  openConnections.length = 0;
});

async function setupConnectedGame(): Promise<{
  black: PlayerConnection;
  white: PlayerConnection;
}> {
  const game = await setupGame(config, { colorAssignment: "black" });
  const local = await connectLocalPlayer(config, game);
  const remote = await connectRemotePlayer(config, keys, game);
  openConnections.push(local, remote);

  // Assign by actual game response, not assumption
  const localActorURI = `${config.target}/users/${config.username}`;
  const isLocalBlack = game.black === localActorURI;
  const black = isLocalBlack ? local : remote;
  const white = isLocalBlack ? remote : local;

  return { black, white };
}

describe("GAMEPLAY", () => {
  it("both players receive game_state on connect", async () => {
    const { black, white } = await setupConnectedGame();

    const blackState = black.messages.find((m) => m.type === "game_state");
    const whiteState = white.messages.find((m) => m.type === "game_state");

    expect(blackState).toBeDefined();
    expect(whiteState).toBeDefined();

    if (blackState!.type === "game_state") {
      expect(blackState!.data.phase).toBe("active");
      expect(blackState!.data.moves).toEqual([]);
      expect(blackState!.data.nextToPlay).toBe("black");
    }
  });

  it("move is broadcast to both players", async () => {
    const { black, white } = await setupConnectedGame();

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
  });

  it("move out of turn is rejected", async () => {
    const { white } = await setupConnectedGame();

    // White tries to move first (it's black's turn)
    white.send({ type: "move", data: { pos: [3, 3] } });

    const rejection = await white.waitFor("move_rejected");
    expect(rejection.type).toBe("move_rejected");
    if (rejection.type === "move_rejected") {
      expect(rejection.data.reason).toBe("not_your_turn");
    }
  });

  it("pass move is accepted", async () => {
    const { black } = await setupConnectedGame();

    black.send({ type: "move", data: { pos: null } });

    const msg = await black.waitFor("move_played");
    expect(msg.type).toBe("move_played");
    if (msg.type === "move_played") {
      expect(msg.data.pos).toBeNull();
      expect(msg.data.color).toBe("black");
    }
  });

  it("resign ends game for both players", async () => {
    const { black, white } = await setupConnectedGame();

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
  });

  it("game_state on reconnect includes move history", async () => {
    const game = await setupGame(config, { colorAssignment: "black" });
    const local = await connectLocalPlayer(config, game);
    const remote = await connectRemotePlayer(config, keys, game);
    openConnections.push(local, remote);

    const localActorURI = `${config.target}/users/${config.username}`;
    const isLocalBlack = game.black === localActorURI;
    const black = isLocalBlack ? local : remote;
    const white = isLocalBlack ? remote : local;

    // Play two moves
    black.send({ type: "move", data: { pos: [2, 2] } });
    await white.waitFor("move_played");
    white.send({ type: "move", data: { pos: [6, 6] } });
    await black.waitFor("move_played");

    // Disconnect and reconnect remote player
    remote.close();
    const remote2 = await connectRemotePlayer(config, keys, game);
    openConnections.push(remote2);

    const state = remote2.messages.find((m) => m.type === "game_state");
    expect(state).toBeDefined();
    if (state!.type === "game_state") {
      expect(state!.data.moves.length).toBe(2);
    }
  });

  it("random color assignment produces both colors over multiple games", async () => {
    const colors = new Set<string>();

    for (let i = 0; i < 10; i++) {
      const game = await setupGame(config, { colorAssignment: "random" });
      const localActorURI = `${config.target}/users/${config.username}`;
      if (game.black === localActorURI) {
        colors.add("black");
      } else {
        colors.add("white");
      }
      if (colors.size === 2) break;
    }

    expect(colors.size).toBe(2);
  });
});
