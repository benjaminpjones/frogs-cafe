export interface Config {
  /** Base URL of the target BIK server (e.g. http://localhost:8080) */
  target: string;
  /** WebSocket base URL of the target (derived from target if omitted) */
  targetWs: string;
  /** Session token for a test account on the target server */
  token: string;
  /** Username of the test account on the target */
  username: string;
  /** Port the mock server listens on */
  mockPort: number;
  /** URL the target uses to reach the mock server */
  mockUrl: string;
}

function required(name: string): string {
  const val = process.env[name];
  if (!val) throw new Error(`Missing required env var: ${name}`);
  return val;
}

export function loadConfig(): Config {
  const target = required("BIK_TARGET");
  const mockPort = parseInt(process.env["MOCK_PORT"] ?? "9900", 10);
  const mockUrl = process.env["MOCK_URL"] ?? `http://localhost:${mockPort}`;
  const targetWs =
    process.env["BIK_TARGET_WS"] ?? target.replace(/^http/, "ws");

  return {
    target,
    targetWs,
    token: required("BIK_TOKEN"),
    username: required("BIK_USERNAME"),
    mockPort,
    mockUrl,
  };
}
