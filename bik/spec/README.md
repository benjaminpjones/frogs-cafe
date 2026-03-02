# BIK Protocol (Board InterKonnect)

A federation standard for turn-based game servers, enabling players on independent servers to discover each other, compete, and maintain portable identities and rankings — without requiring a central authority.

## Goals

- **Open**: Built on existing open standards (ActivityPub, Matrix) wherever possible. Novel specification work is minimized to what is truly unique to this domain.
- **Decentralized**: No single server or authority is required. Servers cooperate as peers.
- **Interoperable**: A player on any conforming server can play against a player on any other conforming server.
- **Federated matchmaking**: The automatch pool spans servers, eliminating the empty-lobby problem that plagues small game servers.

## Architecture Overview

The standard is layered:

```
┌─────────────────────────────────────┐
│         Application Layer           │
│  (game rules, UI, ratings, lobbies) │
├─────────────────────────────────────┤
│         Game State Layer            │
│  Matrix event DAG for match state   │
├─────────────────────────────────────┤
│         Social/Federation Layer     │
│  ActivityPub for identity, discovery│
│  invites, seek advertisement        │
├─────────────────────────────────────┤
│         Transport Layer             │
│  HTTPS + HTTP Signatures            │
└─────────────────────────────────────┘
```

## Specification Documents

| Document | Status | Description |
|----------|--------|-------------|
| [01-overview.md](spec/01-overview.md) | Draft | Concepts, terminology, and design principles |
| [02-discovery.md](spec/02-discovery.md) | Draft | Server and player discovery via WebFinger and ActivityPub |
| [03-identity.md](spec/03-identity.md) | Draft | Player identity, server keys, and HTTP Signatures |
| [04-seek.md](spec/04-seek.md) | Draft | Federated automatch: seek advertisement, relay topology, match proposal |
| [05-match-lifecycle.md](spec/05-match-lifecycle.md) | Draft | Match creation, move delivery, result recording |
| [06-game-descriptor.md](spec/06-game-descriptor.md) | Draft | Format for advertising supported games, variants, and time controls |
| [07-rank-maps.md](spec/07-rank-maps.md) | Draft | Cross-server rank translation tables |

## Relationship to Existing Standards

This standard **profiles and extends** existing standards rather than replacing them:

- **[ActivityPub](https://www.w3.org/TR/activitypub/)** (W3C Recommendation): Used for actor identity, server-to-server message delivery, and the social layer (invites, results, clubs). We extend the AP vocabulary with game-specific Activity and Object types.
- **[WebFinger](https://www.rfc-editor.org/rfc/rfc7033)** (RFC 7033): Used for player and server discovery.
- **[HTTP Signatures](https://datatracker.ietf.org/doc/html/draft-cavage-http-signatures)**: Used for authenticating server-to-server requests, consistent with AP's federation model.
- **[Matrix](https://spec.matrix.org/)** (Matrix.org Foundation): The event DAG and room model are used for match state synchronization. The Matrix signing and hashing scheme ensures verifiable move history.

## Status

Early draft. Seeking collaborators and implementers.

## Contributing

Contributions welcome. Please open an issue before submitting large changes.

## License

[CC0 1.0 Universal](LICENSE) — this specification is dedicated to the public domain.
