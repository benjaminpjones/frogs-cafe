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
