# BIK: Baduk InterKonnect Protocol

**Version:** 0.1 (Draft)

BIK enables Go (Baduk/Weiqi) games between players on different servers. It builds on ActivityPub for federation and identity, with WebSocket connections for low-latency gameplay.

## Core Concepts

### Identity

Players are identified as `@username@domain`, which is the standard fediverse format:
- `@alice@server-a.example`
- `@bob@server-b.example`

Servers are identified by domain and expose a WebFinger endpoint for actor discovery.

### Server Roles

- **Host**: The server where the game is played. Stores game state, validates moves, manages clocks.
- **Guest**: The visiting player's home server. Vouches for player identity.

Once a game begins, all gameplay flows through the host. The guest server may observe but does not relay moves.

## Challenge Flow

Challenges use ActivityPub activities for federation.

All BIK activities MUST include `to` and `cc` for correct AP delivery routing:
- Public broadcasts (challenges, game announcements, results): `to: [Public]`, `cc` includes the actor's followers collection and any directly addressed participants
- Directed activities (Accept, direct challenges): `to` contains only the addressed actor(s), no `cc` required
- The followers collection URL is discovered from the actor document (`followers` field)

### 1. Create Challenge

A player creates an open challenge on their server.

```json
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "type": "Create",
  "actor": "https://server-a.example/users/alice",
  "to": ["https://www.w3.org/ns/activitystreams#Public"],
  "cc": ["https://server-a.example/users/alice/followers"],
  "object": {
    "type": "Note",
    "id": "https://server-a.example/challenges/abc123",
    "attributedTo": "https://server-a.example/users/alice",
    "content": "Looking for a game! 19x19, 10min + 5x30s byo-yomi",
    "attachment": {
      "type": "BikChallenge",
      "boardSize": 19,
      "timeControl": {
        "system": "byoyomi",
        "mainTime": 600,
        "periods": 5,
        "periodTime": 30
      },
      "colorAssignment": "random",
      "komi": 6.5,
      "expiresAt": "2024-03-10T15:30:00Z"
    }
  }
}
```

The `Note` is renderable by standard ActivityPub clients (Mastodon, etc). The `attachment` contains machine-readable game parameters.

Challenges MAY be directed at a specific player instead of broadcast publicly. A direct challenge addresses the target actor in `to` and omits `Public`:

```json
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "type": "Create",
  "actor": "https://server-a.example/users/alice",
  "to": ["https://server-b.example/users/bob"],
  "object": {
    "type": "Note",
    "id": "https://server-a.example/challenges/abc123",
    "attributedTo": "https://server-a.example/users/alice",
    "content": "Hey @bob@server-b.example, want a game? 19x19, 10min + 5x30s byo-yomi",
    "attachment": {
      "type": "BikChallenge",
      "boardSize": 19,
      "timeControl": { "system": "byoyomi", "mainTime": 600, "periods": 5, "periodTime": 30 },
      "colorAssignment": "random",
      "expiresAt": "2024-03-10T15:30:00Z"
    }
  }
}
```

Servers receiving a direct challenge SHOULD only allow the addressed player to accept it. Clients can determine challenge visibility by inspecting `to`: if it contains `https://www.w3.org/ns/activitystreams#Public` the challenge is open; otherwise it is direct.

Challenges expire at `expiresAt`. Servers SHOULD reject `Accept` activities for expired challenges.

`komi` is the point compensation given to white. If omitted, it defaults to `0`. Negative values are valid.

`colorAssignment` determines how colors are assigned when the challenge is accepted:
- `"black"` — the challenger (the player who created the challenge) plays black
- `"white"` — the challenger plays white
- `"random"` — the host assigns colors randomly on accept

<!-- TODO: rating-based color assignment (stronger player takes white) — requires rating exchange between servers -->

### 2. Accept Challenge

A remote player accepts:

```json
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "type": "Accept",
  "actor": "https://server-b.example/users/bob",
  "to": ["https://server-a.example/users/alice"],
  "object": "https://server-a.example/challenges/abc123"
}
```

The challenge creator's server (server-a.example) becomes the **host**.

### 3. Challenge Withdrawn

When a challenge is accepted or the creator cancels it, the host broadcasts an `Undo` so followers know it's no longer available:

```json
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "type": "Undo",
  "actor": "https://server-a.example/users/alice",
  "to": ["https://www.w3.org/ns/activitystreams#Public"],
  "cc": ["https://server-a.example/users/alice/followers"],
  "object": "https://server-a.example/challenges/abc123"
}
```

Clients receiving this should remove the challenge from any open challenge lists.

### 4. Game Created

Host notifies both parties and their followers:

```json
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "type": "Create",
  "actor": "https://server-a.example/users/alice",
  "to": ["https://www.w3.org/ns/activitystreams#Public"],
  "cc": ["https://server-a.example/users/alice/followers", "https://server-b.example/users/bob"],
  "object": {
    "type": "Note",
    "id": "https://server-a.example/games/xyz789",
    "content": "@alice vs @bob@server-b.example — Game started! Watch at https://server-a.example/games/xyz789",
    "inReplyTo": "https://server-a.example/challenges/abc123",
    "attachment": {
      "type": "BikGame",
      "id": "https://server-a.example/games/xyz789",
      "black": "https://server-a.example/users/alice",
      "white": "https://server-b.example/users/bob",
      "komi": 6.5,
      "websocket": "wss://server-a.example/ws/games/xyz789"
    }
  }
}
```

`BikGame.id` is the canonical game URI — both its AP identifier and the human-readable page. The `websocket` is for live clients.

## Gameplay

### Authentication

The guest player needs to authenticate to the host's WebSocket. Flow:

1. Guest player requests a signed token from **their own server**:
   ```
   POST /api/v1/bik/token?gameURI=https://server-a.example/games/xyz789
   Authorization: <normal session credential>
   ```
   Response:
   ```json
   { "token": "<signed-jwt>" }
   ```
   The token payload is:
   ```json
   {
     "kid": "a3f2c1b4e5d6f7a8",
     "player": "https://server-b.example/users/bob",
     "game": "https://server-a.example/games/xyz789",
     "exp": 1710000000
   }
   ```
   Signed with the guest server's private key. Tokens MUST have a short expiry (recommended: 5 minutes).

2. Guest player connects to host WebSocket with token:
   ```
   wss://server-a.example/ws/games/xyz789?token=<signed-jwt>
   ```

3. Host verifies token by:
   - Decoding the payload (base64, no key required) to read `player` and `kid`
   - Extracting the guest server's domain from the `player` actor URI
   - Fetching the guest server's JWK set (via `/.well-known/bik/keys`, cached) and selecting the key matching `kid` — re-fetching if `kid` is unknown
   - Verifying the signature against that public key
   - Confirming expiry has not passed
   - Confirming `player` matches a participant in the named game

Local players authenticate with their usual session token.

### Coordinates

Board positions are `[col, row]` zero-indexed integer arrays. `[0, 0]` is the top-left corner. This works for any board size.

Examples: `[0, 0]` (top-left), `[18, 18]` (bottom-right on 19x19), `[52, 52]` (bottom-right on 53x53).

Pass is represented as `null`.

### WebSocket Messages

All messages share a common envelope:

```json
{"type": "<message-type>", "data": { ... }}
```

#### Client → Host

| Type | Description |
|------|-------------|
| `move` | Place a stone or pass |
| `resign` | Forfeit the game |
| `mark_dead` | Mark stones as dead during scoring |
| `score_accept` | Accept the current score |
| `score_reject` | Reject score (resume play) |
| `keepalive` | Signal client liveness |

**`move`**
```json
{"type": "move", "data": {"pos": [3, 3]}}
{"type": "move", "data": {"pos": null}}
```

**`resign`**
```json
{"type": "resign", "data": {}}
```

**`mark_dead`** *(scoring phase only)*
```json
{"type": "mark_dead", "data": {"positions": [[3, 3], [4, 3], [4, 4]]}}
```

**`score_accept`** / **`score_reject`**
```json
{"type": "score_accept", "data": {}}
{"type": "score_reject", "data": {}}
```

**`keepalive`**
```json
{"type": "keepalive", "data": {"ts": 1710000000}}
```

Clients MUST send `keepalive` messages at a regular interval during active games. The `ts` field is the client's Unix timestamp in seconds (integer). The recommended interval is 3 seconds.

Hosts use keepalives to detect client liveness. If a host receives no messages (moves or keepalives) from a player for an implementation-defined timeout (recommended: 15 seconds), it MAY flag the player as disconnected and start a disconnection clock per its own policy.

Hosts MUST NOT require keepalives — a client that sends only game actions is still considered connected. Keepalives supplement, rather than replace, the WebSocket protocol's built-in ping/pong frames.

#### Host → All Clients

| Type | Description |
|------|-------------|
| `game_state` | Full game state (sent on connect/reconnect) |
| `move_played` | A move was accepted and played |
| `move_rejected` | A move was rejected |
| `clock` | Current clock state |
| `phase_change` | Game phase changed (e.g. active → scoring) |
| `dead_stones` | Updated dead stone markings |
| `game_over` | Game has ended |

**`game_state`** — sent immediately on connection so clients can sync
```json
{
  "type": "game_state",
  "data": {
    "moves": [[3, 3], [15, 2], null],
    "phase": "active",
    "clock": { ... },
    "nextToPlay": "black"
  }
}
```

**`move_played`**
```json
{
  "type": "move_played",
  "data": {
    "pos": [3, 3],
    "moveNumber": 47,
    "color": "black",
    "captures": [[3, 4]],
    "nextToPlay": "white",
    "clock": {"system": "byoyomi", "black": {"main": 542, "periods": 4, "periodTime": 30}, "white": {"main": 600, "periods": 5, "periodTime": 30}, "activeColor": "white"}
  }
}
```

**`move_rejected`**
```json
{
  "type": "move_rejected",
  "data": {"reason": "ko"}
}
```
Possible reasons: `ko`, `occupied`, `suicide`, `not_your_turn`, `game_not_active`.

**`phase_change`**
```json
{"type": "phase_change", "data": {"phase": "scoring"}}
```
Phases: `active`, `scoring`, `finished`.

**`game_over`**
```json
{
  "type": "game_over",
  "data": {
    "result": "B+R",
    "winner": "black",
    "scoreBlack": null,
    "scoreWhite": null
  }
}
```
`result` follows SGF convention: `B+R` (black wins by resignation), `W+3.5` (white wins by 3.5 points), `B+T` (black wins on time).

### Clock Sync

The host is authoritative for time. A `clock` message is broadcast after every move and periodically during play.

**`clock`** format varies by time system:

```json
{"type": "clock", "data": {"system": "byoyomi", "black": {"main": 542, "periods": 4, "periodTime": 30}, "white": {"main": 600, "periods": 5, "periodTime": 30}, "activeColor": "black"}}
```
```json
{"type": "clock", "data": {"system": "fischer", "black": 312, "white": 480, "increment": 10, "activeColor": "white"}}
```
```json
{"type": "clock", "data": {"system": "absolute", "black": 215, "white": 430, "activeColor": "black"}}
```

The host is the source of truth for time. If no `clock` message arrives within 5 seconds during active play, clients should request `game_state` to resync.

### Reconnection

Clients may disconnect and reconnect at any time. On reconnect, the host sends a `game_state` message immediately, giving the full move list and current clock. Clients should treat `game_state` as authoritative and discard any locally cached state.

## Game Completion

When the game ends, host publishes result via ActivityPub:

```json
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "type": "Create",
  "actor": "https://server-a.example/users/alice",
  "to": ["https://www.w3.org/ns/activitystreams#Public"],
  "cc": ["https://server-a.example/users/alice/followers", "https://server-b.example/users/bob"],
  "object": {
    "type": "Note",
    "id": "https://server-a.example/games/xyz789/result",
    "content": "Game finished! @alice (B) defeated @bob@server-b.example (W) by resignation.",
    "inReplyTo": "https://server-a.example/games/xyz789",
    "attachment": {
      "type": "BikGameResult",
      "winner": "https://server-a.example/users/alice",
      "result": "B+R",
      "sgf": "https://server-a.example/games/xyz789.sgf"
    }
  }
}
```

This appears as a post on ActivityPub, linkable and viewable by anyone.

## Discovery

### WebFinger

Servers must implement WebFinger for actor discovery:

```
GET /.well-known/webfinger?resource=acct:alice@server-a.example
```

Returns:
```json
{
  "subject": "acct:alice@server-a.example",
  "links": [
    {
      "rel": "self",
      "type": "application/activity+json",
      "href": "https://server-a.example/users/alice"
    }
  ]
}
```

### Actor Document

The actor URI returned by WebFinger MUST dereference to an AP actor document:

```
GET https://server-a.example/users/alice
Accept: application/activity+json
```

Returns:
```json
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "type": "Person",
  "id": "https://server-a.example/users/alice",
  "preferredUsername": "alice",
  "inbox": "https://server-a.example/bik/inbox",
  "followers": "https://server-a.example/users/alice/followers"
}
```

The `inbox` field is where AP activities (e.g. `Accept`) must be delivered. The `followers` collection URL is used by AP servers to fan out public activities to followers.

### Server Keys

For token verification, servers expose public keys at a well-known endpoint:

```
GET /.well-known/bik/keys
```

Returns a JWK set for signature verification. Each key MUST include a `kid` (key ID) field:

```json
{
  "keys": [
    { "kty": "OKP", "crv": "Ed25519", "kid": "key-1", "x": "<base64url>" }
  ]
}
```

Signed tokens MUST include the `kid` of the signing key. When a verifying server encounters an unknown `kid`, it MUST re-fetch `/.well-known/bik/keys` rather than rejecting the token immediately — the signing server may have rotated keys. Servers rotating keys SHOULD retain the old key in the JWK set for at least 10 minutes to allow in-flight tokens to drain.

## Security Considerations

- All endpoints MUST use HTTPS
- Tokens MUST have short expiry (recommend: 5 minutes)
- Servers SHOULD rate-limit incoming federation requests
- Hosts MUST validate that move submitters are game participants
- Servers MUST implement HTTP Signatures for server-to-server requests (per ActivityPub spec) — without this, a malicious actor could spoof AP activities (e.g. fake Accept) to any BIK inbox

## Future Extensions

- **Spectator federation**: Allow remote servers to subscribe to game updates
- **Rating exchange**: Share rating information between trusted servers
- **Tournament support**: Coordinate multi-game events across servers
- **REST snapshot endpoint**: `GET /games/{uri}` returning the same `game_state`
  shape as the WebSocket message. The WebSocket would still own live updates,
  but a REST snapshot would support cacheable embeds, archive scrapers, link
  previews, and other consumers that don't need a persistent connection.
