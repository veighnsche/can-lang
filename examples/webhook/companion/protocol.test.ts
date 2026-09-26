// F05 companion protocol tests: signing vectors, envelope shape,
// response parsers (valid plus fail-closed), and client validation.
import { test, expect } from "bun:test";
import { createHmac } from "node:crypto";
import {
  CARRIER_BODY_LIMIT,
  LEASE_MAX_MS,
  LEASE_MIN_MS,
  MAX_ATTEMPTS,
  PROTOCOL_VERSION,
  CarrierHttp,
  carrierHeaders,
  createCarrierClient,
  newNonce,
  parseAckResult,
  parseClaimPage,
  parseDeadResult,
  parseHeartbeatResult,
  signBody,
  type CarrierTransport,
} from "./protocol.ts";

test("protocol v1 pins version, attempts, lease bounds, and body limit", () => {
  expect(PROTOCOL_VERSION).toBe("1");
  expect(MAX_ATTEMPTS).toBe(5);
  expect([LEASE_MIN_MS, LEASE_MAX_MS]).toEqual([1000, 600000]);
  expect(CARRIER_BODY_LIMIT).toBe(1024);
});

test("signBody matches HMAC-SHA-256 hex and rejects empty secrets", () => {
  const secret = "carrier-test-0123456789abcdef";
  const body = '{"worker_id":"worker-1","lease_ms":5000,"timestamp_ms":1000000,"nonce":"n-1"}';
  expect(signBody(secret, body)).toBe(
    createHmac("sha256", secret).update(body, "utf8").digest("hex"),
  );
  expect(() => signBody("", body)).toThrow(TypeError);
});

test("carrierHeaders envelope the exact bytes and enforce the limit", () => {
  const body = '{"a":1}';
  const headers = carrierHeaders("s3cret", body);
  expect(headers["x-carrier-protocol"]).toBe("1");
  expect(headers["x-carrier-signature"]).toBe(signBody("s3cret", body));
  expect(headers["content-type"]).toBe("application/json");
  expect(() => carrierHeaders("s3cret", "x".repeat(1025))).toThrow(TypeError);
});

test("newNonce mints unique hex nonces", () => {
  const seen = new Set([newNonce(), newNonce(), newNonce()]);
  expect(seen.size).toBe(3);
  for (const nonce of seen) expect(/^[0-9a-f]{32}$/.test(nonce)).toBe(true);
});

test("claim pages parse zero or one item and fail closed otherwise", () => {
  const item = {
    delivery_id: "del-1",
    event: "invoice.paid",
    subscription: "sub-9",
    attempts: 0,
    lease_until_ms: 1005000,
    version: 1,
  };
  expect(parseClaimPage({ status: "claimed", items: [item] }).items).toHaveLength(1);
  expect(parseClaimPage({ status: "empty", items: [] }).items).toHaveLength(0);
  for (const bad of [
    null,
    { status: "claimed" },
    { status: "claimed", items: [item, item] },
    { status: "claimed", items: [{ ...item, attempts: "0" }] },
    { status: "claimed", items: [{ ...item, version: 1.5 }] },
    { status: 200, items: [] },
  ]) {
    expect(() => parseClaimPage(bad)).toThrow(TypeError);
  }
});

test("heartbeat, ack, and dead results validate strictly", () => {
  expect(
    parseHeartbeatResult({ status: "extended", lease_until_ms: 2, version: 2, attempts: 0 }).status,
  ).toBe("extended");
  expect(parseAckResult({ status: "done", attempts: 1 }).attempts).toBe(1);
  expect(parseDeadResult({ status: "dead", attempts: 5 }).status).toBe("dead");
  expect(() => parseHeartbeatResult({ status: "extended", lease_until_ms: 2, version: 2 })).toThrow(
    TypeError,
  );
  expect(() => parseAckResult({ status: "done", attempts: -1.5 })).toThrow(TypeError);
  expect(() => parseDeadResult({ status: "dead" })).toThrow(TypeError);
});

test("client signs timestamped bodies and routes routine statuses", async () => {
  const seen: { path: string; headers: Record<string, string>; body: string }[] = [];
  const transport: CarrierTransport = async (path, headers, body) => {
    seen.push({ path, headers, body });
    if (path === "/outbox/claim")
      return {
        status: 200,
        json: {
          status: "claimed",
          items: [
            {
              delivery_id: "del-1",
              event: "e",
              subscription: "s",
              attempts: 0,
              lease_until_ms: 9,
              version: 1,
            },
          ],
        },
      };
    return { status: 404, json: { status: "unknown", attempts: 0 } };
  };
  const client = createCarrierClient(
    transport,
    "s3cret",
    () => 1000000,
    () => "n-1",
  );
  const page = await client.claim("worker-1", 5000);
  expect(page.items[0].deliveryId).toBe("del-1");
  const call = seen[0];
  expect(call.headers["x-carrier-protocol"]).toBe("1");
  expect(call.headers["x-carrier-signature"]).toBe(signBody("s3cret", call.body));
  expect(JSON.parse(call.body)).toEqual({
    worker_id: "worker-1",
    lease_ms: 5000,
    timestamp_ms: 1000000,
    nonce: "n-1",
  });
  const ack = await client.ack("del-9", true);
  expect(ack.status).toBe("unknown");
});

test("client validates worker, lease, version, and reason before sending", async () => {
  let calls = 0;
  const transport: CarrierTransport = async () => {
    calls += 1;
    return { status: 200, json: { status: "empty", items: [] } };
  };
  const client = createCarrierClient(transport, "s3cret");
  await expect(client.claim("", 5000)).rejects.toThrow(TypeError);
  await expect(client.claim("w", 999)).rejects.toThrow(TypeError);
  await expect(client.claim("w", 600001)).rejects.toThrow(TypeError);
  await expect(client.heartbeat("w", "d", -1, 5000)).rejects.toThrow(TypeError);
  await expect(client.deadLetter("d", "w", "tired")).rejects.toThrow(TypeError);
  expect(() => createCarrierClient(transport, "")).toThrow(TypeError);
  expect(calls).toBe(0);
});

test("client throws CarrierHttp off the routine statuses", async () => {
  const transport: CarrierTransport = async (path) => {
    if (path === "/outbox/claim") return { status: 401, json: { status: "stale" } };
    if (path === "/outbox/ack") return { status: 503, json: { status: "store_unavailable" } };
    return { status: 500, json: null };
  };
  const client = createCarrierClient(transport, "s3cret");
  const denied = await client.claim("w", 5000).catch((cause: unknown) => cause);
  expect(denied).toBeInstanceOf(CarrierHttp);
  expect((denied as CarrierHttp).status).toBe(401);
  expect((denied as CarrierHttp).bodyStatus).toBe("stale");
  const pressure = await client.ack("d", true).catch((cause: unknown) => cause);
  expect((pressure as CarrierHttp).status).toBe(503);
  const broken = await client.deadLetter("d", "w", "poison").catch((cause: unknown) => cause);
  expect((broken as CarrierHttp).status).toBe(500);
  expect((broken as CarrierHttp).bodyStatus).toBe("unparseable-body");
});
