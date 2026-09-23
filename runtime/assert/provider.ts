import {
  contextOwner,
  fixtureIndex,
  registerFixtureTable,
  rawProviderEvidence,
  violation,
  type AssertionContext,
} from "./context.ts";
import type { FailureOrigin } from "../failure.ts";
import { transportFault } from "../transport/deadline.ts";
import { parseDocument } from "../codec/document.ts";
import { CodecIssue } from "../codec/budget.ts";
import { ownBytes } from "../bytes.ts";
import { Buffer } from "node:buffer";

// Compiler conformance data only: no authored Can constructor or public input
// accepts these rows. Bytes are copied at registration and at consumption.
export type RawHTTPFixture = Readonly<{
  request: Readonly<{
    method: string;
    url: string;
    headers: readonly (readonly [string, string])[];
    body: Uint8Array;
  }>;
  response: Readonly<{
    status: number;
    headers: readonly (readonly [string, string])[];
    body: Uint8Array;
  }>;
}>;
export type HTTPExchange = (url: URL, init: RequestInit) => Promise<Response>;
const fixtures = new WeakMap<object, readonly RawHTTPFixture[]>();
const origin: FailureOrigin = Object.freeze({
  source: "can:raw-provider",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
export function provideHTTP(context: AssertionContext, rows: readonly RawHTTPFixture[]): void {
  if (fixtures.has(contextOwner(context))) throw violation(context, "malformed fixture", origin);
  const headers = (entries: readonly (readonly [string, string])[]) =>
    Object.freeze(entries.map(([name, value]) => Object.freeze([name, value] as const)));
  let copied: readonly RawHTTPFixture[];
  try {
    copied = Object.freeze(
      rows.map((row) =>
        Object.freeze({
          request: Object.freeze({
            ...row.request,
            headers: headers(row.request.headers),
            body: new Uint8Array(row.request.body),
          }),
          response: Object.freeze({
            ...row.response,
            headers: headers(row.response.headers),
            body: new Uint8Array(row.response.body),
          }),
        }),
      ),
    );
    for (const row of copied) {
      new Headers(row.request.headers.map(([name, value]) => [name, value]));
      fixtureResponse(row.request.method, row.response);
    }
  } catch {
    throw violation(context, "malformed fixture", origin);
  }
  fixtures.set(contextOwner(context), copied);
  registerFixtureTable(context, "can:raw-http", rows.length, origin);
}
export function providerHTTP(
  context: AssertionContext | undefined,
  where: FailureOrigin,
  operation?: string,
  maxBodyBytes?: number,
): HTTPExchange | undefined {
  if (context === undefined) return undefined;
  if (operation !== undefined) {
    const raw = rawExchange(context, where, operation, maxBodyBytes);
    if (raw !== undefined) return raw;
  }
  const rows = fixtures.get(contextOwner(context));
  if (rows === undefined) return undefined;
  return async (url, init) => {
    const row = rows[fixtureIndex(context, "can:raw-http", rows.length, where)];
    const actualHeaders = Array.from(new Headers(init.headers).entries());
    const expectedHeaders = Array.from(
      new Headers(row.request.headers.map(([name, value]) => [name, value])).entries(),
    );
    const body = init.body === undefined ? new Uint8Array() : init.body;
    if (
      init.method !== row.request.method ||
      url.href !== row.request.url ||
      JSON.stringify(actualHeaders) !== JSON.stringify(expectedHeaders) ||
      !(body instanceof Uint8Array) ||
      body.length !== row.request.body.length ||
      !body.every((byte, i) => byte === row.request.body[i])
    )
      throw violation(context, "argument mismatch", where);
    const response = fixtureResponse(row.request.method, row.response);
    rawProviderEvidence(context);
    return response;
  };
}

// Authored P4.1 raw fixtures: one owned exchange per operation plus fake
// credential environment. The compiler validates can.native-fixture.v1
// strictly; these runtime checks are defense in depth. Bytes are copied at
// registration and at consumption.
export type RawFixtureBody = Readonly<{ bytes_base64: string } | { json_utf8: string }>;
export type RawFixtureFailure = Readonly<
  | { kind: "transport"; phase: "connect" | "body" | "protocol" | "cancelled" }
  | { kind: "timeout" }
  | { kind: "body_limit" }
>;
export type RawFixtureInput = Readonly<{
  environment: Readonly<Record<string, string>>;
  exchange: null | Readonly<{
    request: Readonly<{
      method: string;
      url: string;
      headers: readonly (readonly [string, string])[];
      body: RawFixtureBody;
    }>;
    outcome: Readonly<
      | {
          response: Readonly<{
            status: number;
            headers: readonly (readonly [string, string])[];
            body_base64: string;
          }>;
        }
      | { failure: RawFixtureFailure }
    >;
  }>;
}>;
type StoredRaw = Readonly<{
  environment: Readonly<Record<string, string>>;
  exchange: null | Readonly<{
    request: Readonly<{
      method: string;
      url: string;
      headers: readonly (readonly [string, string])[];
      body: Readonly<{ bytes: Uint8Array } | { json: string }>;
    }>;
    outcome: Readonly<
      | {
          response: Readonly<{
            status: number;
            headers: readonly (readonly [string, string])[];
            body: Uint8Array;
          }>;
        }
      | { failure: RawFixtureFailure }
    >;
  }>;
}>;
const rawFixtures = new WeakMap<object, Map<string, StoredRaw>>();
const encoder = new TextEncoder();
function decodeBase64(value: string): Uint8Array {
  const bytes = Buffer.from(value, "base64");
  if ((bytes.length === 0 && value !== "") || Buffer.from(bytes).toString("base64") !== value)
    throw new TypeError("invalid fixture base64");
  return new Uint8Array(bytes);
}
export function provideRawHTTP(
  context: AssertionContext,
  operation: string,
  spec: RawFixtureInput,
): void {
  let owned = rawFixtures.get(contextOwner(context));
  if (owned?.has(operation)) throw violation(context, "malformed fixture", origin);
  let stored: StoredRaw;
  try {
    if (
      spec === null ||
      typeof spec !== "object" ||
      spec.environment === null ||
      typeof spec.environment !== "object" ||
      Array.isArray(spec.environment)
    )
      throw new TypeError("invalid fixture environment");
    const environment = Object.freeze({ ...spec.environment });
    for (const value of Object.values(environment))
      if (typeof value !== "string") throw new TypeError("invalid fixture environment");
    let exchange: StoredRaw["exchange"] = null;
    if (spec.exchange !== null) {
      if (spec.exchange === null || typeof spec.exchange !== "object")
        throw new TypeError("invalid fixture exchange");
      const request = spec.exchange.request,
        outcome = spec.exchange.outcome;
      if (typeof request.method !== "string" || typeof request.url !== "string")
        throw new TypeError("invalid fixture request");
      const headers = Object.freeze(
        request.headers.map(([name, value]) => {
          if (typeof name !== "string" || typeof value !== "string")
            throw new TypeError("invalid fixture header");
          return Object.freeze([name, value] as const);
        }),
      );
      new Headers(headers.map(([name, value]) => [name, value]));
      const body =
        "bytes_base64" in request.body && "json_utf8" in request.body
          ? null
          : "bytes_base64" in request.body
            ? { bytes: decodeBase64(request.body.bytes_base64) }
            : "json_utf8" in request.body
              ? { json: request.body.json_utf8 }
              : null;
      if (body === null || ("json" in body && typeof body.json !== "string"))
        throw new TypeError("invalid fixture body");
      if (
        ("response" in outcome && "failure" in outcome) ||
        (!("response" in outcome) && !("failure" in outcome))
      )
        throw new TypeError("invalid fixture outcome");
      if ("response" in outcome) {
        const response = Object.freeze({
          status: outcome.response.status,
          headers: Object.freeze(
            outcome.response.headers.map(([name, value]) => {
              if (typeof name !== "string" || typeof value !== "string")
                throw new TypeError("invalid fixture header");
              return Object.freeze([name, value] as const);
            }),
          ),
          body: decodeBase64(outcome.response.body_base64),
        });
        fixtureResponse(request.method, response);
        exchange = Object.freeze({
          request: Object.freeze({
            method: request.method,
            url: request.url,
            headers,
            body: Object.freeze(body),
          }),
          outcome: Object.freeze({ response }),
        });
      } else {
        const failure = outcome.failure;
        if (
          (failure.kind === "transport" &&
            !["connect", "body", "protocol", "cancelled"].includes(failure.phase)) ||
          (failure.kind !== "transport" &&
            failure.kind !== "timeout" &&
            failure.kind !== "body_limit")
        )
          throw new TypeError("invalid fixture failure");
        exchange = Object.freeze({
          request: Object.freeze({
            method: request.method,
            url: request.url,
            headers,
            body: Object.freeze(body),
          }),
          outcome: Object.freeze({ failure: Object.freeze({ ...failure }) }),
        });
      }
    }
    stored = Object.freeze({ environment, exchange });
  } catch (cause) {
    if (cause instanceof TypeError) throw violation(context, "malformed fixture", origin);
    throw cause;
  }
  if (!owned) {
    owned = new Map();
    rawFixtures.set(contextOwner(context), owned);
  }
  owned.set(operation, stored);
  registerFixtureTable(context, "can:raw:" + operation, stored.exchange ? 1 : 0, origin);
}
export function rawEnvironment(
  context: AssertionContext | undefined,
  operation: string,
): ((name: string) => string | undefined) | undefined {
  if (context === undefined) return undefined;
  const spec = rawFixtures.get(contextOwner(context))?.get(operation);
  if (spec === undefined) return undefined;
  return (name) => (Object.hasOwn(spec.environment, name) ? spec.environment[name] : undefined);
}
function rawExchange(
  context: AssertionContext,
  where: FailureOrigin,
  operation: string,
  maxBodyBytes?: number,
): HTTPExchange | undefined {
  const spec = rawFixtures.get(contextOwner(context))?.get(operation);
  if (spec === undefined) return undefined;
  if (spec.exchange === null)
    return async () => {
      throw violation(context, "unexpected live boundary", where);
    };
  if (maxBodyBytes === undefined) throw new TypeError("raw fixture requires its connection limit");
  const row = spec.exchange;
  return async (url, init) => {
    fixtureIndex(context, "can:raw:" + operation, 1, where);
    const actualHeaders = Array.from(new Headers(init.headers).entries());
    const expectedHeaders = Array.from(
      new Headers(row.request.headers.map(([name, value]) => [name, value])).entries(),
    );
    const body = init.body === undefined ? new Uint8Array() : init.body;
    let match =
      init.method === row.request.method &&
      url.href === row.request.url &&
      JSON.stringify(actualHeaders) === JSON.stringify(expectedHeaders) &&
      body instanceof Uint8Array;
    if (match && body instanceof Uint8Array) {
      if ("bytes" in row.request.body) {
        const want = row.request.body.bytes;
        match = body.length === want.length && body.every((byte, i) => byte === want[i]);
      } else match = jsonTokensEqual(context, row.request.body.json, body, where);
    }
    if (!match) throw violation(context, "argument mismatch", where);
    rawProviderEvidence(context);
    if ("failure" in row.outcome) {
      const failure = row.outcome.failure;
      if (failure.kind === "transport")
        throw transportFault({ kind: "transport", phase: failure.phase });
      if (failure.kind === "timeout") throw transportFault({ kind: "timeout" });
      throw transportFault({ kind: "limit", limit: maxBodyBytes });
    }
    return fixtureResponse(row.request.method, row.outcome.response);
  };
}
function tryParseDocument(
  bytes: Uint8Array,
):
  | { parsed: unknown; rootHolder: object; tokens: WeakMap<object, Map<string, string>> }
  | undefined {
  try {
    return parseDocument(ownBytes(bytes));
  } catch (cause) {
    if (cause instanceof CodecIssue) return undefined;
    throw cause;
  }
}
function jsonTokensEqual(
  context: AssertionContext,
  expected: string,
  actual: Uint8Array,
  where: FailureOrigin,
): boolean {
  const want = tryParseDocument(encoder.encode(expected));
  if (want === undefined) throw violation(context, "malformed fixture", where);
  const got = tryParseDocument(actual);
  if (got === undefined) return false;
  const spell = (
    doc: { tokens: WeakMap<object, Map<string, string>> },
    holder: object,
    key: string,
  ) => doc.tokens.get(holder)?.get(key);
  const compare = (
    left: unknown,
    leftHolder: object,
    leftKey: string,
    right: unknown,
    rightHolder: object,
    rightKey: string,
  ): boolean => {
    if (typeof left !== typeof right) return false;
    if (typeof left === "number")
      return (
        spell(want, leftHolder, leftKey) !== undefined &&
        spell(want, leftHolder, leftKey) === spell(got, rightHolder, rightKey)
      );
    if (left === null || right === null || typeof left !== "object") return Object.is(left, right);
    if (Array.isArray(left) !== Array.isArray(right)) return false;
    if (Array.isArray(left) && Array.isArray(right))
      return (
        left.length === right.length &&
        left.every((item, index) =>
          compare(
            item,
            left as object,
            String(index),
            right[index],
            right as object,
            String(index),
          ),
        )
      );
    const a = left as Record<string, unknown>,
      b = right as Record<string, unknown>;
    const keys = Object.keys(a);
    return (
      keys.length === Object.keys(b).length &&
      keys.every(
        (key) =>
          Object.hasOwn(b, key) &&
          compare(a[key], left as object, key, b[key], right as object, key),
      )
    );
  };
  return compare(want.parsed, want.rootHolder, "", got.parsed, got.rootHolder, "");
}

function fixtureResponse(method: string, response: RawHTTPFixture["response"]): Response {
  if (!Number.isInteger(response.status) || response.status < 200 || response.status > 599)
    throw new TypeError("invalid fixture status");
  const bodyless = method.toUpperCase() === "HEAD" || [204, 205, 304].includes(response.status);
  if (bodyless && response.body.length !== 0) throw new TypeError("invalid fixture body");
  return new Response(bodyless ? null : new Uint8Array(response.body), {
    status: response.status,
    headers: response.headers.map(([name, value]) => [name, value]),
  });
}
