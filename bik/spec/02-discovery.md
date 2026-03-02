# BIK-02: Discovery

**Status**: Draft  
**Dependencies**: [BIK-01](01-overview.md)  
**Normative references**: [RFC 7033 (WebFinger)](https://www.rfc-editor.org/rfc/rfc7033), [ActivityPub](https://www.w3.org/TR/activitypub/), [ActivityStreams 2.0](https://www.w3.org/TR/activitystreams-core/)

---

## 1. Overview

Discovery covers two related problems:

- **Player discovery**: Given `alice@chess.example`, how does another server find Alice's AP Actor document and federation endpoints?
- **Server discovery**: Given `chess.example`, how does another server learn what games it supports, what its signing key is, and what BIK capabilities it offers?

Both are solved by profiling existing standards: WebFinger for the initial lookup, ActivityPub Actor documents for the details.

## 2. Player Discovery via WebFinger

A BIK server MUST implement WebFinger at `/.well-known/webfinger` per RFC 7033.

A WebFinger query for a player `alice@chess.example`:

```
GET /.well-known/webfinger?resource=acct:alice@chess.example
Host: chess.example
Accept: application/jrd+json
```

The response MUST include a link with `rel=self` pointing to the player's AP Actor document:

```json
{
  "subject": "acct:alice@chess.example",
  "links": [
    {
      "rel": "self",
      "type": "application/activity+json",
      "href": "https://chess.example/players/alice"
    }
  ]
}
```

## 3. Player Actor Document

The AP Actor document at the linked URL MUST conform to ActivityPub and MUST additionally include the `bik` extension object:

```json
{
  "@context": [
    "https://www.w3.org/ns/activitystreams",
    "https://bik.example/ns/v1"
  ],
  "type": "Person",
  "id": "https://chess.example/players/alice",
  "name": "Alice",
  "preferredUsername": "alice",
  "inbox": "https://chess.example/players/alice/inbox",
  "outbox": "https://chess.example/players/alice/outbox",
  "publicKey": {
    "id": "https://chess.example/players/alice#main-key",
    "owner": "https://chess.example/players/alice",
    "publicKeyPem": "-----BEGIN PUBLIC KEY-----\n..."
  },
  "bik": {
    "server": "https://chess.example/bik"
  }
}
```

The `bik.server` field points to the server capability document (see Section 5).

## 4. Server Discovery via Well-Known

A BIK server MUST serve its capability document at:

```
/.well-known/bik.json
```

This allows server-level discovery without needing a specific player as an entry point.

```
GET /.well-known/bik.json
Host: chess.example
```

## 5. Server Capability Document

The capability document describes what the server supports and where its endpoints are.

```json
{
  "bik": "1.0",
  "server": "https://chess.example",
  "name": "Example Chess Server",
  "publicKey": {
    "id": "https://chess.example#server-key",
    "publicKeyPem": "-----BEGIN PUBLIC KEY-----\n..."
  },
  "conformanceLevel": 2,
  "endpoints": {
    "seek": "https://chess.example/bik/seek",
    "matchProposal": "https://chess.example/bik/match-proposal",
    "inbox": "https://chess.example/bik/inbox"
  },
  "games": [
    "https://chess.example/games/chess",
    "https://chess.example/games/chess960"
  ],
  "peers": [
    "https://go.example",
    "https://othergames.example"
  ]
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `bik` | Yes | BIK version string |
| `server` | Yes | Canonical server URL |
| `name` | No | Human-readable server name |
| `publicKey` | Yes | Server signing key (see [BIK-03](03-identity.md)) |
| `conformanceLevel` | Yes | 0–3 per BIK-01 Section 4 |
| `endpoints` | Yes | Map of BIK endpoint URLs |
| `games` | Yes | URLs of supported game descriptor documents |
| `peers` | No | Known peer servers (informational) |

## 6. Game Descriptor Links

Each URL in the `games` array resolves to a game descriptor document as defined in [BIK-06](06-game-descriptor.md). Remote servers may fetch these to determine match compatibility before proposing a match.

## 7. Caching

Server capability documents SHOULD include standard HTTP cache headers. Clients SHOULD respect them. A minimum TTL of 1 hour is RECOMMENDED to reduce discovery overhead.

Player Actor documents SHOULD similarly be cacheable. Servers MUST re-fetch them when a request signed by that player fails signature verification (the key may have rotated).
