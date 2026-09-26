// X-R01-1 (D01) UNREVIEWED PROTOTYPE — companion tier, Op B.
//
// Clipboard text exchange through a typed companion data boundary. The
// honest server shape for device-bound text is a pastebin cell: the
// companion stores server-side text the client posts and serves it back
// on read. It CANNOT reach the device clipboard — the server
// constructor takes no device handle, and the gap leg pins that the
// device cell and the companion cell are disjoint by construction.
// Whether that semantic gap excludes T3 for Op B is the discrimination
// question; the prototype exists to make the gap measurable rather
// than assumed. (The only satisfiable T3 shape for device text is a
// served isolated frame plus postMessage boundary; unbuilt — see the
// record for trip conditions.)
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

export const CLIPBOARD_PROTOCOL = "d01.clipboard/1";
export const CLIPBOARD_TEXT_MAX_CHARS = 1048576;

export type CompanionClipboardFailureCode =
  | "clipboard::invalid_text"
  | "clipboard::denied"
  | "clipboard::empty"
  | "clipboard::unavailable"
  | "companion::destination_denied"
  | "companion::unauthorized"
  | "companion::version_mismatch"
  | "companion::transport_failed"
  | "companion::protocol_error";

export type CompanionClipboardFailure = Readonly<{ code: CompanionClipboardFailureCode; detail: string }>;

export type CompanionClipboardResult<T> =
  | Readonly<{ ok: true; value: T }>
  | Readonly<{ ok: false; failure: CompanionClipboardFailure }>;

function ok<T>(value: T): CompanionClipboardResult<T> {
  return Object.freeze({ ok: true, value }) as CompanionClipboardResult<T>;
}

function fail<T>(code: CompanionClipboardFailureCode, detail: string): CompanionClipboardResult<T> {
  return Object.freeze({ ok: false, failure: Object.freeze({ code, detail }) }) as CompanionClipboardResult<T>;
}

export type ClipboardOp = "read" | "write";

export type ClipboardRequest = Readonly<{
  v: string;
  tenant: string;
  pool: string;
  correlation: string;
  op: ClipboardOp;
  text?: string;
}>;

export type ClipboardResponse = Readonly<
  | { v: string; correlation: string; ok: true; text: string | null }
  | { v: string; correlation: string; ok: false; code: string; detail: string }
>;

const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder();

function checkText(value: unknown): string | null {
  return typeof value === "string" && value.length > 0 && value.isWellFormed() && value.length <= CLIPBOARD_TEXT_MAX_CHARS
    ? value
    : null;
}

export function encodeClipboardRequest(request: ClipboardRequest): Uint8Array {
  return textEncoder.encode(JSON.stringify(request));
}

export function decodeClipboardRequest(bytes: Uint8Array): ClipboardRequest | CompanionClipboardFailure {
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
  if (shape.v !== CLIPBOARD_PROTOCOL) return { code: "companion::version_mismatch", detail: "unsupported protocol version" };
  if (shape.op !== "read" && shape.op !== "write") {
    return { code: "companion::protocol_error", detail: "unknown clipboard op" };
  }
  try {
    tenantId(shape.tenant);
    poolId(shape.pool);
    correlationId(shape.correlation);
  } catch {
    return { code: "companion::protocol_error", detail: "invalid identity" };
  }
  if (shape.op === "write") {
    const text = checkText(shape.text);
    if (text === null) {
      return { code: "clipboard::invalid_text", detail: "text must be well-formed non-empty text of at most 1048576 chars" };
    }
    return { v: CLIPBOARD_PROTOCOL, tenant: String(shape.tenant), pool: String(shape.pool), correlation: String(shape.correlation), op: shape.op, text };
  }
  return { v: CLIPBOARD_PROTOCOL, tenant: String(shape.tenant), pool: String(shape.pool), correlation: String(shape.correlation), op: shape.op };
}

export function encodeClipboardResponse(response: ClipboardResponse): Uint8Array {
  return textEncoder.encode(JSON.stringify(response));
}

export function decodeClipboardResponse(bytes: Uint8Array): ClipboardResponse | CompanionClipboardFailure {
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
  if (shape.v !== CLIPBOARD_PROTOCOL) return { code: "companion::version_mismatch", detail: "unsupported protocol version" };
  if (typeof shape.correlation !== "string") return { code: "companion::protocol_error", detail: "response lacks correlation" };
  if (shape.ok === true) {
    if (typeof shape.text !== "string" && shape.text !== null) {
      return { code: "companion::protocol_error", detail: "response text must be text or null" };
    }
    return { v: CLIPBOARD_PROTOCOL, correlation: shape.correlation, ok: true, text: shape.text };
  }
  if (shape.ok === false && typeof shape.code === "string" && typeof shape.detail === "string") {
    return { v: CLIPBOARD_PROTOCOL, correlation: shape.correlation, ok: false, code: shape.code, detail: shape.detail };
  }
  return { code: "companion::protocol_error", detail: "response shape unknown" };
}

export type CompanionTransport = {
  send(url: string, authorization: string, body: Uint8Array): Uint8Array;
};

export type ClipboardCompanionClient = Readonly<{
  readText: () => Promise<CompanionClipboardResult<string>>;
  writeText: (text: unknown) => Promise<CompanionClipboardResult<null>>;
}>;

export function createClipboardCompanionClient(init: Readonly<{
  policy: DestinationPolicy;
  endpoint: string;
  context: InvocationContext;
  credentialEnv: EnvName;
  readEnvironment: (key: string) => string | undefined;
  transport: CompanionTransport;
}>): ClipboardCompanionClient {
  const roundtrip = async (op: ClipboardOp, text?: string): Promise<CompanionClipboardResult<string | null>> => {
    const decision = evaluateDestination(init.policy, init.endpoint);
    if (decision.decision !== "allowed") return fail("companion::destination_denied", `destination denied: ${decision.reason}`);
    const secret = credentialValue(init.credentialEnv, init.readEnvironment);
    if (secret === undefined) return fail("companion::unauthorized", "credential unavailable");
    const body = encodeClipboardRequest({
      v: CLIPBOARD_PROTOCOL,
      tenant: init.context.tenant,
      pool: init.context.pool,
      correlation: init.context.correlation,
      op,
      ...(text === undefined ? {} : { text }),
    });
    let raw: Uint8Array;
    try {
      raw = init.transport.send(init.endpoint, `Bearer ${secret}`, body);
    } catch {
      return fail("companion::transport_failed", "companion unreachable");
    }
    const decoded = decodeClipboardResponse(raw);
    if ("code" in decoded) return fail(decoded.code as CompanionClipboardFailureCode, decoded.detail);
    if (decoded.correlation !== init.context.correlation) {
      return fail("companion::protocol_error", "response correlation mismatch");
    }
    if (!decoded.ok) {
      const code = ((): CompanionClipboardFailureCode => {
        switch (decoded.code) {
          case "clipboard::invalid_text":
          case "clipboard::denied":
          case "clipboard::empty":
          case "clipboard::unavailable":
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
    return ok(decoded.text);
  };
  return Object.freeze({
    async readText(): Promise<CompanionClipboardResult<string>> {
      const result = await roundtrip("read");
      if (!result.ok) return fail(result.failure.code, result.failure.detail);
      if (result.value === null) return fail("clipboard::empty", "companion holds no text");
      return ok(result.value);
    },
    async writeText(text: unknown): Promise<CompanionClipboardResult<null>> {
      const checked = checkText(text);
      if (checked === null) {
        return fail("clipboard::invalid_text", "text must be well-formed non-empty text of at most 1048576 chars");
      }
      const result = await roundtrip("write", checked);
      if (!result.ok) return fail(result.failure.code, result.failure.detail);
      return ok(null);
    },
  });
}

function compareSecret(presented: string, expected: string): boolean {
  if (presented.length !== expected.length) return false;
  let diff = 0;
  for (let index = 0; index < expected.length; index += 1) {
    diff |= presented.charCodeAt(index) ^ expected.charCodeAt(index);
  }
  return diff === 0;
}

export type ClipboardCompanionServer = Readonly<{
  policyVersion: string;
  handle: (authorization: string | undefined, body: Uint8Array) => Uint8Array;
  diagnostics: () => readonly string[];
}>;

export function createClipboardCompanionServer(init: Readonly<{
  policy: DestinationPolicy;
  credentialEnv: EnvName;
  readEnvironment: (key: string) => string | undefined;
}>): ClipboardCompanionServer {
  // Server-side pastebin cell. There is deliberately no device handle:
  // the server cannot observe or mutate the device clipboard.
  let cell: string | null = null;
  const log: string[] = [];
  const report = policyReport(init.policy);
  return Object.freeze({
    policyVersion: report.version,
    handle(authorization: string | undefined, body: Uint8Array): Uint8Array {
      const decoded = decodeClipboardRequest(body);
      if ("code" in decoded) {
        log.push(`reject:${decoded.code}`);
        let correlation = "unknown";
        try {
          const parsed = JSON.parse(textDecoder.decode(body)) as { correlation?: unknown };
          if (typeof parsed.correlation === "string") correlation = parsed.correlation;
        } catch {
          correlation = "unknown";
        }
        return encodeClipboardResponse({ v: CLIPBOARD_PROTOCOL, correlation, ok: false, code: decoded.code, detail: decoded.detail });
      }
      const expected = credentialValue(init.credentialEnv, init.readEnvironment);
      const presented = authorization !== undefined && authorization.startsWith("Bearer ") ? authorization.slice(7) : "";
      if (expected === undefined || !compareSecret(presented, expected)) {
        log.push(`reject:companion::unauthorized:${decoded.tenant}:${decoded.correlation}`);
        return encodeClipboardResponse({ v: CLIPBOARD_PROTOCOL, correlation: decoded.correlation, ok: false, code: "companion::unauthorized", detail: "bad credential" });
      }
      log.push(`accept:${decoded.op}:${decoded.tenant}:${decoded.correlation}`);
      if (decoded.op === "read") {
        return encodeClipboardResponse({ v: CLIPBOARD_PROTOCOL, correlation: decoded.correlation, ok: true, text: cell });
      }
      cell = decoded.text!;
      return encodeClipboardResponse({ v: CLIPBOARD_PROTOCOL, correlation: decoded.correlation, ok: true, text: null });
    },
    diagnostics(): readonly string[] {
      return Object.freeze([...log]);
    },
  });
}

export function clipboardCompanionPolicy(): DestinationPolicy {
  return destinationPolicy({
    version: "2026-09-26.d01-clipboard",
    rules: [{ scheme: "https", host: "companion.example", pathPrefix: "/v1/", credential: "D01_CLIPBOARD_COMPANION_TOKEN" }],
    redirect: "deny",
    maxRedirectHops: 0,
    loopback: "deny",
    privateNetworks: "deny",
  });
}

export function clipboardCompanionContext(correlation: string): InvocationContext {
  return invocationContext("tenant-acme", "pool-default", correlation as CorrelationId);
}
