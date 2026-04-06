import { createHash, generateKeyPairSync, sign } from "node:crypto";

export interface BikKeys {
  publicKey: Buffer;
  privateKey: Buffer;
  kid: string;
}

/** Derive kid the same way as Go: hex(sha256(publicKeyBytes)[:8]) */
function deriveKid(publicKey: Buffer): string {
  const hash = createHash("sha256").update(publicKey).digest();
  return hash.subarray(0, 8).toString("hex");
}

/** Generate a new Ed25519 keypair for BIK token signing. */
export function generateKeys(): BikKeys {
  const { publicKey, privateKey } = generateKeyPairSync("ed25519", {
    publicKeyEncoding: { type: "spki", format: "der" },
    privateKeyEncoding: { type: "pkcs8", format: "der" },
  });

  // Ed25519 DER-encoded SPKI has a 12-byte prefix before the 32-byte raw key
  const rawPublic = publicKey.subarray(12);

  return {
    publicKey: rawPublic,
    privateKey: privateKey,
    kid: deriveKid(rawPublic),
  };
}

/** Return JWK set matching the format from /.well-known/bik/keys */
export function jwkSet(keys: BikKeys) {
  return {
    keys: [
      {
        kty: "OKP",
        crv: "Ed25519",
        kid: keys.kid,
        x: base64url(keys.publicKey),
      },
    ],
  };
}

/**
 * Sign a BIK token matching the Go format:
 * base64url(jsonPayload).base64url(ed25519Signature)
 *
 * Payload field order must match Go's json.Marshal of BikToken:
 * { kid, player, game, exp }
 */
export function signToken(
  keys: BikKeys,
  player: string,
  gameURI: string,
  ttlSeconds: number = 300,
): string {
  const payload = JSON.stringify({
    kid: keys.kid,
    player,
    game: gameURI,
    exp: Math.floor(Date.now() / 1000) + ttlSeconds,
  });

  const payloadBytes = Buffer.from(payload, "utf-8");
  const signature = sign(null, payloadBytes, {
    key: keys.privateKey,
    format: "der",
    type: "pkcs8",
  });

  return base64url(payloadBytes) + "." + base64url(signature);
}

function base64url(buf: Buffer): string {
  return buf.toString("base64url");
}
