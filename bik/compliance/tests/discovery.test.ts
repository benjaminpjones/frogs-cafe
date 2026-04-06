import { describe, it, expect } from "vitest";
import { config } from "../src/setup.js";

describe("DISCOVERY", () => {
  describe("WebFinger", () => {
    it("returns actor link for known user", async () => {
      const domain = new URL(config.target).host;
      const res = await fetch(
        `${config.target}/.well-known/webfinger?resource=acct:${config.username}@${domain}`,
      );
      expect(res.status).toBe(200);
      expect(res.headers.get("content-type")).toContain("jrd+json");

      const data = (await res.json()) as {
        subject: string;
        links: Array<{ rel: string; type: string; href: string }>;
      };
      expect(data.subject).toBe(`acct:${config.username}@${domain}`);
      expect(data.links).toBeInstanceOf(Array);

      const self = data.links.find((l) => l.rel === "self");
      expect(self).toBeDefined();
      expect(self!.type).toBe("application/activity+json");
      expect(self!.href).toContain(config.username);
    });

    it("returns 404 for unknown user", async () => {
      const domain = new URL(config.target).host;
      const res = await fetch(
        `${config.target}/.well-known/webfinger?resource=acct:nonexistent_user_xyz@${domain}`,
      );
      expect(res.status).toBe(404);
    });

    it("returns 400 for non-acct resource", async () => {
      const res = await fetch(
        `${config.target}/.well-known/webfinger?resource=http://example.com`,
      );
      expect(res.status).toBe(400);
    });
  });

  describe("Actor Document", () => {
    it("returns valid AP actor with required fields", async () => {
      const domain = new URL(config.target).host;
      const wfRes = await fetch(
        `${config.target}/.well-known/webfinger?resource=acct:${config.username}@${domain}`,
      );
      const wfData = (await wfRes.json()) as {
        links: Array<{ rel: string; href: string }>;
      };
      const actorUrl = wfData.links.find((l) => l.rel === "self")!.href;

      const res = await fetch(actorUrl, {
        headers: { Accept: "application/activity+json" },
      });
      expect(res.status).toBe(200);

      const actor = (await res.json()) as Record<string, unknown>;
      expect(actor["@context"]).toBe("https://www.w3.org/ns/activitystreams");
      expect(actor.type).toBe("Person");
      expect(actor.id).toBe(actorUrl);
      expect(actor.preferredUsername).toBe(config.username);
      expect(actor.inbox).toBeDefined();
      expect(actor.followers).toBeDefined();
    });
  });

  describe("JWK Keys", () => {
    it("returns valid Ed25519 key set", async () => {
      const res = await fetch(`${config.target}/.well-known/bik/keys`);
      expect(res.status).toBe(200);

      const data = (await res.json()) as {
        keys: Array<{ kty: string; crv: string; kid: string; x: string }>;
      };
      expect(data.keys).toBeInstanceOf(Array);
      expect(data.keys.length).toBeGreaterThan(0);

      for (const key of data.keys) {
        expect(key.kty).toBe("OKP");
        expect(key.crv).toBe("Ed25519");
        expect(key.kid).toBeDefined();
        expect(typeof key.kid).toBe("string");
        expect(key.x).toBeDefined();
        // Ed25519 public key is 32 bytes = 43 base64url characters (no padding)
        expect(key.x.length).toBe(43);
      }
    });
  });
});
