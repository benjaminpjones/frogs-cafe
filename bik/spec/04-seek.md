# BIK-04: Seek and Federated Automatch

**Status**: Draft  
**Dependencies**: [BIK-02](02-discovery.md), [BIK-03](03-identity.md), [BIK-06](06-game-descriptor.md)

---

## 1. Overview

Federated automatch is the most valuable piece of BIK federation. A player seeking a game should be matched against the best available opponent across all federated servers, not just their home server. This eliminates the empty-lobby problem that afflicts small servers.

The seek protocol is the most novel part of BIK — no existing standard covers this ground.

### 1.1 Design Constraints

- Seeks are time-sensitive. Players abandon queues quickly; the pool must reflect current availability.
- Federation is eventually consistent. Race conditions (two servers simultaneously matching the same player) are acceptable if they resolve gracefully.
- No central coordinator is required. Optional relay servers improve match quality but are not mandatory.
- Player privacy must be respected. Seeks are opt-in and scopeable.

## 2. The Seek Object

A `Seek` is a BIK Activity representing a player's intent to find a game.

```json
{
  "@context": [
    "https://www.w3.org/ns/activitystreams",
    "https://bik.example/ns/v1"
  ],
  "type": "Seek",
  "id": "https://chess.example/seeks/abc123",
  "actor": "https://chess.example/players/alice",
  "published": "2026-02-28T12:00:00Z",
  "expiresAt": "2026-02-28T12:01:00Z",
  "game": "https://chess.example/games/chess",
  "variant": "standard",
  "timeControl": {
    "type": "fischer",
    "initial": 600,
    "increment": 0
  },
  "ratingRange": {
    "min": 1400,
    "max": 1700
  },
  "color": "any",
  "scope": "federation"
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Canonical URL of this seek |
| `actor` | Yes | AP Actor URL of the seeking player |
| `published` | Yes | ISO 8601 timestamp when seek was created |
| `expiresAt` | Yes | ISO 8601 timestamp when seek expires. MUST be ≤ 120 seconds from `published` |
| `game` | Yes | URL of the game descriptor |
| `variant` | Yes | Variant identifier from the game descriptor |
| `timeControl` | Yes | Time control specification (see [BIK-06](06-game-descriptor.md)) |
| `ratingRange` | No | Acceptable opponent rating range on the seeking player's home server scale |
| `color` | No | Preferred color/side (`"white"`, `"black"`, `"any"`) |
| `scope` | Yes | `"local"` (home server only), `"peers"` (direct peers), `"federation"` (full federation) |

### 2.1 Seek Expiry

Seeks MUST expire. The maximum TTL is 120 seconds. Short TTLs ensure the pool reflects actual availability. Servers MUST NOT match against expired seeks.

When a player cancels a seek, the server SHOULD broadcast a `CancelSeek` activity (same `id` as the original seek) to peers that received the original.

## 3. Seek Distribution

### 3.1 Peer-to-Peer Distribution

When a player creates a seek with `scope: "peers"` or `scope: "federation"`, the home server MUST POST the `Seek` activity to the BIK inbox of each known peer server.

Peers that receive a seek MAY forward it to their own peers (one additional hop) if the original seek's `scope` is `"federation"`. Seeks MUST NOT be forwarded more than one hop from the originating server, to prevent unbounded propagation.

```
chess.example → go.example (direct peer)
chess.example → othergames.example (direct peer)
go.example → shogi.example (one hop, if scope=federation)
```

### 3.2 Relay Distribution

A server with `conformanceLevel: 3` (Relay) aggregates seeks from subscribing peers and handles cross-server matching on their behalf. Participation is voluntary.

A server subscribes to a relay by POSTing a `Subscribe` activity to the relay's inbox:

```json
{
  "type": "Subscribe",
  "actor": "https://chess.example",
  "object": "https://relay.example/bik/seek-pool"
}
```

Subscribed servers receive incoming seeks from the relay and MUST forward their own seeks to the relay. The relay handles matching across all subscribers.

Using a relay does not prevent direct peer matching in parallel. Servers MAY participate in multiple relays.

## 4. Match Proposal

When a server determines that two seeks are compatible, it initiates a match by sending a `MatchProposal` to the other player's home server.

```json
{
  "@context": [
    "https://www.w3.org/ns/activitystreams",
    "https://bik.example/ns/v1"
  ],
  "type": "MatchProposal",
  "id": "https://chess.example/proposals/xyz789",
  "actor": "https://chess.example",
  "published": "2026-02-28T12:00:05Z",
  "expiresAt": "2026-02-28T12:00:10Z",
  "localSeek": "https://chess.example/seeks/abc123",
  "remoteSeek": "https://go.example/seeks/def456",
  "proposedMatch": {
    "host": "https://chess.example",
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
    }
  }
}
```

The receiving server MUST respond within the `expiresAt` window (RECOMMENDED: 5 seconds). Proposals that are not answered in time are treated as rejected.

### 4.1 Proposal Response

**Accept:**
```json
{
  "type": "Accept",
  "object": "https://chess.example/proposals/xyz789",
  "actor": "https://go.example"
}
```

**Reject:**
```json
{
  "type": "Reject",
  "object": "https://chess.example/proposals/xyz789",
  "actor": "https://go.example",
  "reason": "player_unavailable"
}
```

Rejection reasons: `player_unavailable` (already matched or cancelled), `seek_expired`, `incompatible_parameters`, `server_error`.

### 4.2 Race Condition Handling

Two servers may simultaneously propose a match for the same seek. This is acceptable. The receiving server rejects the second proposal with `player_unavailable`. The proposing server re-enters the matching loop.

Because turn-based games tolerate this gracefully (the failed proposal costs only milliseconds), no coordination mechanism is required to prevent races.

## 5. Post-Match-Acceptance

Once a `MatchProposal` is accepted, both servers proceed to match creation per [BIK-05](05-match-lifecycle.md). Both servers MUST immediately mark the relevant seeks as fulfilled and SHOULD broadcast `CancelSeek` for any other outstanding seeks from the same players.

## 6. Rating Compatibility

When comparing seeks from different servers, rating ranges are expressed in the **seeking player's home server scale**. The receiving server uses its rank map for the originating server (see [BIK-07](07-rank-maps.md)) to translate the range into local terms before evaluating compatibility.

If no rank map exists for a server pair, servers MAY use percentile distribution matching as a fallback, or MAY decline to match cross-server until a map is established. This behavior SHOULD be configurable by server operators.
