import { WebSocket } from "ws";
import { BikClient } from "../../bik-js/src/index.js";
import type { ServerMessage, ClientMessage } from "../../bik-js/src/index.js";
import { buildAcceptChallenge, postActivity } from "../../bik-js/src/index.js";
import type { Config } from "./config.js";
import type { BikKeys } from "./keys.js";
import { signToken } from "./keys.js";

const WS_IMPL = WebSocket as unknown as typeof globalThis.WebSocket;

export interface GameSetup {
  gameURI: string;
  wsURL: string;
  black: string;
  white: string;
  challengeURI: string;
}

export interface PlayerConnection {
  send(msg: ClientMessage): void;
  waitFor(type: string, timeoutMs?: number): Promise<ServerMessage>;
  messages: ServerMessage[];
  close(): void;
}

/** Create a challenge on the target server using the test account. */
export async function createChallenge(
  config: Config,
  opts: {
    boardSize?: number;
    expiresIn?: number;
    colorAssignment?: string;
  } = {},
): Promise<string> {
  const res = await fetch(`${config.target}/api/v1/bik/challenges`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${config.token}`,
    },
    body: JSON.stringify({
      boardSize: opts.boardSize ?? 9,
      timeControl: { system: "absolute", mainTime: 300 },
      colorAssignment: opts.colorAssignment ?? "random",
      expiresIn: opts.expiresIn ?? 3600,
    }),
  });
  if (!res.ok) {
    throw new Error(`createChallenge failed: ${res.status} ${await res.text()}`);
  }
  const data = (await res.json()) as { object: { id: string } };
  return data.object.id;
}

/** Accept a challenge as the mock server's test user. Returns game info. */
export async function acceptChallenge(
  config: Config,
  challengeURI: string,
): Promise<GameSetup> {
  const mockActorURI = `${config.mockUrl}/users/testbot`;
  const accept = buildAcceptChallenge(mockActorURI, challengeURI);
  const res = await postActivity(`${config.target}/bik/inbox`, accept);
  if (!res.ok) {
    throw new Error(`acceptChallenge failed: ${res.status} ${await res.text()}`);
  }
  const activity = (await res.json()) as {
    object: {
      id: string;
      attachment: {
        id: string;
        black: string;
        white: string;
        websocket: string;
      };
    };
  };
  const game = activity.object.attachment;
  return {
    gameURI: game.id,
    wsURL: game.websocket,
    black: game.black,
    white: game.white,
    challengeURI,
  };
}

/** Create a challenge and accept it in one step. */
export async function setupGame(
  config: Config,
  opts?: { boardSize?: number; colorAssignment?: string },
): Promise<GameSetup> {
  const challengeURI = await createChallenge(config, opts);
  return acceptChallenge(config, challengeURI);
}

/** Connect to a game WebSocket and return a controllable connection. */
export function connectPlayer(
  wsURL: string,
  token: string,
): Promise<PlayerConnection> {
  return new Promise((resolve, reject) => {
    const messages: ServerMessage[] = [];
    const waiters: Array<{
      type: string;
      resolve: (msg: ServerMessage) => void;
      reject: (err: Error) => void;
    }> = [];

    let consumed = 0; // tracks how many messages have been "seen" by waitFor

    const waiterConsumed = new Set<number>(); // indices of messages consumed by waiters


    const client = new BikClient(wsURL, token, {
      onReady() {
        resolve(conn);
      },
      onMessage(msg) {
        const idx = messages.length;
        messages.push(msg);
        // Check if any waiter matches
        for (let i = waiters.length - 1; i >= 0; i--) {
          if (waiters[i].type === msg.type) {
            const w = waiters.splice(i, 1)[0];
            waiterConsumed.add(idx);
            w.resolve(msg);
            return; // one message satisfies one waiter
          }
        }
      },
      onError(err) {
        reject(err);
      },
      onClose() {
        for (const w of waiters) {
          w.reject(new Error(`WebSocket closed while waiting for ${w.type}`));
        }
        waiters.length = 0;
      },
      WebSocketImpl: WS_IMPL,
    });

    const conn: PlayerConnection = {
      messages,
      send(msg: ClientMessage) {
        client.send(msg);
      },
      waitFor(type: string, timeoutMs = 5000): Promise<ServerMessage> {
        // Check unconsumed messages (skip ones already consumed by live waiters)
        for (let i = consumed; i < messages.length; i++) {
          if (!waiterConsumed.has(i) && messages[i].type === type) {
            consumed = i + 1;
            return Promise.resolve(messages[i]);
          }
        }
        consumed = messages.length;

        // Wait for a future message
        return new Promise((res, rej) => {
          const waiter = {
            type,
            resolve(msg: ServerMessage) {
              clearTimeout(timer);
              res(msg);
            },
            reject(err: Error) {
              clearTimeout(timer);
              rej(err);
            },
          };

          const timer = setTimeout(() => {
            const idx = waiters.indexOf(waiter);
            if (idx >= 0) waiters.splice(idx, 1);
            rej(new Error(`Timed out waiting for message type: ${type}`));
          }, timeoutMs);

          waiters.push(waiter);
        });
      },
      close() {
        client.close();
      },
    };

    client.connect();
  });
}

/** Connect the local test user (uses session token directly). */
export function connectLocalPlayer(
  config: Config,
  game: GameSetup,
): Promise<PlayerConnection> {
  return connectPlayer(game.wsURL, config.token);
}

/** Connect the mock remote user (signs a BIK token). */
export function connectRemotePlayer(
  config: Config,
  keys: BikKeys,
  game: GameSetup,
): Promise<PlayerConnection> {
  const mockActorURI = `${config.mockUrl}/users/testbot`;
  const token = signToken(keys, mockActorURI, game.gameURI);
  return connectPlayer(game.wsURL, token);
}
