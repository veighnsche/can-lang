// X-R01-1 (D01) UNREVIEWED PROTOTYPE — companion tier, Op A.
//
// Persistent client key-value storage through a typed companion data
// boundary instead of in-process native calls. The client validates
// with the same budgets as the adapter tier, admits the destination
// through F01's C-G policy, attaches F01's identity vocabulary, and
// strict-decodes the response; the companion stub authenticates,
// enforces the same policy version, and answers from its own store.
// The injected transport is synchronous for determinism, but the
// client surface is async: the network boundary turns sync ops async,
// which is the structural cost this prototype measures.
import {
  credentialValue,
  destinationPolicy,
  evaluateDestination,
  policyReport,
  type DestinationPolicy,
  type EnvName,
} from "../../../runtime/outbound/destination-policy.ts";
import {
  correlationId,
  invocationContext,
  poolId,
  tenantId,
  type CorrelationId,
  type InvocationContext,
} from "../../../runtime/outbound/identity.ts";

export const STORAGE_PROTOCOL = "d01.storage/1";
export const STORAGE_KEY_MAX_CHARS = 256;
export const STORAGE_VALUE_MAX_CHARS = 1048576;

export type CompanionStorageFailureCode =
  | "storage::invalid_key"
  | "storage::quota_exceeded"
  | "storage::unavailable"
  | "companion::destination_denied"
  | "companion::unauthorized"
  | "companion::version_mismatch"
  | "companion::transport_failed"
  | "companion::protocol_error";

export type CompanionStorageFailure = Readonly<{ code: CompanionStorageFailureCode; detail: string }>;

export type CompanionStorageResult<T> =
  | Readonly<{ ok: true; value: T }>
  | Readonly<{ ok: false; failure: CompanionStorageFailure }>;

function ok<T>(value: T): CompanionStorageResult<T> {
  return Object.freeze({ ok: true, value }) as CompanionStorageResult<T>;
}

function fail<T>(code: CompanionStorageFailureCode, detail: string): CompanionStorageResult<T> {
  return Object.freeze({ ok: false, failure: Object.freeze({ code, detail }) }) as CompanionStorageResult<T>;
}

export type StorageOp = "get" | "set" | "remove";

export type StorageRequest = Readonly<{
  v: string;
  tenant: string;
  pool: string;
  correlation: string;
  op: StorageOp;
  key: string;
  value?: string;
}>;

export type StorageResponse = Readonly<
  | { v: string; correlation: string; ok: true; value: string | null }
  | { v: string; correlation: string; ok: false; code: string; detail: string }
>;

const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder();

function checkKey(key: unknown): string | null {
  return typeof key === "string" && key.length > 0 && key.length <= STORAGE_KEY_MAX_CHARS && key.isWellFormed()
    ? key
    : null;
}

function checkValue(value: unknown): string | null {
  return typeof value === "string" && value.isWellFormed() && value.length <= STORAGE_VALUE_MAX_CHARS ? value : null;
}

export function encodeStorageRequest(request: StorageRequest): Uint8Array {
  return textEncoder.encode(JSON.stringify(request));
}

export function decodeStorageRequest(bytes: Uint8Array): StorageRequest | CompanionStorageFailure {
  let parsed: unknown;
  try {
    parsed = JSON.parse(textDecoder.decode(bytes));
  } catch {
    return { code: "companion::protocol_error", detail: "request is not well-formed JSON" };
  }
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    return { code: "companion::protocol_error", detail: "request must be an object" };
  }
  const shape = parsed as Record<string, unknown>;
  if (shape.v !== STORAGE_PROTOCOL) return { code: "companion::version_mismatch", detail: "unsupported protocol version" };
  if (shape.op !== "get" && shape.op !== "set" && shape.op !== "remove") {
    return { code: "companion::protocol_error", detail: "unknown storage op" };
  }
  try {
    tenantId(shape.tenant);
    poolId(shape.pool);
    correlationId(shape.correlation);
  } catch {
    return { code: "companion::protocol_error", detail: "invalid identity" };
  }
  const key = checkKey(shape.key);
  if (key === null) return { code: "storage::invalid_key", detail: "key must be well-formed text of 1..256 chars" };
  if (shape.op === "set") {
    const value = checkValue(shape.value);
    if (value === null) {
      return { code: "storage::invalid_key", detail: "value must be well-formed text of at most 1048576 chars" };
    }
    return { v: STORAGE_PROTOCOL, tenant: String(shape.tenant), pool: String(shape.pool), correlation: String(shape.correlation), op: shape.op, key, value };
  }
  return { v: STORAGE_PROTOCOL, tenant: String(shape.tenant), pool: String(shape.pool), correlation: String(shape.correlation), op: shape.op, key };
}

export function encodeStorageResponse(response: StorageResponse): Uint8Array {
  return textEncoder.encode(JSON.stringify(response));
}

export function decodeStorageResponse(bytes: Uint8Array): StorageResponse | CompanionStorageFailure {
  let parsed: unknown;
  try {
    parsed = JSON.parse(textDecoder.decode(bytes));
  } catch {
    return { code: "companion::protocol_error", detail: "response is not well-formed JSON" };
  }
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    return { code: "companion::protocol_error", detail: "response must be an object" };
  }
  const shape = parsed as Record<string, unknown>;
  if (shape.v !== STORAGE_PROTOCOL) return { code: "companion::version_mismatch", detail: "unsupported protocol version" };
  if (typeof shape.correlation !== "string") return { code: "companion::protocol_error", detail: "response lacks correlation" };
  if (shape.ok === true) {
    if (typeof shape.value !== "string" && shape.value !== null) {
      return { code: "companion::protocol_error", detail: "response value must be text or null" };
    }
    return { v: STORAGE_PROTOCOL, correlation: shape.correlation, ok: true, value: shape.value };
  }
  if (shape.ok === false && typeof shape.code === "string" && typeof shape.detail === "string") {
    return { v: STORAGE_PROTOCOL, correlation: shape.correlation, ok: false, code: shape.code, detail: shape.detail };
  }
  return { code: "companion::protocol_error", detail: "response shape unknown" };
}

export type CompanionTransport = {
  send(url: string, authorization: string, body: Uint8Array): Uint8Array;
};

export type StorageCompanionClient = Readonly<{
  localGet: (key: unknown) => Promise<CompanionStorageResult<string | null>>;
  localSet: (key: unknown, value: unknown) => Promise<CompanionStorageResult<null>>;
  localRemove: (key: unknown) => Promise<CompanionStorageResult<null>>;
}>;

export function createStorageCompanionClient(init: Readonly<{
  policy: DestinationPolicy;
  endpoint: string;
  context: InvocationContext;
  credentialEnv: EnvName;
  readEnvironment: (key: string) => string | undefined;
  transport: CompanionTransport;
}>): StorageCompanionClient {
  const roundtrip = async (op: StorageOp, key: string, value?: string): Promise<CompanionStorageResult<string | null>> => {
    const decision = evaluateDestination(init.policy, init.endpoint);
    if (decision.decision !== "allowed") return fail("companion::destination_denied", `destination denied: ${decision.reason}`);
    const secret = credentialValue(init.credentialEnv, init.readEnvironment);
    if (secret === undefined) return fail("companion::unauthorized", "credential unavailable");
    const body = encodeStorageRequest({
      v: STORAGE_PROTOCOL,
      tenant: init.context.tenant,
      pool: init.context.pool,
      correlation: init.context.correlation,
      op,
      key,
      ...(value === undefined ? {} : { value }),
    });
    let raw: Uint8Array;
    try {
      raw = init.transport.send(init.endpoint, `Bearer ${secret}`, body);
    } catch {
      return fail("companion::transport_failed", "companion unreachable");
    }
    const decoded = decodeStorageResponse(raw);
    if ("code" in decoded) return fail(decoded.code as CompanionStorageFailureCode, decoded.detail);
    if (decoded.correlation !== init.context.correlation) {
      return fail("companion::protocol_error", "response correlation mismatch");
    }
    if (!decoded.ok) {
      const code = ((): CompanionStorageFailureCode => {
        switch (decoded.code) {
          case "storage::quota_exceeded":
          case "storage::unavailable":
          case "storage::invalid_key":
            return decoded.code;
          case "companion::unauthorized":
            return "companion::unauthorized";
          case "companion::version_mismatch":
            return "companion::version_mismatch";
          default:
            return "companion::protocol_error";
        }
      })();
      return fail(code, decoded.detail);
    }
    return ok(decoded.value);
  };
  const gate = (key: unknown, value?: unknown): string | CompanionStorageFailure => {
    const checkedKey = checkKey(key);
    if (checkedKey === null) return { code: "storage::invalid_key", detail: "key must be well-formed text of 1..256 chars" };
    if (value !== undefined && checkValue(value) === null) {
      return { code: "storage::invalid_key", detail: "value must be well-formed text of at most 1048576 chars" };
    }
    return checkedKey;
  };
  return Object.freeze({
    async localGet(key: unknown): Promise<CompanionStorageResult<string | null>> {
      const checked = gate(key);
      if (typeof checked !== "string") return fail(checked.code, checked.detail);
      return roundtrip("get", checked);
    },
    async localSet(key: unknown, value: unknown): Promise<CompanionStorageResult<null>> {
      const checked = gate(key, value);
      if (typeof checked !== "string") return fail(checked.code, checked.detail);
      const result = await roundtrip("set", checked, value as string);
      if (!result.ok) return fail(result.failure.code, result.failure.detail);
      return ok(null);
    },
    async localRemove(key: unknown): Promise<CompanionStorageResult<null>> {
      const checked = gate(key);
      if (typeof checked !== "string") return fail(checked.code, checked.detail);
      const result = await roundtrip("remove", checked);
      if (!result.ok) return fail(result.failure.code, result.failure.detail);
      return ok(null);
    },
  });
}

// compareSecret is a length-checked accumulating comparison. It is not a
// certified constant-time primitive; the real pair (F05) owns that claim
// and its honest limitation. The prototype's guarantee is narrower: the
// secret value never enters a report, decision, or diagnostic string.
function compareSecret(presented: string, expected: string): boolean {
  if (presented.length !== expected.length) return false;
  let diff = 0;
  for (let index = 0; index < expected.length; index += 1) {
    diff |= presented.charCodeAt(index) ^ expected.charCodeAt(index);
  }
  return diff === 0;
}

export type StorageCompanionServer = Readonly<{
  policyVersion: string;
  handle: (authorization: string | undefined, body: Uint8Array) => Uint8Array;
  diagnostics: () => readonly string[];
}>;

export function createStorageCompanionServer(init: Readonly<{
  policy: DestinationPolicy;
  credentialEnv: EnvName;
  readEnvironment: (key: string) => string | undefined;
  quotaChars?: number;
}>): StorageCompanionServer {
  const store = new Map<string, string>();
  const quota = init.quotaChars ?? 1024;
  const log: string[] = [];
  const report = policyReport(init.policy);
  const answer = (response: StorageResponse): Uint8Array => encodeStorageResponse(response);
  return Object.freeze({
    policyVersion: report.version,
    handle(authorization: string | undefined, body: Uint8Array): Uint8Array {
      const decoded = decodeStorageRequest(body);
      if ("code" in decoded) {
        log.push(`reject:${decoded.code}`);
        const correlation = (() => {
          try {
            const parsed = JSON.parse(textDecoder.decode(body)) as { correlation?: unknown };
            return typeof parsed.correlation === "string" ? parsed.correlation : "unknown";
          } catch {
            return "unknown";
          }
        })();
        return answer({ v: STORAGE_PROTOCOL, correlation, ok: false, code: decoded.code, detail: decoded.detail });
      }
      const expected = credentialValue(init.credentialEnv, init.readEnvironment);
      const presented = authorization !== undefined && authorization.startsWith("Bearer ") ? authorization.slice(7) : "";
      if (expected === undefined || !compareSecret(presented, expected)) {
        log.push(`reject:companion::unauthorized:${decoded.tenant}:${decoded.correlation}`);
        return answer({ v: STORAGE_PROTOCOL, correlation: decoded.correlation, ok: false, code: "companion::unauthorized", detail: "bad credential" });
      }
      log.push(`accept:${decoded.op}:${decoded.tenant}:${decoded.correlation}`);
      if (decoded.op === "get") return answer({ v: STORAGE_PROTOCOL, correlation: decoded.correlation, ok: true, value: store.has(decoded.key) ? store.get(decoded.key)! : null });
      if (decoded.op === "remove") {
        store.delete(decoded.key);
        return answer({ v: STORAGE_PROTOCOL, correlation: decoded.correlation, ok: true, value: null });
      }
      const value = decoded.value!;
      const current = store.has(decoded.key) ? store.get(decoded.key)!.length : 0;
      let total = 0;
      for (const entry of store.values()) total += entry.length;
      if (total - current + value.length > quota) {
        return answer({ v: STORAGE_PROTOCOL, correlation: decoded.correlation, ok: false, code: "storage::quota_exceeded", detail: "companion quota exceeded" });
      }
      store.set(decoded.key, value);
      return answer({ v: STORAGE_PROTOCOL, correlation: decoded.correlation, ok: true, value: null });
    },
    diagnostics(): readonly string[] {
      return Object.freeze([...log]);
    },
  });
}

export function storageCompanionPolicy(): DestinationPolicy {
  return destinationPolicy({
    version: "2026-09-26.d01-storage",
    rules: [{ scheme: "https", host: "companion.example", pathPrefix: "/v1/", credential: "D01_STORAGE_COMPANION_TOKEN" }],
    redirect: "deny",
    maxRedirectHops: 0,
    loopback: "deny",
    privateNetworks: "deny",
  });
}

export function storageCompanionContext(correlation: string): InvocationContext {
  return invocationContext("tenant-acme", "pool-default", correlation as CorrelationId);
}
