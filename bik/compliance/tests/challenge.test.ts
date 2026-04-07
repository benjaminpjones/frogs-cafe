import { describe, it, expect } from "vitest";
import { config } from "../src/setup.js";
import { createChallenge, acceptChallenge } from "../src/helpers.js";

describe("CHALLENGE FLOW", () => {
  it("create challenge returns Create activity with BikChallenge", async () => {
    const res = await fetch(`${config.target}/api/v1/bik/challenges`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${config.token}`,
      },
      body: JSON.stringify({
        boardSize: 9,
        timeControl: { system: "absolute", mainTime: 300 },
        colorAssignment: "random",
        expiresIn: 3600,
      }),
    });
    expect(res.status).toBe(201);

    const activity = (await res.json()) as Record<string, unknown>;
    expect(activity["@context"]).toBe("https://www.w3.org/ns/activitystreams");
    expect(activity.type).toBe("Create");
    expect(activity.actor).toBeDefined();

    const obj = activity.object as Record<string, unknown>;
    expect(obj.type).toBe("Note");
    expect(obj.id).toBeDefined();
    expect(obj.attributedTo).toBe(activity.actor);

    const attachment = obj.attachment as Record<string, unknown>;
    expect(attachment.type).toBe("BikChallenge");
    expect(attachment.boardSize).toBe(9);
    expect(attachment.colorAssignment).toBe("random");
    expect(attachment.expiresAt).toBeDefined();
  });

  it("accept challenge creates game with BikGame", async () => {
    const challengeURI = await createChallenge(config);
    const game = await acceptChallenge(config, challengeURI);

    expect(game.gameURI).toBeDefined();
    expect(game.wsURL).toBeDefined();
    expect(game.black).toBeDefined();
    expect(game.white).toBeDefined();
    // Both players should be represented
    const players = [game.black, game.white];
    expect(players.some((p) => p.includes("testbot"))).toBe(true);
    expect(players.some((p) => p.includes(config.username))).toBe(true);
  });

  it("rejects accept for expired challenge", async () => {
    const challengeURI = await createChallenge(config, { expiresIn: 1 });
    // Wait for expiry
    await new Promise((r) => setTimeout(r, 2000));

    const mockActorURI = `${config.mockUrl}/users/testbot`;
    const accept = {
      "@context": "https://www.w3.org/ns/activitystreams",
      type: "Accept",
      actor: mockActorURI,
      object: challengeURI,
    };
    const res = await fetch(`${config.target}/bik/inbox`, {
      method: "POST",
      headers: { "Content-Type": "application/activity+json" },
      body: JSON.stringify(accept),
    });
    expect(res.status).toBe(410);
  });

  it("rejects double accept", async () => {
    const challengeURI = await createChallenge(config);
    await acceptChallenge(config, challengeURI);

    // Second accept should fail
    const mockActorURI = `${config.mockUrl}/users/testbot`;
    const accept = {
      "@context": "https://www.w3.org/ns/activitystreams",
      type: "Accept",
      actor: mockActorURI,
      object: challengeURI,
    };
    const res = await fetch(`${config.target}/bik/inbox`, {
      method: "POST",
      headers: { "Content-Type": "application/activity+json" },
      body: JSON.stringify(accept),
    });
    expect(res.status).toBe(409);
  });
});
