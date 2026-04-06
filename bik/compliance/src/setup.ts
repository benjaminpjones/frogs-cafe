import { inject } from "vitest";
import { loadConfig, type Config } from "./config.js";
import type { BikKeys } from "./keys.js";

export let config: Config;
export let keys: BikKeys;

// Config is loaded from env in each worker (same values).
// Keys are injected from the global setup (the mock server uses these keys).
config = loadConfig();

const serialized = inject("bikKeys") as {
  publicKey: number[];
  privateKey: number[];
  kid: string;
};
keys = {
  publicKey: Buffer.from(serialized.publicKey),
  privateKey: Buffer.from(serialized.privateKey),
  kid: serialized.kid,
};
