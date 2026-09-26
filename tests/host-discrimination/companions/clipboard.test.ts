import { test, expect } from "bun:test";
import { envName, evaluateDestination, policyReport } from "../../../runtime/outbound/destination-policy.ts";
import { createClipboardAdapter, createFakeClipboardHost } from "../adapters/clipboard.ts";
import {
  clipboardCompanionContext,
  clipboardCompanionPolicy,
  createClipboardCompanionClient,
  createClipboardCompanionServer,
  decodeClipboardRequest,
  decodeClipboardResponse,
  encodeClipboardRequest,
  encodeClipboardResponse,
  CLIPBOARD_PROTOCOL,
  type CompanionTransport,
} from "./clipboard.ts";

const SECRET = "D01-PROBE-CLIPBOARD-SECRET-9e2b41";
const ENDPOINT = "https://companion.example/v1/clipboard";

function pair(correlation = "corr-1001") {
  const readEnvironment = (key: string): string | undefined =>
    key === "D01_CLIPBOARD_COMPANION_TOKEN" ? SECRET : undefined;
  const server = createClipboardCompanionServer({
    policy: clipboardCompanionPolicy(),
    credentialEnv: envName("D01_CLIPBOARD_COMPANION_TOKEN"),
    readEnvironment,
  });
  const transport: CompanionTransport = {
    send(url: string, authorization: string, body: Uint8Array): Uint8Array {
      expect(url).toBe(ENDPOINT);
      return server.handle(authorization, body);
    },
  };
  const client = createClipboardCompanionClient({
    policy: clipboardCompanionPolicy(),
    endpoint: ENDPOINT,
    context: clipboardCompanionContext(correlation),
    credentialEnv: envName("D01_CLIPBOARD_COMPANION_TOKEN"),
    readEnvironment,
    transport,
  });
  return { client, server };
}

test("pastebin roundtrip: write then read returns the text", async () => {
  const { client } = pair();
  expect(await client.writeText("invoice-42")).toEqual({ ok: true, value: null });
  expect(await client.readText()).toEqual({ ok: true, value: "invoice-42" });
});

test("empty companion cell reads the empty leaf", async () => {
  const { client } = pair("corr-1002");
  const got = await client.readText();
  expect(got.ok).toBe(false);
  if (!got.ok) expect(got.failure.code).toBe("clipboard::empty");
});

test("denied destination and bad credential fail closed", async () => {
  const readEnvironment = (key: string): string | undefined =>
    key === "D01_CLIPBOARD_COMPANION_TOKEN" ? SECRET : undefined;
  let sent = 0;
  const denied = createClipboardCompanionClient({
    policy: clipboardCompanionPolicy(),
    endpoint: "https://evil.example/v1/clipboard",
    context: clipboardCompanionContext("corr-1003"),
    credentialEnv: envName("D01_CLIPBOARD_COMPANION_TOKEN"),
    readEnvironment,
    transport: { send: () => { sent += 1; return new Uint8Array(); } },
  });
  const deniedResult = await denied.readText();
  expect(deniedResult.ok).toBe(false);
  if (!deniedResult.ok) expect(deniedResult.failure.code).toBe("companion::destination_denied");
  expect(sent).toBe(0);

  const strict = createClipboardCompanionServer({
    policy: clipboardCompanionPolicy(),
    credentialEnv: envName("D01_CLIPBOARD_COMPANION_TOKEN"),
    readEnvironment: (key: string) => (key === "D01_CLIPBOARD_COMPANION_TOKEN" ? SECRET : undefined),
  });
  const wrong = createClipboardCompanionClient({
    policy: clipboardCompanionPolicy(),
    endpoint: ENDPOINT,
    context: clipboardCompanionContext("corr-1004"),
    credentialEnv: envName("D01_CLIPBOARD_COMPANION_TOKEN"),
    readEnvironment: () => "WRONG-VALUE",
    transport: { send: (url: string, authorization: string, body: Uint8Array) => strict.handle(authorization, body) },
  });
  const wrongResult = await wrong.readText();
  expect(wrongResult.ok).toBe(false);
  if (!wrongResult.ok) expect(wrongResult.failure.code).toBe("companion::unauthorized");
});

test("version mismatch and transport failure map honestly", async () => {
  const request = decodeClipboardRequest(new TextEncoder().encode(JSON.stringify({ v: "d01.clipboard/0", op: "read" })));
  expect("code" in request && request.code).toBe("companion::version_mismatch");
  const response = decodeClipboardResponse(
    new TextEncoder().encode(JSON.stringify({ v: "d01.clipboard/0", correlation: "c", ok: true, text: null })),
  );
  expect("code" in response && response.code).toBe("companion::version_mismatch");
  const readEnvironment = (key: string): string | undefined =>
    key === "D01_CLIPBOARD_COMPANION_TOKEN" ? SECRET : undefined;
  const offline = createClipboardCompanionClient({
    policy: clipboardCompanionPolicy(),
    endpoint: ENDPOINT,
    context: clipboardCompanionContext("corr-1005"),
    credentialEnv: envName("D01_CLIPBOARD_COMPANION_TOKEN"),
    readEnvironment,
    transport: { send: () => { throw new Error("offline"); } },
  });
  const got = await offline.readText();
  expect(got.ok).toBe(false);
  if (!got.ok) {
    expect(got.failure.code).toBe("companion::transport_failed");
    expect(JSON.stringify(got)).not.toContain("offline");
  }
});

test("auth parity: same policy both sides; secret never logged", async () => {
  const policy = clipboardCompanionPolicy();
  const { client, server } = pair("corr-1006");
  expect(evaluateDestination(policy, ENDPOINT).decision).toBe("allowed");
  expect(server.policyVersion).toBe(policy.version);
  expect((await client.writeText("x")).ok).toBe(true);
  const exposed = JSON.stringify({
    report: policyReport(policy),
    decision: evaluateDestination(policy, ENDPOINT),
    diagnostics: server.diagnostics(),
  });
  expect(exposed).not.toContain(SECRET);
  expect(exposed).toContain("D01_CLIPBOARD_COMPANION_TOKEN");
});

test("semantic gap: companion cell and device cell are disjoint", async () => {
  // The device holds text the companion never observes: the server
  // takes no device handle, so a companion read cannot satisfy a
  // device-clipboard read. This pins the compare-target gap that
  // excludes T3 for Op B (see record).
  const deviceHost = createFakeClipboardHost();
  const device = createClipboardAdapter(deviceHost);
  expect((await device.writeText("device-secret")).ok).toBe(true);
  const { client } = pair("corr-1007");
  const companionRead = await client.readText();
  expect(companionRead.ok).toBe(false);
  if (!companionRead.ok) expect(companionRead.failure.code).toBe("clipboard::empty");
  expect(await device.readText()).toEqual({ ok: true, value: "device-secret" });
  expect((await client.writeText("server-text")).ok).toBe(true);
  expect(await device.readText()).toEqual({ ok: true, value: "device-secret" });
});

test("envelope bytes are deterministic and measured", () => {
  const request = encodeClipboardRequest({
    v: CLIPBOARD_PROTOCOL,
    tenant: "tenant-acme",
    pool: "pool-default",
    correlation: "corr-1001",
    op: "read",
  });
  const setRequest = encodeClipboardRequest({
    v: CLIPBOARD_PROTOCOL,
    tenant: "tenant-acme",
    pool: "pool-default",
    correlation: "corr-1001",
    op: "write",
    text: "invoice-42",
  });
  const response = encodeClipboardResponse({ v: CLIPBOARD_PROTOCOL, correlation: "corr-1001", ok: true, text: "invoice-42" });
  expect(request.length).toBe(106);
  expect(setRequest.length).toBe(127);
  expect(response.length).toBe(79);
});
