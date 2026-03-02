# BIK-06: Game Descriptor

**Status**: Draft  
**Dependencies**: [BIK-01](01-overview.md)

---

## 1. Overview

A game descriptor is a machine-readable document that describes a game type: its variants, supported time controls, and the format of move objects. Game descriptors allow servers to advertise what they support and let peers evaluate match compatibility before proposing a game.

The game descriptor format is game-agnostic. BIK does not define move formats or game rules — those are defined by the descriptor itself and enforced by implementations.

## 2. Game Descriptor Document

```json
{
  "@context": "https://bik.example/ns/v1",
  "type": "GameDescriptor",
  "id": "https://chess.example/games/chess",
  "name": "Chess",
  "shortName": "chess",
  "version": "1.0",
  "variants": [
    {
      "id": "standard",
      "name": "Standard Chess",
      "description": "FIDE rules, standard starting position"
    },
    {
      "id": "chess960",
      "name": "Chess960",
      "description": "Fischer Random Chess"
    }
  ],
  "timeControls": {
    "types": ["fischer", "bronstein", "simple", "correspondence"],
    "correspondence": {
      "maxDaysPerMove": 7
    }
  },
  "moveFormat": {
    "type": "json-schema",
    "schema": {
      "$schema": "https://json-schema.org/draft/2020-12/schema",
      "oneOf": [
        {
          "description": "Standard move",
          "type": "object",
          "properties": {
            "notation": {
              "type": "string",
              "description": "UCI notation, e.g. e2e4 or e7e8q for promotion"
            }
          },
          "required": ["notation"]
        },
        {
          "description": "Draw offer",
          "type": "object",
          "properties": {
            "offerDraw": { "type": "boolean", "const": true }
          },
          "required": ["offerDraw"]
        },
        {
          "description": "Resignation",
          "type": "object",
          "properties": {
            "resign": { "type": "boolean", "const": true }
          },
          "required": ["resign"]
        }
      ]
    }
  },
  "ratingSystem": {
    "type": "elo",
    "kFactor": 32,
    "defaultRating": 1500
  }
}
```

## 3. Field Reference

### 3.1 Top-Level Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Canonical URL of this descriptor |
| `name` | Yes | Human-readable game name |
| `shortName` | Yes | Short identifier, lowercase alphanumeric and hyphens |
| `version` | Yes | Descriptor version string |
| `variants` | Yes | Array of supported variants (at least one) |
| `timeControls` | Yes | Supported time control types and parameters |
| `moveFormat` | Yes | Schema describing valid move objects |
| `ratingSystem` | No | Rating system used by the publishing server |

### 3.2 Variants

Each variant has an `id` (used in Seek and MatchProposal objects), a `name`, and an optional `description`. The `id` MUST be unique within the descriptor and stable across versions.

### 3.3 Time Controls

The `types` array lists supported time control types. Defined types:

| Type | Description |
|------|-------------|
| `fischer` | Base time + increment per move |
| `bronstein` | Base time + delay (time only added if not fully used) |
| `simple` | Fixed time per move |
| `hourglass` | Time used by one player accrues to the other |
| `correspondence` | Days per move |
| `unlimited` | No time limit |

Time control objects in Seek and MatchProposal use these types with type-specific parameters:

```json
{ "type": "fischer", "initial": 300, "increment": 3 }
{ "type": "correspondence", "daysPerMove": 3 }
{ "type": "unlimited" }
```

Times are in seconds unless otherwise noted.

### 3.4 Move Format

The `moveFormat` object defines how moves are encoded in `MoveEvent.move`. The `type` field indicates the schema language; `json-schema` is required. Servers MUST validate moves against this schema before signing and forwarding them.

The move format MUST include a way to express resignation and draw offers, as these are required by [BIK-05](05-match-lifecycle.md).

### 3.5 Rating System

The optional `ratingSystem` object describes how the publishing server calculates ratings. This is informational — it does not constrain how other servers handle cross-server results internally. Common types: `elo`, `glicko2`, `trueskill`.

## 4. Game Compatibility

Two servers support a compatible game if:
- They both reference a game descriptor with the same `shortName`
- They agree on a `variant` id
- They agree on a `timeControl` type and parameters

Servers do NOT need to reference the same descriptor URL. Two servers may each publish their own chess descriptor at different URLs; compatibility is determined by `shortName` + variant + time control agreement.

This allows servers to extend descriptors (e.g., adding local variants) without breaking interoperability on the standard variants.

## 5. Well-Known Game Short Names

To encourage interoperability, the following `shortName` values are reserved and their semantics are standardized:

| Short Name | Game |
|------------|------|
| `chess` | Chess (FIDE rules) |
| `chess960` | Fischer Random Chess |
| `go` | Go (with komi specified in variant) |
| `shogi` | Shogi |
| `checkers` | Checkers/Draughts |
| `backgammon` | Backgammon |
| `reversi` | Reversi/Othello |

Servers implementing these games SHOULD use these short names. Custom or experimental games SHOULD use reverse-domain namespacing: `com.example.mygame`.
