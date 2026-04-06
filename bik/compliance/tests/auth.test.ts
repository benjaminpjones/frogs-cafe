import { describe, it, expect } from "vitest";
import { config, keys } from "../src/setup.js";
import { setupGame, connectPlayer } from "../src/helpers.js";
import { signToken, generateKeys } from "../src/keys.js";

describe("AUTH", () => {
  it("valid BIK token authenticates to WebSocket", async () => {
    const game = await setupGame(config);
    const mockActorURI = `${config.mockUrl}/users/testbot`;
    const token = signToken(keys, mockActorURI, game.gameURI);

    const player = await connectPlayer(game.wsURL, token);
    const state = await player.waitFor("game_state");
    expect(state.type).toBe("game_state");
    player.close();
  });

  it("expired token is rejected", async () => {
    const game = await setupGame(config);
    const mockActorURI = `${config.mockUrl}/users/testbot`;
    // Token that expired 60 seconds ago
    const token = signToken(keys, mockActorURI, game.gameURI, -60);

    try {
      const player = await connectPlayer(game.wsURL, token);
      // If we get here, the server didn't reject — that's a failure
      player.close();
      expect.fail("Expected connection to be rejected for expired token");
    } catch {
      // Connection rejected or closed — expected
    }
  });

  it("token signed with unknown key is rejected", async () => {
    const game = await setupGame(config);
    const mockActorURI = `${config.mockUrl}/users/testbot`;
    const wrongKeys = generateKeys();
    const token = signToken(wrongKeys, mockActorURI, game.gameURI);

    try {
      const player = await connectPlayer(game.wsURL, token);
      player.close();
      expect.fail("Expected connection to be rejected for unknown key");
    } catch {
      // Connection rejected — expected
    }
  });
});
