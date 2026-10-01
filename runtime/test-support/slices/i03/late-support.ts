// NT-I03 behavioral port of the checkFactsInert closure from
// tools/runtime/test-services/native-values/schema.ts (K01, accepted).
// The runtime ship boundary forbids runtime/ from importing tools/, so
// the I03 slice carries its own copy of exactly what the late service
// needs: outcome/kind vocabularies, NativeSchemaError, the forbidden
// fact-key set, scan bounds, and checkFactsInert. Logic below is
// unchanged from K01; only this header differs.
export const NATIVE_OUTCOMES = [
  "completed",
  "rejected",
  "failed",
  "deadline",
  "indeterminate",
] as const;
export type NativeOutcome = (typeof NATIVE_OUTCOMES)[number];

export const MECHANICAL_KINDS = [
  "ok",
  "invalid-request",
  "unsupported-capability",
  "wrong-owner",
  "stale-handle",
  "closed-handle",
  "not-found",
  "permission",
  "changed-input",
  "kind-collision",
  "resource-limit",
  "native-io",
  "transport-failure",
  "unresolved-cleanup",
] as const;
export type MechanicalKind = (typeof MECHANICAL_KINDS)[number];

export class NativeSchemaError extends Error {
  readonly outcome: NativeOutcome;
  readonly kind: MechanicalKind;
  constructor(outcome: NativeOutcome, kind: MechanicalKind, message: string) {
    super(message);
    this.name = "NativeSchemaError";
    this.outcome = outcome;
    this.kind = kind;
  }
}

export const FORBIDDEN_ARG_KEYS = Object.freeze([
  "module",
  "modulePath",
  "export",
  "exportName",
  "source",
  "code",
  "script",
  "eval",
  "evaluate",
  "expected",
  "expect",
  "expectedValue",
  "oracle",
  "answer",
] as const);

const FORBIDDEN_SET: ReadonlySet<string> = new Set(FORBIDDEN_ARG_KEYS as readonly string[]);

const MAX_SCAN_DEPTH = 8;
const MAX_SCAN_KEYS = 256;

export function checkFactsInert(facts: unknown, what: string): void {
  let seen = 0;
  const visit = (node: unknown, depth: number): void => {
    if (node === null) return;
    const kind = typeof node;
    if (kind === "function" || kind === "symbol" || kind === "bigint") {
      throw new NativeSchemaError("failed", "native-io", `${what} carries a non-inert ${kind}`);
    }
    if (typeof node !== "object") return;
    if (depth > MAX_SCAN_DEPTH) {
      throw new NativeSchemaError("failed", "native-io", `${what} nests too deeply`);
    }
    if (node instanceof Promise) {
      throw new NativeSchemaError("failed", "native-io", `${what} carries a raw Promise`);
    }
    if ("then" in (node as Record<string, unknown>)) {
      throw new NativeSchemaError("failed", "native-io", `${what} carries a thenable-shaped value`);
    }
    if (Array.isArray(node)) {
      for (const item of node) visit(item, depth + 1);
      return;
    }
    const keys = Object.keys(node);
    if (keys.length > 32) {
      throw new NativeSchemaError("failed", "native-io", `${what} exceeds the fact-map bound`);
    }
    for (const key of keys) {
      seen += 1;
      if (seen > MAX_SCAN_KEYS) {
        throw new NativeSchemaError("failed", "native-io", `${what} carries too many keys`);
      }
      if (FORBIDDEN_SET.has(key)) {
        throw new NativeSchemaError("failed", "native-io", `${what} carries forbidden key: ${key}`);
      }
      visit((node as Record<string, unknown>)[key], depth + 1);
    }
  };
  visit(facts, 0);
}
