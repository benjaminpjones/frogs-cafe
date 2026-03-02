# BIK-07: Rank Maps

**Status**: Draft  
**Dependencies**: [BIK-02](02-discovery.md), [BIK-03](03-identity.md)

---

## 1. Overview

A rank map is a server-maintained translation table expressing the equivalence between ranks on two different servers. The core assumption is:

> All players of rank X on server A are roughly equivalent to players of rank Y on server B.

Rank maps are **unilateral**: each server maintains its own maps independently. Server A's map for server B does not need to agree with server B's map for server A. This is intentional — it reflects that each server has its own perspective on rank equivalence based on its observed cross-server game data.

Rank maps are used by the seek protocol ([BIK-04](04-seek.md)) to evaluate rating range compatibility across servers.

## 2. Rank Map Document

```json
{
  "@context": "https://bik.example/ns/v1",
  "type": "RankMap",
  "id": "https://chess.example/rank-maps/go.example/chess",
  "fromServer": "https://chess.example",
  "toServer": "https://go.example",
  "game": "chess",
  "variant": "standard",
  "updatedAt": "2026-02-28T00:00:00Z",
  "sampleSize": 4200,
  "method": "empirical",
  "entries": [
    { "from": 1000, "to": 950 },
    { "from": 1100, "to": 1040 },
    { "from": 1200, "to": 1135 },
    { "from": 1300, "to": 1230 },
    { "from": 1400, "to": 1330 },
    { "from": 1500, "to": 1425 },
    { "from": 1600, "to": 1520 },
    { "from": 1700, "to": 1615 },
    { "from": 1800, "to": 1710 },
    { "from": 1900, "to": 1810 },
    { "from": 2000, "to": 1915 }
  ],
  "serverSignature": "..."
}
```

### 2.1 Interpretation

`entries` is an array of `{ from, to }` pairs where:
- `from` is a rank on `fromServer`
- `to` is the equivalent rank on `toServer`

Entries MUST be sorted by `from` ascending. For ranks between listed entries, implementations SHOULD use linear interpolation.

`fromServer` is the server publishing this map. The map expresses: "in my experience, my rank X is equivalent to rank Y on the other server."

### 2.2 Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Canonical URL of this map |
| `fromServer` | Yes | The server publishing this map (always the server hosting the document) |
| `toServer` | Yes | The server being mapped to |
| `game` | Yes | Game short name |
| `variant` | Yes | Variant id |
| `updatedAt` | Yes | When this map was last updated |
| `sampleSize` | No | Number of cross-server games this map is based on |
| `method` | Yes | How the map was derived (see Section 4) |
| `entries` | Yes | Translation table, sorted by `from` ascending |
| `serverSignature` | Yes | Signature by `fromServer`'s signing key |

## 3. Publishing and Discovery

Servers SHOULD publish rank maps at a well-known path:

```
GET /bik/rank-maps/{peer-domain}/{game-shortname}
```

For example:
```
GET https://chess.example/bik/rank-maps/go.example/chess
```

A server may also list all available rank maps:

```
GET /bik/rank-maps
```

Returns an array of rank map URLs.

Rank maps are signed by the publishing server and MAY be cached by peers. A `Cache-Control` max-age of 24 hours is RECOMMENDED. Maps SHOULD be re-fetched before any cross-server matching session if the cached copy is older than the max-age.

## 4. Derivation Methods

The `method` field indicates how the map was derived.

### 4.1 `empirical`

Derived from actual cross-server game results. For each cross-server game, observed win rates are used to calibrate rank equivalence (similar to how cross-pool ELO calculations work). This is the most accurate method but requires a cold-start period.

Recommended minimum sample size: 100 games per rating band. Maps with `sampleSize < 100` SHOULD be treated with low confidence.

### 4.2 `distribution`

Derived by aligning rating percentiles across the two servers' player populations. Server A's 60th percentile maps to server B's 60th percentile. Servers share their rating distributions (histograms, not individual player data) to compute this.

This method works without cross-server game history and serves as a useful cold-start prior. It assumes similar skill distributions at each percentile, which may not hold if the server populations differ significantly.

### 4.3 `manual`

Set by a server operator, based on their judgment. MUST include a `note` field explaining the rationale. This is appropriate for small servers with insufficient game data, as a temporary measure.

### 4.4 `transitive`

Derived by composing two other maps. If `chess.example` has a map for `go.example` and `go.example` has a map for `shogi.example`, `chess.example` can derive a transitive map for `shogi.example`. Transitive maps MUST reference the intermediate maps used:

```json
{
  "method": "transitive",
  "via": [
    "https://chess.example/rank-maps/go.example/chess",
    "https://go.example/rank-maps/shogi.example/chess"
  ]
}
```

Transitive maps SHOULD be treated with lower confidence than direct maps. Implementations SHOULD prefer direct empirical maps when available.

## 5. Cold Start

When two servers have no cross-server game history:

1. Start with `distribution` method maps based on exchanged rating histograms
2. Begin accepting cross-server matches with relaxed rating range requirements
3. Accumulate game data and transition to `empirical` method over time

Servers SHOULD exchange rating histograms upon establishing a new peer relationship. A rating histogram is a simple JSON array of `{ boundary, count }` objects representing player counts per rating band — no individual player data is exposed.

```json
{
  "type": "RatingHistogram",
  "server": "https://chess.example",
  "game": "chess",
  "variant": "standard",
  "updatedAt": "2026-02-28T00:00:00Z",
  "bands": [
    { "min": 1000, "max": 1099, "count": 142 },
    { "min": 1100, "max": 1199, "count": 287 },
    { "min": 1200, "max": 1299, "count": 531 }
  ]
}
```

## 6. Using Rank Maps in Matching

When evaluating a seek from a remote server, the receiving server:

1. Fetches (or uses cached) rank map for the remote server
2. Translates the seek's `ratingRange` from remote scale to local scale using the map
3. Checks whether any local players' ratings fall within the translated range
4. Proceeds with matching if compatible

If no rank map exists for the remote server, the receiving server MAY:
- Decline cross-server matching with that server until a map is established
- Fall back to distribution-based estimation
- Accept any rating range (open matching), configurable by operator

Server operators SHOULD configure a policy for this case.
