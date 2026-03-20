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

### 1. Create Challenge

A player creates an open challenge on their server.

```json
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "type": "Create",
  "actor": "https://server-a.example/users/alice",
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
      "colorPreference": "any",
      "expiresAt": "2024-03-10T15:30:00Z"
    }
  }
}
```

The `Note` is renderable by standard ActivityPub clients (Mastodon, etc). The `attachment` contains machine-readable game parameters.

Challenges expire at `expiresAt`. Servers SHOULD reject `Accept` activities for expired challenges.

### 2. Accept Challenge

A remote player accepts:

```json
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "type": "Accept",
  "actor": "https://server-b.example/users/bob",
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
  "object": {
    "type": "Note",
    "id": "https://server-a.example/games/xyz789",
    "content": "@alice vs @bob@server-b.example — Game started! Watch at https://server-a.example/games/xyz789",
    "inReplyTo": "https://server-a.example/challenges/abc123",
    "attachment": {
      "type": "BikGame",
      "id": "xyz789",
      "black": "https://server-a.example/users/alice",
      "white": "https://server-b.example/users/bob",
      "url": "https://server-a.example/games/xyz789",
      "websocket": "wss://server-a.example/ws/games/xyz789"
    }
  }
}
```

The `url` is a human-readable page to watch the game. The `websocket` is for live clients.

## Gameplay

### Authentication

The guest player needs to authenticate to the host's WebSocket. Flow:

1. Guest server issues a **signed token** for its player:
   ```json
   {
     "player": "@bob@server-b.example",
     "game": "https://server-a.example/games/xyz789",
     "exp": 1710000000
   }
   ```
   Signed with the guest server's private key.

2. Guest player connects to host WebSocket with token:
   ```
   wss://server-a.example/ws/games/xyz789?token=<base64-encoded-token>
   ```

3. Host verifies token by:
   - Fetching guest server's public key (via `/.well-known/bik/keys`, cached)
   - Validating signature and expiry
   - Confirming player matches game participant

Local players authenticate with their usual session token.

### WebSocket Messages

Once connected, standard game messages flow:

```json
{"type": "move", "data": {"x": 3, "y": 4, "moveNumber": 1}}
{"type": "pass", "data": {"moveNumber": 42}}
{"type": "resign", "data": {}}
{"type": "clock", "data": {"black": 542, "white": 600}}
```

All clients (local and remote) receive the same broadcasts from the host.

### Clock Sync

The host is authoritative for time. Clients receive periodic `clock` messages. On move receipt, host broadcasts updated clock state.

For blitz games, clients should display server time, not local calculations.

## Game Completion

When the game ends, host publishes result via ActivityPub:

```json
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "type": "Create",
  "actor": "https://server-a.example/users/alice",
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

### Server Keys

For token verification, servers expose public keys at a well-known endpoint:

```
GET /.well-known/bik/keys
```

Returns JWK set for signature verification.

## Security Considerations

- All endpoints MUST use HTTPS
- Tokens MUST have short expiry (recommend: 5 minutes)
- Servers SHOULD rate-limit incoming federation requests
- Hosts MUST validate that move submitters are game participants
- Servers SHOULD implement HTTP Signatures for server-to-server requests (per ActivityPub spec)

## Future Extensions

- **Spectator federation**: Allow remote servers to subscribe to game updates
- **Rating exchange**: Share rating information between trusted servers
- **Tournament support**: Coordinate multi-game events across servers
