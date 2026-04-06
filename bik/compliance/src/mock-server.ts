import { createServer, type IncomingMessage, type ServerResponse } from "node:http";
import type { BikKeys } from "./keys.js";
import { jwkSet } from "./keys.js";

export interface MockServerOptions {
  port: number;
  externalUrl: string;
  keys: BikKeys;
}

export interface ReceivedActivity {
  body: unknown;
  receivedAt: number;
}

export interface MockServer {
  /** Activities received at /bik/inbox */
  inbox: ReceivedActivity[];
  /** Clear the inbox */
  clearInbox(): void;
  /** Stop the server */
  close(): Promise<void>;
}

export async function startMockServer(
  opts: MockServerOptions,
): Promise<MockServer> {
  const inbox: ReceivedActivity[] = [];
  const username = "testbot";
  const actorUrl = `${opts.externalUrl}/users/${username}`;

  const server = createServer((req: IncomingMessage, res: ServerResponse) => {
    const url = new URL(req.url ?? "/", `http://localhost:${opts.port}`);

    // WebFinger
    if (url.pathname === "/.well-known/webfinger") {
      const resource = url.searchParams.get("resource");
      const domain = new URL(opts.externalUrl).host;
      if (resource !== `acct:${username}@${domain}`) {
        res.writeHead(404);
        res.end();
        return;
      }
      res.writeHead(200, { "Content-Type": "application/jrd+json" });
      res.end(
        JSON.stringify({
          subject: resource,
          links: [
            {
              rel: "self",
              type: "application/activity+json",
              href: actorUrl,
            },
          ],
        }),
      );
      return;
    }

    // Actor document
    if (url.pathname === `/users/${username}`) {
      res.writeHead(200, { "Content-Type": "application/activity+json" });
      res.end(
        JSON.stringify({
          "@context": "https://www.w3.org/ns/activitystreams",
          type: "Person",
          id: actorUrl,
          preferredUsername: username,
          inbox: `${opts.externalUrl}/bik/inbox`,
          followers: `${actorUrl}/followers`,
        }),
      );
      return;
    }

    // JWK keys
    if (url.pathname === "/.well-known/bik/keys") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify(jwkSet(opts.keys)));
      return;
    }

    // Inbox
    if (url.pathname === "/bik/inbox" && req.method === "POST") {
      let body = "";
      req.on("data", (chunk: Buffer) => {
        body += chunk.toString();
      });
      req.on("end", () => {
        try {
          inbox.push({ body: JSON.parse(body), receivedAt: Date.now() });
        } catch {
          inbox.push({ body, receivedAt: Date.now() });
        }
        res.writeHead(202);
        res.end();
      });
      return;
    }

    res.writeHead(404);
    res.end();
  });

  await new Promise<void>((resolve) => {
    server.listen(opts.port, () => resolve());
  });

  return {
    inbox,
    clearInbox() {
      inbox.length = 0;
    },
    close() {
      return new Promise((resolve, reject) => {
        server.close((err) => (err ? reject(err) : resolve()));
      });
    },
  };
}
