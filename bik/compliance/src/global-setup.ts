import type { GlobalSetupContext } from "vitest/node";
import { loadConfig } from "./config.js";
import { generateKeys } from "./keys.js";
import { startMockServer, type MockServer } from "./mock-server.js";

let mockServer: MockServer;

export default async function setup({ provide }: GlobalSetupContext) {
  const config = loadConfig();
  const keys = generateKeys();

  mockServer = await startMockServer({
    port: config.mockPort,
    externalUrl: config.mockUrl,
    keys,
  });

  // Share keys with test workers via serializable data
  provide("bikKeys", {
    publicKey: Array.from(keys.publicKey),
    privateKey: Array.from(keys.privateKey),
    kid: keys.kid,
  });
}

export async function teardown() {
  await mockServer?.close();
}
