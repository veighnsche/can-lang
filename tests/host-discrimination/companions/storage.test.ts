import { test, expect } from "bun:test";
import { envName, evaluateDestination, policyReport } from "../../../runtime/outbound/destination-policy.ts";
import {
  createStorageCompanionClient,
  createStorageCompanionServer,
  decodeStorageRequest,
  decodeStorageResponse,
  encodeStorageRequest,
  encodeStorageResponse,
  storageCompanionContext,
  storageCompanionPolicy,
  STORAGE_PROTOCOL,
  type CompanionTransport,
} from "./storage.ts";

const SECRET = "D01-PROBE-SECRET-VALUE-7f3a9c";
const ENDPOINT = "https://companion.example/v1/storage";

function pair(correlation = "corr-0001", env: Record<string, string | undefined> = {}) {
  const readEnvironment = (key: string): string | undefined =>
    env[key] ?? (key === "D01_STORAGE_COMPANION_TOKEN" ? SECRET : undefined);
  const server = createStorageCompanionServer({
    policy: storageCompanionPolicy(),
    credentialEnv: envName("D01_STORAGE_COMPANION_TOKEN"),
    readEnvironment,
  });
  const transport: CompanionTransport = {
    send(url: string, authorization: string, body: Uint8Array): Uint8Array {
      expect(url).toBe(ENDPOINT);
      return server.handle(authorization, body);
    },
  };
  const client = createStorageCompanionClient({
    policy: storageCompanionPolicy(),
    endpoint: ENDPOINT,
    context: storageCompanionContext(correlation),
    credentialEnv: envName("D01_STORAGE_COMPANION_TOKEN"),
    readEnvironment,
    transport,
  });
  return { client, server, transport };
}

test("roundtrip: set, get, remove, missing reads null", async () => {
  const { client } = pair();
  expect(await client.localSet("theme", "dark")).toEqual({ ok: true, value: null });
  expect(await client.localGet("theme")).toEqual({ ok: true, value: "dark" });
  expect(await client.localRemove("theme")).toEqual({ ok: true, value: null });
  expect(await client.localGet("theme")).toEqual({ ok: true, value: null });
});

test("invalid keys reject client-side without sending", async () => {
  let sent = 0;
  const { transport } = pair();
  const counting: CompanionTransport = {
    send(url: string, authorization: string, body: Uint8Array): Uint8Array {
      sent += 1;
      return transport.send(url, authorization, body);
    },
  };
  const readEnvironment = (key: string): string | undefined =>
    key === "D01_STORAGE_COMPANION_TOKEN" ? SECRET : undefined;
  const client = createStorageCompanionClient({
    policy: storageCompanionPolicy(),
    endpoint: ENDPOINT,
    context: storageCompanionContext("corr-0002"),
    credentialEnv: envName("D01_STORAGE_COMPANION_TOKEN"),
    readEnvironment,
    transport: counting,
  });
  const got = await client.localGet("");
  expect(got.ok).toBe(false);
  if (!got.ok) expect(got.failure.code).toBe("storage::invalid_key");
  expect(sent).toBe(0);
});

test("denied destination fails closed without sending", async () => {
  let sent = 0;
  const readEnvironment = (key: string): string | undefined =>
    key === "D01_STORAGE_COMPANION_TOKEN" ? SECRET : undefined;
  const client = createStorageCompanionClient({
    policy: storageCompanionPolicy(),
    endpoint: "https://evil.example/v1/storage",
    context: storageCompanionContext("corr-0003"),
    credentialEnv: envName("D01_STORAGE_COMPANION_TOKEN"),
    readEnvironment,
    transport: { send: () => { sent += 1; return new Uint8Array(); } },
  });
  const got = await client.localGet("k");
  expect(got.ok).toBe(false);
  if (!got.ok) {
    expect(got.failure.code).toBe("companion::destination_denied");
    expect(got.failure.detail).toContain("no-matching-rule");
  }
  expect(sent).toBe(0);
});

test("bad credential maps to unauthorized on both sides", async () => {
  const { client, server } = pair("corr-0004", { D01_STORAGE_COMPANION_TOKEN: "WRONG-VALUE" });
  // Client resolves the wrong value from its environment; the server
  // below expects the right one, so the pair disagrees honestly.
  const strict = createStorageCompanionServer({
    policy: storageCompanionPolicy(),
    credentialEnv: envName("D01_STORAGE_COMPANION_TOKEN"),
    readEnvironment: (key: string) => (key === "D01_STORAGE_COMPANION_TOKEN" ? SECRET : undefined),
  });
  void server;
  const direct = createStorageCompanionClient({
    policy: storageCompanionPolicy(),
    endpoint: ENDPOINT,
    context: storageCompanionContext("corr-0004"),
    credentialEnv: envName("D01_STORAGE_COMPANION_TOKEN"),
    readEnvironment: (key: string) => (key === "D01_STORAGE_COMPANION_TOKEN" ? "WRONG-VALUE" : undefined),
    transport: { send: (url: string, authorization: string, body: Uint8Array) => strict.handle(authorization, body) },
  });
  void client;
  const got = await direct.localGet("k");
  expect(got.ok).toBe(false);
  if (!got.ok) expect(got.failure.code).toBe("companion::unauthorized");
  expect(strict.diagnostics().some((line) => line.startsWith("reject:companion::unauthorized"))).toBe(true);
});

test("version mismatch fails closed both directions", () => {
  const request = decodeStorageRequest(
    new TextEncoder().encode(JSON.stringify({ v: "d01.storage/0", op: "get", key: "k" })),
  );
  expect("code" in request && request.code).toBe("companion::version_mismatch");
  const response = decodeStorageResponse(
    new TextEncoder().encode(JSON.stringify({ v: "d01.storage/0", correlation: "c", ok: true, value: null })),
  );
  expect("code" in response && response.code).toBe("companion::version_mismatch");
});

test("transport failure maps to transport_failed", async () => {
  const readEnvironment = (key: string): string | undefined =>
    key === "D01_STORAGE_COMPANION_TOKEN" ? SECRET : undefined;
  const client = createStorageCompanionClient({
    policy: storageCompanionPolicy(),
    endpoint: ENDPOINT,
    context: storageCompanionContext("corr-0005"),
    credentialEnv: envName("D01_STORAGE_COMPANION_TOKEN"),
    readEnvironment,
    transport: { send: () => { throw new Error("dial failed"); } },
  });
  const got = await client.localGet("k");
  expect(got.ok).toBe(false);
  if (!got.ok) {
    expect(got.failure.code).toBe("companion::transport_failed");
    expect(JSON.stringify(got)).not.toContain("dial failed");
  }
});

test("malformed response maps to protocol_error", async () => {
  const readEnvironment = (key: string): string | undefined =>
    key === "D01_STORAGE_COMPANION_TOKEN" ? SECRET : undefined;
  const client = createStorageCompanionClient({
    policy: storageCompanionPolicy(),
    endpoint: ENDPOINT,
    context: storageCompanionContext("corr-0006"),
    credentialEnv: envName("D01_STORAGE_COMPANION_TOKEN"),
    readEnvironment,
    transport: { send: () => new TextEncoder().encode("not json{{{") },
  });
  const got = await client.localGet("k");
  expect(got.ok).toBe(false);
  if (!got.ok) expect(got.failure.code).toBe("companion::protocol_error");
});

test("server quota passes through as quota_exceeded", async () => {
  const readEnvironment = (key: string): string | undefined =>
    key === "D01_STORAGE_COMPANION_TOKEN" ? SECRET : undefined;
  const server = createStorageCompanionServer({
    policy: storageCompanionPolicy(),
    credentialEnv: envName("D01_STORAGE_COMPANION_TOKEN"),
    readEnvironment,
    quotaChars: 4,
  });
  const client = createStorageCompanionClient({
    policy: storageCompanionPolicy(),
    endpoint: ENDPOINT,
    context: storageCompanionContext("corr-0007"),
    credentialEnv: envName("D01_STORAGE_COMPANION_TOKEN"),
    readEnvironment,
    transport: { send: (url: string, authorization: string, body: Uint8Array) => server.handle(authorization, body) },
  });
  expect((await client.localSet("a", "1234")).ok).toBe(true);
  const over = await client.localSet("b", "x");
  expect(over.ok).toBe(false);
  if (!over.ok) expect(over.failure.code).toBe("storage::quota_exceeded");
});

test("auth parity: same policy admits on both sides; secret never logged", async () => {
  const policy = storageCompanionPolicy();
  const { client, server } = pair("corr-0008");
  expect(evaluateDestination(policy, ENDPOINT).decision).toBe("allowed");
  expect(server.policyVersion).toBe(policy.version);
  expect((await client.localSet("k", "v")).ok).toBe(true);
  const exposed = JSON.stringify({
    report: policyReport(policy),
    decision: evaluateDestination(policy, ENDPOINT),
    diagnostics: server.diagnostics(),
  });
  expect(exposed).not.toContain(SECRET);
  expect(exposed).toContain("D01_STORAGE_COMPANION_TOKEN");
});

test("envelope bytes are deterministic and measured", () => {
  const request = encodeStorageRequest({
    v: STORAGE_PROTOCOL,
    tenant: "tenant-acme",
    pool: "pool-default",
    correlation: "corr-0001",
    op: "get",
    key: "theme",
  });
  const setRequest = encodeStorageRequest({
    v: STORAGE_PROTOCOL,
    tenant: "tenant-acme",
    pool: "pool-default",
    correlation: "corr-0001",
    op: "set",
    key: "theme",
    value: "dark",
  });
  const response = encodeStorageResponse({ v: STORAGE_PROTOCOL, correlation: "corr-0001", ok: true, value: "dark" });
  // Pinned measurements for the record: per-call wire cost the
  // adapter tier pays zero of.
  expect(request.length).toBe(117);
  expect(setRequest.length).toBe(132);
  expect(response.length).toBe(72);
});
