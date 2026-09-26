// F05 companion: versioned carrier protocol client (protocol v1).
//
// This module is the companion half of the authenticated Can/companion
// pair. It mirrors the Can side in examples/webhook/src: every carrier
// request carries an x-carrier-protocol version header plus an
// HMAC-SHA-256 signature over the exact JSON body bytes, with
// timestamp_ms and nonce inside the signed body. Constants here must
// match the Can side; the live pair test proves they agree.
//
// Secrets (the carrier shared secret, downstream credential values)
// pass through as opaque strings and are never logged, reported, or
// stored by this module.
import { createHmac, randomBytes } from "node:crypto";

// Protocol v1: the only admitted carrier protocol version. Bump with
// the Can-side protocol_valid plus a coordinated companion release.
export const PROTOCOL_VERSION = "1";
// N protocol-versioned attempts: a row claimed at this many attempts
// is poison and the worker dead-letters it without delivering.
export const MAX_ATTEMPTS = 5;
// Timestamp skew window mirror: the Can side admits timestamps within
// +-300000ms of its clock. The companion stamps Date.now(); operators
// keep both clocks within the window.
export const SKEW_MS = 300000;
// Lease bounds mirror: the Can side admits 1000..600000ms lease asks.
export const LEASE_MIN_MS = 1000;
export const LEASE_MAX_MS = 600000;
// Carrier body bound mirror: the Can side caps carrier bodies at 1024
// bytes. Requests built here stay far under it; the check fails closed.
export const CARRIER_BODY_LIMIT = 1024;

export type ClaimItem = Readonly<{
  deliveryId: string;
  event: string;
  subscription: string;
  attempts: number;
  leaseUntilMs: number;
  version: number;
}>;

export type ClaimPage = Readonly<{
  status: string;
  items: readonly ClaimItem[];
}>;

export type HeartbeatResult = Readonly<{
  status: string;
  leaseUntilMs: number;
  version: number;
  attempts: number;
}>;

export type AckResult = Readonly<{
  status: string;
  attempts: number;
}>;

export type DeadResult = Readonly<{
  status: string;
  attempts: number;
}>;

// signBody computes the lowercase hex HMAC-SHA-256 of the exact body
// bytes under the shared secret. The companion signs the serialized
// bytes it sends; the Can side verifies the bytes it received, so any
// tampering with headers, fields, or framing breaks the tag.
export function signBody(secret: string, body: string): string {
  if (secret === "") throw new TypeError("empty carrier secret");
  return createHmac("sha256", secret).update(body, "utf8").digest("hex");
}

// newNonce mints one single-use carrier nonce: 16 random bytes as 24
// hex chars. Replays die on the Can-side nonce table.
export function newNonce(): string {
  return randomBytes(16).toString("hex");
}

// carrierHeaders builds the authenticated envelope headers for one
// carrier body. The protocol version rides beside the signature so a
// future version negotiates explicitly instead of misreading bodies.
export function carrierHeaders(secret: string, body: string): Record<string, string> {
  if (Buffer.byteLength(body, "utf8") > CARRIER_BODY_LIMIT)
    throw new TypeError("carrier body over limit");
  return {
    "content-type": "application/json",
    "x-carrier-protocol": PROTOCOL_VERSION,
    "x-carrier-signature": signBody(secret, body),
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function checkString(field: string, value: unknown): string {
  if (typeof value !== "string") throw new TypeError(`invalid carrier ${field}`);
  return value;
}

function checkInt(field: string, value: unknown): number {
  if (typeof value !== "number" || !Number.isSafeInteger(value))
    throw new TypeError(`invalid carrier ${field}`);
  return value;
}

function checkClaimItem(value: unknown): ClaimItem {
  if (!isRecord(value)) throw new TypeError("invalid carrier claim item");
  return Object.freeze({
    deliveryId: checkString("delivery_id", value["delivery_id"]),
    event: checkString("event", value["event"]),
    subscription: checkString("subscription", value["subscription"]),
    attempts: checkInt("attempts", value["attempts"]),
    leaseUntilMs: checkInt("lease_until_ms", value["lease_until_ms"]),
    version: checkInt("version", value["version"]),
  });
}

// parseClaimPage validates one POST /outbox/claim response body. The
// page holds zero or one item under protocol v1; anything else fails
// closed so a confused or tampered Can side cannot inject phantom rows.
export function parseClaimPage(body: unknown): ClaimPage {
  if (!isRecord(body)) throw new TypeError("invalid carrier claim page");
  const status = checkString("status", body["status"]);
  if (!Array.isArray(body["items"])) throw new TypeError("invalid carrier claim items");
  if (body["items"].length > 1) throw new TypeError("claim page holds more than one item");
  return Object.freeze({ status, items: Object.freeze(body["items"].map(checkClaimItem)) });
}

export function parseHeartbeatResult(body: unknown): HeartbeatResult {
  if (!isRecord(body)) throw new TypeError("invalid carrier heartbeat result");
  return Object.freeze({
    status: checkString("status", body["status"]),
    leaseUntilMs: checkInt("lease_until_ms", body["lease_until_ms"]),
    version: checkInt("version", body["version"]),
    attempts: checkInt("attempts", body["attempts"]),
  });
}

export function parseAckResult(body: unknown): AckResult {
  if (!isRecord(body)) throw new TypeError("invalid carrier ack result");
  return Object.freeze({
    status: checkString("status", body["status"]),
    attempts: checkInt("attempts", body["attempts"]),
  });
}

export function parseDeadResult(body: unknown): DeadResult {
  if (!isRecord(body)) throw new TypeError("invalid carrier dead result");
  return Object.freeze({
    status: checkString("status", body["status"]),
    attempts: checkInt("attempts", body["attempts"]),
  });
}

export type CarrierTransport = (
  path: string,
  headers: Record<string, string>,
  body: string,
) => Promise<{ status: number; json: unknown }>;

// CarrierHttp carries a non-routine carrier HTTP status from the
// transport up to the worker. Routine protocol outcomes ride 200
// (delivered states), 404 (unknown ids) and 409 (lease conflicts);
// every other status throws so the worker can fail fast (400/401/413)
// or back off (500/503) instead of misreading the outcome.
export class CarrierHttp extends Error {
  readonly status: number;
  readonly bodyStatus: string;
  constructor(status: number, bodyStatus: string) {
    super(`carrier http ${status}: ${bodyStatus}`);
    this.name = "CarrierHttp";
    this.status = status;
    this.bodyStatus = bodyStatus;
  }
}

// CarrierTransportError marks a failed round trip to Can (refused,
// reset, timed out at the socket): the request may never have
// arrived, so the worker backs off and retries with a fresh nonce.
// Transports must wrap their native failures in this type; anything
// else thrown (parser violations, bugs) is fatal.
export class CarrierTransportError extends Error {
  constructor() {
    super("carrier transport failed");
    this.name = "CarrierTransportError";
  }
}

export type CarrierClient = Readonly<{
  claim: (workerId: string, leaseMs: number) => Promise<ClaimPage>;
  heartbeat: (
    workerId: string,
    deliveryId: string,
    version: number,
    leaseMs: number,
  ) => Promise<HeartbeatResult>;
  ack: (deliveryId: string, settled: boolean) => Promise<AckResult>;
  deadLetter: (deliveryId: string, workerId: string, reason: string) => Promise<DeadResult>;
}>;

// createCarrierClient binds one worker identity to the Can carrier
// endpoints. Timestamp and nonce are minted per call inside the signed
// body; callers pass op fields only. Protocol-versioned bounds
// (worker length, lease range) are enforced here as well as Can-side,
// so violations fail fast without a round trip.
export function createCarrierClient(
  transport: CarrierTransport,
  secret: string,
  now: () => number = Date.now,
  nonce: () => string = newNonce,
): CarrierClient {
  if (secret === "") throw new TypeError("empty carrier secret");
  async function call<T>(
    path: string,
    fields: Record<string, string | number | boolean>,
    parse: (body: unknown) => T,
  ): Promise<T> {
    const body = JSON.stringify({ ...fields, timestamp_ms: now(), nonce: nonce() });
    const response = await transport(path, carrierHeaders(secret, body), body);
    if (response.status === 200 || response.status === 404 || response.status === 409)
      return parse(response.json);
    let bodyStatus = "unparseable-body";
    try {
      bodyStatus = checkString("status", (response.json as Record<string, unknown>)?.["status"]);
    } catch {
      // Keep the default: the status line already identifies the failure.
    }
    throw new CarrierHttp(response.status, bodyStatus);
  }
  function checkWorker(workerId: string): void {
    if (workerId.length < 1 || workerId.length > 128)
      throw new TypeError("invalid carrier worker id");
  }
  function checkLease(leaseMs: number): void {
    if (!Number.isSafeInteger(leaseMs) || leaseMs < LEASE_MIN_MS || leaseMs > LEASE_MAX_MS)
      throw new TypeError("invalid carrier lease");
  }
  return Object.freeze({
    claim: async (workerId, leaseMs) => {
      checkWorker(workerId);
      checkLease(leaseMs);
      return call("/outbox/claim", { worker_id: workerId, lease_ms: leaseMs }, parseClaimPage);
    },
    heartbeat: async (workerId, deliveryId, version, leaseMs) => {
      checkWorker(workerId);
      checkLease(leaseMs);
      if (!Number.isSafeInteger(version) || version < 0)
        throw new TypeError("invalid carrier version");
      return call(
        "/outbox/heartbeat",
        { worker_id: workerId, delivery_id: deliveryId, version, lease_ms: leaseMs },
        parseHeartbeatResult,
      );
    },
    ack: async (deliveryId, settled) =>
      call("/outbox/ack", { delivery_id: deliveryId, settled }, parseAckResult),
    deadLetter: async (deliveryId, workerId, reason) => {
      checkWorker(workerId);
      if (reason !== "poison" && reason !== "destination_denied" && reason !== "no_destination")
        throw new TypeError("invalid carrier dead-letter reason");
      return call(
        "/outbox/dead_letter",
        { delivery_id: deliveryId, worker_id: workerId, reason },
        parseDeadResult,
      );
    },
  });
}
