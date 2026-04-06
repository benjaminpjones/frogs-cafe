# BIK Protocol Compliance Checker

Black-box test suite that validates any BIK server against the [protocol spec](../spec/protocol.md). The checker acts as a mock federation peer and exercises the target server through spec-defined interfaces only.

## How it works

The compliance checker runs a lightweight HTTP server (the "mock server") that impersonates a BIK federation peer. It exposes WebFinger, actor documents, JWK keys, and an inbox — just enough for the target server to treat it as a real remote server.

Tests create challenges on the target, accept them as the mock's remote player, connect via WebSocket, and verify the target's behavior matches the spec.

## Prerequisites

- Node.js 22+
- A running BIK server to test against
- A test account on that server (username + session token)

## Setup

```bash
cd bik/compliance
npm install
```

Create a test account on the target server:

```bash
curl -X POST http://localhost:8080/api/v1/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"testbot","email":"testbot@test.com","password":"testpass123"}'
```

Save the `token` from the response.

## Running

```bash
BIK_TARGET=http://localhost:8080 \
BIK_TOKEN=<session-token> \
BIK_USERNAME=testbot \
npx vitest run
```

Or via justfile:

```bash
just compliance http://localhost:8080 <session-token> testbot
```

### Environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `BIK_TARGET` | yes | | Base URL of the server under test |
| `BIK_TOKEN` | yes | | Session token for the test account |
| `BIK_USERNAME` | yes | | Username of the test account |
| `BIK_TARGET_WS` | no | derived from `BIK_TARGET` | WebSocket URL (if different from HTTP) |
| `MOCK_PORT` | no | `9900` | Port the mock server listens on |
| `MOCK_URL` | no | `http://localhost:9900` | URL the target uses to reach the mock |

### Docker note

If the target server runs in Docker, it can't reach `localhost:9900` on the host. Set `MOCK_URL` to the Docker-resolvable address:

```bash
MOCK_URL=http://host.docker.internal:9900 \
BIK_TARGET=http://localhost:8080 \
BIK_TOKEN=<token> \
BIK_USERNAME=testbot \
npx vitest run
```

## What's tested

| Suite | Tests | Description |
|-------|-------|-------------|
| Discovery | 5 | WebFinger, actor document, JWK keys endpoint |
| Challenge Flow | 4 | Create, accept, expiry rejection, double-accept rejection |
| Auth | 3 | Valid BIK token, expired token, wrong signing key |
| Gameplay | 6 | game_state on connect, moves, pass, out-of-turn rejection, resign, reconnect |
| Scoring | 4 (todo) | Scoring phase, dead stones, accept/reject |
| Clock | 2 (todo) | Clock in move_played, periodic clock broadcasts |

## Output

The checker produces a compliance report:

```
BIK Protocol Compliance Report
Target: http://localhost:8080
Date:   2026-04-06

DISCOVERY
  [PASS] returns actor link for known user
  [PASS] returns 404 for unknown user
  [PASS] returns 400 for non-acct resource
  [FAIL] returns valid AP actor with required fields
         expected 404 to be 200 // Object.is equality
  [PASS] returns valid Ed25519 key set

...

Summary: 16 passed, 2 failed, 0 skipped, 6 todo (24 total)
```

## Adding tests

Tests live in `tests/` and import shared helpers from `src/`:

- `setupGame(config)` — creates a challenge and accepts it as the mock user, returns game info
- `connectLocalPlayer(config, game)` — connects the test account to the game WebSocket
- `connectRemotePlayer(config, keys, game)` — signs a BIK token and connects as the mock user
- `player.waitFor("message_type")` — returns a promise that resolves when a matching WS message arrives
- `player.messages` — array of all received messages
