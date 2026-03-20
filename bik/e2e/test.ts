/**
 * BIK E2E test: two frogs_cafe instances play a game across servers.
 *
 * Server A hosts the game. Alice is a local player on A.
 * Server B is the guest. Bob is a local player on B.
 *
 * Flow:
 *  1. Register alice on server A, bob on server B
 *  2. Alice creates a BIK challenge on server A
 *  3. Bob (on server B) accepts the challenge → server A creates a game
 *  4. Bob gets a BIK token from server B for the game
 *  5. Both connect to server A's WebSocket
 *  6. They alternate moves; bob resigns
 *  7. Both receive game_over
 */

import { WebSocket } from "ws";
import { buildAcceptChallenge, postActivity, BikClient } from "../bik-js/src/index.js";
import type { ServerMessage, CreateGameActivity } from "../bik-js/src/index.js";

const A = process.env.SERVER_A ?? "http://localhost:8080";
const B = process.env.SERVER_B ?? "http://localhost:8081";

const WS_A = A.replace("http", "ws");
const WS_IMPL = WebSocket as unknown as typeof globalThis.WebSocket;

async function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms));
}

async function waitForServer(url: string, retries = 20) {
  for (let i = 0; i < retries; i++) {
    try {
      const res = await fetch(`${url}/health`);
      if (res.ok) return;
    } catch {}
    await sleep(1000);
  }
  throw new Error(`Server not ready: ${url}`);
}

async function register(server: string, username: string, password: string) {
  const res = await fetch(`${server}/api/v1/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, email: `${username}@example.com`, password }),
  });
  if (!res.ok) throw new Error(`register ${username} failed: ${res.status} ${await res.text()}`);
  const data = await res.json() as { token: string };
  return data.token;
}

async function createChallenge(server: string, token: string) {
  const res = await fetch(`${server}/api/v1/bik/challenges`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({
      boardSize: 9,
      timeControl: { system: "absolute", mainTime: 300 },
      colorPreference: "any",
      expiresIn: 3600,
    }),
  });
  if (!res.ok) throw new Error(`createChallenge failed: ${res.status} ${await res.text()}`);
  return res.json() as Promise<{ object: { id: string } }>;
}

async function getBIKToken(server: string, token: string, gameID: string) {
  const res = await fetch(`${server}/api/v1/bik/token/${gameID}`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error(`getBIKToken failed: ${res.status} ${await res.text()}`);
  const data = await res.json() as { token: string };
  return data.token;
}

function connectAndCollect(url: string, token: string): { messages: ServerMessage[]; client: BikClient } {
  const messages: ServerMessage[] = [];
  const client = new BikClient(url, token, {
    onMessage: (msg) => messages.push(msg),
    WebSocketImpl: WS_IMPL,
  });
  client.connect();
  return { messages, client };
}

// ---- main ----

console.log("Waiting for servers...");
await waitForServer(A);
await waitForServer(B);
console.log("✓ Both servers ready\n");

// 1. Register users
console.log("Registering alice on server A and bob on server B...");
const aliceToken = await register(A, "alice", "password123");
const bobToken   = await register(B, "bob",   "password123");
console.log("✓ Users registered\n");

// 2. Alice creates a challenge on server A
console.log("Alice creates a challenge...");
const challenge = await createChallenge(A, aliceToken);
const challengeURI = challenge.object.id;
console.log(`✓ Challenge created: ${challengeURI}\n`);

// 3. Bob accepts the challenge — server A responds with a CreateGame activity
console.log("Bob accepts the challenge...");
const accept = buildAcceptChallenge(`${B}/users/bob`, challengeURI);
const res = await postActivity(`${A}/bik/inbox`, accept);
if (!res.ok) throw new Error(`Accept failed: ${res.status} ${await res.text()}`);
const gameActivity = await res.json() as CreateGameActivity;
const game = gameActivity.object.attachment;
const gameID = game.id;
console.log(`✓ Game created: ${game.url}`);
console.log(`  Black: ${game.black}`);
console.log(`  White: ${game.white}`);
console.log(`  WebSocket: ${game.websocket}\n`);

// 4. Bob gets a BIK token from server B
console.log("Bob gets a BIK token from server B...");
const bikToken = await getBIKToken(B, bobToken, gameID);
console.log("✓ BIK token issued\n");

// 5. Both connect to server A's WebSocket
console.log("Connecting to server A's WebSocket...");
const wsURL = `${WS_A}/ws/games/${gameID}`;
const { messages: aliceMsgs, client: alice } = connectAndCollect(wsURL, aliceToken);
await sleep(200);
const { messages: bobMsgs, client: bob } = connectAndCollect(wsURL, bikToken);
await sleep(200);

if (!aliceMsgs.find(m => m.type === "game_state")) throw new Error("Alice did not receive game_state");
if (!bobMsgs.find(m => m.type === "game_state")) throw new Error("Bob did not receive game_state");
console.log("✓ Both players connected and received game_state\n");

// 6. Play some moves (alice is black, bob is white)
console.log("Playing moves...");
alice.send({ type: "move", data: { pos: [2, 2] } }); // alice plays
await sleep(200);
bob.send({ type: "move", data: { pos: [6, 6] } });   // bob plays
await sleep(200);
alice.send({ type: "move", data: { pos: [2, 6] } }); // alice plays
await sleep(200);

const movesPlayed = aliceMsgs.filter(m => m.type === "move_played");
console.log(`✓ ${movesPlayed.length} moves played\n`);

// 7. Bob resigns
console.log("Bob resigns...");
bob.send({ type: "resign", data: {} });
await sleep(200);

const aliceGameOver = aliceMsgs.find(m => m.type === "game_over");
const bobGameOver   = bobMsgs.find(m => m.type === "game_over");

if (!aliceGameOver || !bobGameOver) throw new Error("game_over not received by both players");

console.log("✓ game_over received by both players");
console.log(`  Result: ${aliceGameOver.type === "game_over" ? aliceGameOver.data.result : "?"}\n`);

alice.close();
bob.close();

console.log("🎉 E2E test passed — two servers, real WebSocket, real database");
