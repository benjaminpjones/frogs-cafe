# BIK-03: Identity

**Status**: Draft  
**Dependencies**: [BIK-02](02-discovery.md)  
**Normative references**: [HTTP Signatures (draft-cavage-http-signatures-12)](https://datatracker.ietf.org/doc/html/draft-cavage-http-signatures), [RFC 8017 (RSA)](https://www.rfc-editor.org/rfc/rfc8017)

---

## 1. Overview

BIK uses two levels of identity:

- **Player identity**: A player is identified by their federated address (`alice@chess.example`) and their AP Actor document. Their public key is embedded in the Actor document.
- **Server identity**: A server is identified by its domain and its server-level signing key, published in the capability document.

Authentication between servers uses **HTTP Signatures**, consistent with the ActivityPub federation model. This means BIK server-to-server communication is compatible with existing AP implementations.

## 2. Key Requirements

Both players and servers MUST have an ed25519 keypair. RSA (2048-bit minimum) is also accepted for compatibility with existing AP implementations, but ed25519 is RECOMMENDED for new implementations.

Key IDs are URLs:
- Player key: `https://chess.example/players/alice#main-key`
- Server key: `https://chess.example#server-key`

Public keys MUST be published in PEM format in the Actor document or capability document respectively (see [BIK-02](02-discovery.md)).

## 3. HTTP Signatures

All BIK server-to-server HTTP requests MUST be signed using HTTP Signatures.

The `Signature` header MUST include:
- `keyId`: URL of the signing key
- `algorithm`: `ed25519` or `rsa-sha256`
- `headers`: MUST include `(request-target)`, `host`, `date`, and `digest` (for POST requests)
- `signature`: Base64-encoded signature

Example signed request:

```http
POST /bik/inbox HTTP/1.1
Host: go.example
Date: Sat, 28 Feb 2026 12:00:00 GMT
Content-Type: application/activity+json
Digest: SHA-256=base64encodeddigest==
Signature: keyId="https://chess.example#server-key",
           algorithm="ed25519",
           headers="(request-target) host date digest",
           signature="base64encodedsignature=="

{ ... }
```

### 3.1 Signature Verification

Upon receiving a signed request, the receiving server MUST:

1. Extract `keyId` from the `Signature` header
2. Fetch the public key from that URL (with caching)
3. Verify the signature over the signed string
4. Reject requests with a `Date` header more than 30 seconds from the server's current time (replay protection)
5. Return `401 Unauthorized` if verification fails

### 3.2 Request Signing by Players vs. Servers

Server-to-server requests (e.g., delivering a seek, proposing a match) are signed with the **server key**. This is consistent with how ActivityPub works — servers sign on behalf of their users for delivery.

When a player action needs to be individually attributable (e.g., a signed move in a match), the move envelope is signed with the **player's key** and then delivered by the server. See [BIK-05](05-match-lifecycle.md) for move signing details.

## 4. Player Address Portability

Player addresses are bound to their home server. If a player moves servers, their address changes. Cross-server migration is out of scope for v1 but SHOULD be considered in future versions (AP's `Move` activity is a starting point).

## 5. Server Key Rotation

Servers MAY rotate their signing keys. When rotating:

1. Publish the new key at a new key ID URL
2. Continue accepting verification with the old key for 48 hours
3. After 48 hours, the old key MAY be removed

Peers that have cached the old key will encounter verification failures and MUST re-fetch the capability document to discover the new key.
