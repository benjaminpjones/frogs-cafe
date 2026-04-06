import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    testTimeout: 30_000,
    hookTimeout: 15_000,
    sequence: { concurrent: false },
    reporters: ["./src/reporter.ts"],
    globalSetup: ["./src/global-setup.ts"],
  },
});

declare module "vitest" {
  export interface ProvidedContext {
    bikKeys: {
      publicKey: number[];
      privateKey: number[];
      kid: string;
    };
  }
}
