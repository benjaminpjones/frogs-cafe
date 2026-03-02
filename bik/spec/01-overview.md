# BIK-01: Overview

**Status**: Draft  
**Dependencies**: None

---

## 1. Introduction

The BIK Protocol (Board InterKonnect) defines protocols for interoperability between independently-operated turn-based game servers. A conforming server can discover peers, advertise available players, negotiate matches, synchronize game state, and exchange ranking information — all without requiring a central authority.

The standard is designed for **turn-based** games specifically. This constraint is important: turns are discrete events, not continuous streams. This makes federated state synchronization tractable and means the cost of a failed match negotiation is low (a brief retry, not a broken real-time session).

## 2. Design Principles

**Reuse before specifying.** Where existing open standards solve a problem well, we profile them rather than reinvent. Novel specification work is reserved for what is genuinely unique to this domain.

**Eventual correctness over strict consistency.** Federation is inherently distributed. We prefer designs that tolerate temporary disagreement and converge, over designs that require synchronous coordination.

**Degraded operation over failure.** A server that cannot reach its peers should still serve local players. Federation enhances capability; it is not a dependency for basic operation.

**No mandatory central authority.** Optional relay servers and rank translation authorities may emerge and provide value, but conformance must never require them.

**Privacy by default.** Player seeks and game history should be scoped to the minimum necessary audience unless the player opts into broader visibility.

## 3. Terminology

**Server**: An independently-operated instance running BIK-conformant software. Identified by a domain name (e.g., `chess.example`).

**Player**: A human user with an account on a server. Identified by a federated address in the form `player@server` (e.g., `alice@chess.example`).

**Seek**: An advertisement that a player is looking for a game, with specified parameters (game type, time control, rating range, etc.).

**Match**: A single game between two players, potentially on different servers. Has a canonical identifier, a verifiable move history, and a recorded result.

**Host server**: The server responsible for coordinating a match. Typically the server of the player who issued the original seek or accepted a challenge.

**Guest server**: The other server in a federated match.

**Relay**: An optional server that aggregates seeks from multiple peers and performs cross-server matching. Participation is voluntary.

**Game descriptor**: A machine-readable document describing a game type, its variants, supported time controls, and move format.

**Rank map**: A server-maintained translation table mapping ranks on one server to equivalent ranks on another.

## 4. Conformance

A server MAY implement any subset of BIK. However, the following capability levels are defined for interoperability purposes:

| Level | Name | Required capabilities |
|-------|------|-----------------------|
| 0 | **Observer** | Discovery, identity |
| 1 | **Participant** | + Match lifecycle (can play federated matches when invited) |
| 2 | **Seeker** | + Seek advertisement (participates in federated automatch) |
| 3 | **Relay** | + Seek aggregation and cross-server matching for peers |

A server MUST advertise its conformance level in its capability document (see [BIK-02](02-discovery.md)).

## 5. Document Map

- **[BIK-02: Discovery](02-discovery.md)** — How servers and players are discovered. Profiles WebFinger and ActivityPub actor discovery.
- **[BIK-03: Identity](03-identity.md)** — Player identity, server signing keys, and request authentication. Profiles HTTP Signatures.
- **[BIK-04: Seek](04-seek.md)** — The federated automatch pool. Novel specification.
- **[BIK-05: Match Lifecycle](05-match-lifecycle.md)** — Match creation through result recording. Profiles Matrix event DAG for state sync.
- **[BIK-06: Game Descriptor](06-game-descriptor.md)** — Format for describing games and their variants. Novel specification.
- **[BIK-07: Rank Maps](07-rank-maps.md)** — Cross-server rank translation. Novel specification.

## 6. Data Formats

All BIK messages MUST be encoded as JSON. All BIK endpoints MUST be served over HTTPS. 

Where this standard defines JSON schemas, implementations MUST accept unknown fields gracefully (ignore them) to allow forward compatibility.

## 7. Versioning

This standard uses semantic versioning. The version is advertised in the server capability document. Servers SHOULD negotiate the highest mutually supported version when establishing a federated match.
