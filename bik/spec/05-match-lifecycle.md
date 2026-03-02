# BIK-05: Match Lifecycle

**Status**: Draft  
**Dependencies**: [BIK-03](03-identity.md), [BIK-04](04-seek.md)  
**Normative references**: [Matrix Specification](https://spec.matrix.org/), [Matrix Room Version 10](https://spec.matrix.org/v1.9/rooms/v10/)

---

## 1. Overview

Once a match proposal is accepted, the two servers must:

1. Create a shared match record with a canonical identifier
2. Synchronize game state as moves are played
3. Handle the end of the game and record the result

BIK uses a **Matrix-inspired event DAG** for match state. Each move is a signed event chained to the previous state. This provides a verifiable, tamper-evident game history without requiring synchronous coordination.

Implementations are NOT required to run a full Matrix homeserver. They must implement the relevant subset of the Matrix event format and signing scheme.

## 2. Match Creation

Upon receiving an `Accept` for a `MatchProposal`, the **host server** (as specified in the proposal) creates the match and sends a `MatchCreated` activity to the guest server.

```json
{
  "@context": [
    "https://www.w3.org/ns/activitystreams",
    "https://bik.example/ns/v1"
  ],
  "type": "MatchCreated",
  "id": "https://chess.example/matches/match001/events/0",
  "actor": "https://chess.example",
  "published": "2026-02-28T12:00:06Z",
  "match": {
    "id": "https://chess.example/matches/match001",
    "host": "https://chess.example",
    "guest": "https://go.example",
    "game": "https://chess.example/games/chess",
    "variant": "standard",
    "timeControl": {
      "type": "fischer",
      "initial": 600,
      "increment": 0
    },
    "players": {
      "white": "https://chess.example/players/alice",
      "black": "https://go.example/players/bob"
    },
    "startedAt": "2026-02-28T12:00:06Z"
  },
  "stateHash": "sha256:e3b0c44298fc1c149afb...",
  "serverSignature": "..."
}
```

The `stateHash` is the hash of the initial game state (typically an empty board or defined starting position). The host server signs this event. The guest server MUST verify the signature and store this as the genesis event of the match DAG.

## 3. The Match Event DAG

Each move in the game is a **Match Event** — a signed record that includes the move, the previous state hash, and the resulting state hash.

```json
{
  "type": "MoveEvent",
  "id": "https://chess.example/matches/match001/events/5",
  "matchId": "https://chess.example/matches/match001",
  "turn": 5,
  "player": "https://chess.example/players/alice",
  "published": "2026-02-28T12:01:30Z",
  "prevEventId": "https://chess.example/matches/match001/events/4",
  "prevStateHash": "sha256:aabbcc...",
  "move": {
    "notation": "e2e4"
  },
  "stateHash": "sha256:ddeeff...",
  "playerSignature": "...",
  "serverSignature": "..."
}
```

| Field | Description |
|-------|-------------|
| `turn` | Sequential turn number, 1-indexed |
| `prevEventId` | ID of the previous event in the chain |
| `prevStateHash` | Hash of game state before this move |
| `move` | Game-specific move payload (opaque to BIK layer) |
| `stateHash` | Hash of game state after this move |
| `playerSignature` | Move signed by the player's private key |
| `serverSignature` | Move signed by the home server's key |

### 3.1 Dual Signing

Each move carries two signatures: the player's and the server's. This ensures:

- The **player's signature** proves the move was authorized by the player (non-repudiation)
- The **server's signature** proves the move was processed and validated by the home server

The guest server MUST verify both signatures before accepting a move event.

### 3.2 Move Delivery

When a player makes a move, their home server:

1. Validates the move against game rules
2. Signs the event with both the player key and server key
3. POSTs the `MoveEvent` to the opponent's home server inbox

The guest server:
1. Verifies both signatures
2. Validates the move against its own copy of the game rules
3. Returns `200 OK` if accepted, `409 Conflict` if the move is invalid

If the guest server rejects a move, both servers flag the match for dispute resolution (see Section 6).

### 3.3 Move Format

The `move` object is **opaque to BIK**. Its format is defined by the game descriptor (see [BIK-06](06-game-descriptor.md)). BIK only requires that it is a valid JSON object.

## 4. Clock Handling

Time controls are enforced locally by each player's home server. The host server's clock is authoritative for determining timeouts.

When a player runs out of time, their home server sends a `Timeout` event to the opponent's server:

```json
{
  "type": "Timeout",
  "matchId": "https://chess.example/matches/match001",
  "player": "https://go.example/players/bob",
  "serverSignature": "..."
}
```

## 5. Game End

Games end via: checkmate/game-over condition, resignation, draw agreement, timeout, or abandonment.

The host server sends a `MatchResult` activity to the guest server and to both players' inboxes:

```json
{
  "type": "MatchResult",
  "id": "https://chess.example/matches/match001/result",
  "matchId": "https://chess.example/matches/match001",
  "published": "2026-02-28T12:30:00Z",
  "result": "white_wins",
  "termination": "checkmate",
  "finalStateHash": "sha256:ffee11...",
  "moves": 42,
  "serverSignature": "..."
}
```

Result values: `white_wins`, `black_wins`, `draw`, `abandoned`.
Termination values: `checkmate`, `resignation`, `timeout`, `draw_agreement`, `stalemate`, `repetition`, `insufficient_material`, `abandonment`.

Both servers update their local rating systems after receiving the `MatchResult`.

## 6. Dispute Resolution

If servers disagree on game state (e.g., a move is accepted by one server but rejected by the other), the match enters **disputed** status.

Dispute resolution in v1 uses the **replay approach**: either server may request a full event log from the other. Both replay all events from the genesis event to reconstruct state independently. If they agree, the dispute is resolved. If they still disagree, the match is voided and both players are restored to their pre-match ratings.

More sophisticated arbitration (third-party arbiter servers) is reserved for a future version.

## 7. Match Recovery

If a server goes offline mid-match, the match is paused. Upon reconnection, the reconnecting server requests the event log from the peer to resync state. Matches have a maximum idle time of 7 days before being declared abandoned (configurable by server operators, minimum 24 hours).
