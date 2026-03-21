# bik-js

TypeScript client library for the [BIK (Baduk InterKonnect)](../spec/protocol.md) protocol.

## Install

```bash
npm install bik-js
```

For Node.js environments, also install the `ws` package:

```bash
npm install ws
```

## Usage

### Connecting to a game

```ts
import { BikClient } from "bik-js";

const client = new BikClient(
  "wss://server-a.example/ws/games/xyz789", // from the BikGame AP activity
  "<token>",                                 // issued by your home server
);

client.connect();
```

In Node.js, pass the `ws` WebSocket implementation:

```ts
import { WebSocket } from "ws";

const client = new BikClient(url, token, { WebSocketImpl: WebSocket });
client.connect();
```

### Playing a game

```ts
const client = new BikClient(url, token, {
  onReady(msg) {
    // msg.type === "game_state" — received on connect
    console.log("game ready, moves so far:", msg.data.moves);
  },
  onMessage(msg) {
    switch (msg.type) {
      case "move_played":
        console.log(`${msg.data.color} played ${msg.data.pos}, move #${msg.data.moveNumber}`);
        break;
      case "move_rejected":
        console.warn("move rejected:", msg.data.reason);
        break;
      case "clock":
        console.log("clock update:", msg.data);
        break;
      case "game_over":
        console.log("result:", msg.data.result);
        break;
    }
  },
});

client.connect();

// Place a stone at col 3, row 3
client.send({ type: "move", data: { pos: [3, 3] } });

// Pass
client.send({ type: "move", data: { pos: null } });

// Resign
client.send({ type: "resign", data: {} });
```

### Scoring phase

When both players pass, the server sends `{ type: "phase_change", data: { phase: "scoring" } }`.

```ts
// Mark groups as dead
client.send({ type: "mark_dead", data: { positions: [[3, 3], [4, 3]] } });

// Accept the score
client.send({ type: "score_accept", data: {} });

// Reject and resume play
client.send({ type: "score_reject", data: {} });
```

## Matchmaking (ActivityPub)

BIK uses ActivityPub for challenge advertisement and acceptance. Use the activity builders to construct valid AP payloads, then POST them to the target server's inbox.

### Advertising a challenge

```ts
import { buildCreateChallenge, postActivity } from "bik-js";

const activity = buildCreateChallenge(
  "https://frogs.cafe/users/alice",           // your actor URI
  "https://frogs.cafe/challenges/abc123",     // challenge URI (you mint this)
  {
    type: "BikChallenge",
    boardSize: 19,
    timeControl: { system: "byoyomi", mainTime: 600, periods: 5, periodTime: 30 },
    colorAssignment: "random",
    expiresAt: "2024-03-10T15:30:00Z",
  },
  "Looking for a game! 19x19, 10min + 5x30s byo-yomi",
);

// Broadcast to your followers / federate as needed
```

### Accepting a challenge

```ts
import { buildAcceptChallenge, postActivity } from "bik-js";

const activity = buildAcceptChallenge(
  "https://server-b.example/users/bob",       // your actor URI
  "https://server-a.example/challenges/abc123", // URI of the challenge
);

await postActivity("https://server-a.example/inbox", activity);
// Server A responds with a CreateGame activity containing the WebSocket URL
```

### Withdrawing a challenge

When a challenge is accepted or cancelled, the host sends an `Undo` so other servers can remove it from open challenge lists.

```ts
import { buildUndoChallenge, postActivity } from "bik-js";

const activity = buildUndoChallenge(
  "https://server-a.example/users/alice",
  "https://server-a.example/challenges/abc123",
);

// Broadcast to followers
```

### Announcing a game result

```ts
import { buildCreateGameResult, postActivity } from "bik-js";

const activity = buildCreateGameResult(
  "https://server-a.example/users/alice",
  "https://server-a.example/games/xyz789/result",
  "https://server-a.example/games/xyz789",
  {
    type: "BikGameResult",
    winner: "https://server-a.example/users/alice",
    result: "B+R",
    sgf: "https://server-a.example/games/xyz789.sgf",
  },
  "@alice (B) defeated @bob@server-b.example (W) by resignation.",
);
```

Results are standard AP `Note` objects and will appear as posts on Mastodon and other ActivityPub clients.

## Types

All types are exported from the package root:

```ts
import type {
  // Common
  Position, Color, Clock, GamePhase, GameResult, TimeControl,

  // WebSocket messages
  ClientMessage, ServerMessage,
  MovePlayed, MoveRejectedReason, GameStateData,

  // ActivityPub activities
  BikChallenge, BikGame, BikGameResult,
  CreateChallengeActivity, AcceptChallengeActivity,
  UndoChallengeActivity, CreateGameActivity, CreateGameResultActivity,
} from "bik-js";
```

### Coordinates

Positions are `[col, row]` zero-indexed integer arrays. `[0, 0]` is the top-left corner. Pass is `null`.

```ts
const d4: Position = [3, 3];
const pass: Position = null;
```

### Clock

Clocks are discriminated by `system`:

```ts
if (clock.system === "byoyomi") {
  console.log(clock.black.periods, "periods remaining");
} else if (clock.system === "fischer") {
  console.log("increment:", clock.increment);
}
```
