// K05: independent C adapter witness.
//
// Proves a C adapter was ACTUALLY invoked through the generated path instead
// of bypassed. Generated-path calls carry a witness token minted here; the
// witness verifies the token, checks the claimed export/signature against the
// K04 ingress manifest, and only then binds inert facts to the invocation:
//   - bypass: an uninstrumented artifact id or an unknown token has no
//     binding, so begin/complete/observe reject (negative control: a locally
//     fabricated token string never verifies);
//   - wrong export/signature: an export outside the closed vocabulary, an
//     export missing from the manifest, or a signature that differs from the
//     manifest's entry rejects at instrumentation time;
//   - eager read: observing facts for an invocation that has begun but not
//     completed rejects as unbound;
//   - forged local completion: completing an invocation id with no matching
//     open witnessed invocation (or a replayed completion) rejects.
//
// Returned facts are inert counter/identity data only (counts, artifact ids,
// binding digests; never raw values) and pass checkFactsInert.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, or live runtimes. Local controls only.

import { createHash, randomBytes } from "node:crypto";
import { checkFactsInert, NativeSchemaError } from "./schema.ts";

export const NATIVE_C_WITNESS_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Closed vocabularies
// ---------------------------------------------------------------------------

// Adapter exports the witness can bind. Each must also be present in the K04
// manifest signatures table with an exactly matching signature string.
export const C_WITNESS_EXPORTS = ["$canMain", "$canCase", "runEntry", "runAssertionRoot"] as const;
export type CWitnessExport = (typeof C_WITNESS_EXPORTS)[number];

// Terminal completion tags for one witnessed invocation.
export const C_WITNESS_COMPLETIONS = ["completed", "rejected", "failed"] as const;
export type CWitnessCompletion = (typeof C_WITNESS_COMPLETIONS)[number];

function isWitnessExport(value: unknown): value is CWitnessExport {
  return typeof value === "string" && (C_WITNESS_EXPORTS as readonly string[]).includes(value);
}

function isWitnessCompletion(value: unknown): value is CWitnessCompletion {
  return typeof value === "string" && (C_WITNESS_COMPLETIONS as readonly string[]).includes(value);
}

// ---------------------------------------------------------------------------
// Manifest view (K04 c-ingress.manifest.json signatures table)
// ---------------------------------------------------------------------------

export type CWitnessManifestView = Readonly<{
  signatures: Readonly<Record<string, string>>;
}>;

export function checkManifestView(value: unknown): CWitnessManifestView {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new NativeSchemaError(
      "rejected",
      "invalid-request",
      "witness manifest must be an object",
    );
  }
  const signatures = (value as Record<string, unknown>)["signatures"];
  if (signatures === null || typeof signatures !== "object" || Array.isArray(signatures)) {
    throw new NativeSchemaError(
      "rejected",
      "invalid-request",
      "witness manifest needs a signatures table",
    );
  }
  for (const [key, entry] of Object.entries(signatures)) {
    if (typeof entry !== "string" || entry.length === 0 || entry.length > 512) {
      throw new NativeSchemaError(
        "rejected",
        "invalid-request",
        `invalid manifest signature for ${key}`,
      );
    }
  }
  return value as CWitnessManifestView;
}

// ---------------------------------------------------------------------------
// Bounded caps and id shapes
// ---------------------------------------------------------------------------

export const MAX_WITNESS_ARTIFACTS = 8;
export const MAX_WITNESS_INVOCATIONS = 32;

const ARTIFACT_PATTERN = /^cartifact[1-9][0-9]*$/;
const INVOCATION_PATTERN = /^cinv[1-9][0-9]*$/;
const TOKEN_PATTERN = /^cwt:[1-9][0-9]*:[0-9a-f]{16}$/;

// ---------------------------------------------------------------------------
// Binding digest (local, deterministic; independent of the K03 seal)
// ---------------------------------------------------------------------------

function canonicalize(value: unknown): string {
  if (value === null) return "null";
  if (typeof value === "number" || typeof value === "string" || typeof value === "boolean") {
    return JSON.stringify(value) as string;
  }
  if (Array.isArray(value)) {
    return `[${value.map((item) => canonicalize(item)).join(",")}]`;
  }
  if (typeof value === "object") {
    const record = value as Record<string, unknown>;
    const keys = Object.keys(record).sort();
    const body = keys.map((key) => `${JSON.stringify(key)}:${canonicalize(record[key])}`).join(",");
    return `{${body}}`;
  }
  throw new NativeSchemaError("failed", "native-io", "witness input carries a non-inert value");
}

export function digestBinding(value: unknown): string {
  return `sha256:${createHash("sha256").update(canonicalize(value), "utf8").digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Fact shapes (all inert data)
// ---------------------------------------------------------------------------

export type WitnessArtifactFacts = Readonly<{
  artifact: string;
  adapter_export: CWitnessExport;
  token: string;
  seq: number;
}>;

export type WitnessInvocationFacts = Readonly<{
  artifact: string;
  adapter_export: CWitnessExport;
  invocation: string;
  seq: number;
}>;

export type WitnessCompletionFacts = Readonly<{
  artifact: string;
  invocation: string;
  completion: CWitnessCompletion;
  binding: string;
}>;

export type WitnessObservationFacts = Readonly<{
  artifact: string;
  adapter_export: CWitnessExport;
  invocation: string;
  completion: CWitnessCompletion;
  binding: string;
}>;

export type WitnessCounters = Readonly<{
  artifacts: number;
  invocations: number;
  completions: number;
  observations: number;
  rejected: number;
}>;

// ---------------------------------------------------------------------------
// Witness: tokens bind invocations to instrumented artifacts
// ---------------------------------------------------------------------------

type ArtifactRecord = {
  artifactId: string;
  token: string;
  exportName: CWitnessExport;
  signature: string;
  seq: number;
};

type InvocationRecord = {
  invocationId: string;
  artifactId: string;
  completion: CWitnessCompletion | null;
  binding: string | null;
};

export class CAdapterWitness {
  private readonly nonce: string;
  private readonly artifacts = new Map<string, ArtifactRecord>();
  private readonly byToken = new Map<string, ArtifactRecord>();
  private readonly invocations = new Map<string, InvocationRecord>();
  private artifactCounter = 0;
  private invocationCounter = 0;
  private completionCount = 0;
  private observationCount = 0;
  private rejectedCount = 0;

  constructor() {
    this.nonce = randomBytes(8).toString("hex");
  }

  counters(): WitnessCounters {
    return Object.freeze({
      artifacts: this.artifacts.size,
      invocations: this.invocations.size,
      completions: this.completionCount,
      observations: this.observationCount,
      rejected: this.rejectedCount,
    });
  }

  // Instrument one adapter artifact: the export must be in the closed
  // vocabulary and the claimed signature must exactly match the manifest's
  // entry. Returns the artifact id plus its witness token; only tokens
  // minted here ever verify.
  instrument(exportName: unknown, signature: unknown, manifest: unknown): WitnessArtifactFacts {
    const checked = checkManifestView(manifest);
    if (!isWitnessExport(exportName)) {
      throw this.reject("unsupported-capability", `unknown witness export: ${String(exportName)}`);
    }
    if (typeof signature !== "string" || signature.length === 0 || signature.length > 512) {
      throw this.reject("invalid-request", "witness signature must be a bounded string");
    }
    const authoritative = checked.signatures[exportName];
    if (authoritative === undefined) {
      throw this.reject(
        "unsupported-capability",
        `export ${exportName} is not in the ingress manifest`,
      );
    }
    if (signature !== authoritative) {
      throw this.reject("changed-input", `signature for ${exportName} differs from the manifest`);
    }
    if (this.artifacts.size >= MAX_WITNESS_ARTIFACTS) {
      throw this.reject("resource-limit", "witness artifact cap reached");
    }
    this.artifactCounter += 1;
    const artifactId = `cartifact${this.artifactCounter}`;
    const token = this.mintToken(this.artifactCounter);
    const record: ArtifactRecord = {
      artifactId,
      token,
      exportName,
      signature,
      seq: this.artifactCounter,
    };
    this.artifacts.set(artifactId, record);
    this.byToken.set(token, record);
    const facts = Object.freeze({
      artifact: artifactId,
      adapter_export: exportName,
      token,
      seq: this.artifactCounter,
    });
    checkFactsInert(facts, "witness artifact facts");
    return facts;
  }

  // Begin one generated-path invocation. The token must verify and name the
  // claimed artifact; uninstrumented artifact ids and fabricated tokens have
  // no binding and reject, which is the bypass control.
  begin(token: unknown, artifactId: unknown): WitnessInvocationFacts {
    const record = this.requireBinding(token, artifactId);
    if (this.invocations.size >= MAX_WITNESS_INVOCATIONS) {
      throw this.reject("resource-limit", "witness invocation cap reached");
    }
    this.invocationCounter += 1;
    const invocationId = `cinv${this.invocationCounter}`;
    this.invocations.set(invocationId, {
      invocationId,
      artifactId: record.artifactId,
      completion: null,
      binding: null,
    });
    const facts = Object.freeze({
      artifact: record.artifactId,
      adapter_export: record.exportName,
      invocation: invocationId,
      seq: this.invocationCounter,
    });
    checkFactsInert(facts, "witness invocation facts");
    return facts;
  }

  // Complete one open witnessed invocation. A completion tag outside the
  // closed vocabulary, an invocation id with no matching open witnessed
  // invocation, or a replayed completion rejects as forged.
  complete(
    token: unknown,
    artifactId: unknown,
    invocationId: unknown,
    completion: unknown,
  ): WitnessCompletionFacts {
    const record = this.requireBinding(token, artifactId);
    if (!isWitnessCompletion(completion)) {
      throw this.reject("unsupported-capability", `unknown completion tag: ${String(completion)}`);
    }
    if (typeof invocationId !== "string" || !INVOCATION_PATTERN.test(invocationId)) {
      throw this.reject("stale-handle", "completion names no witnessed invocation");
    }
    const invocation = this.invocations.get(invocationId);
    if (invocation === undefined || invocation.artifactId !== record.artifactId) {
      throw this.reject("stale-handle", "completion has no matching witnessed invocation");
    }
    if (invocation.completion !== null) {
      throw this.reject("stale-handle", "invocation is already complete");
    }
    invocation.completion = completion;
    const binding = digestBinding({
      schema_version: NATIVE_C_WITNESS_SCHEMA_VERSION,
      artifact: record.artifactId,
      adapter_export: record.exportName,
      signature: record.signature,
      token: record.token,
      invocation: invocationId,
      completion,
    });
    invocation.binding = binding;
    this.completionCount += 1;
    const facts = Object.freeze({
      artifact: record.artifactId,
      invocation: invocationId,
      completion,
      binding,
    });
    checkFactsInert(facts, "witness completion facts");
    return facts;
  }

  // Observe the inert facts bound to one invocation. The invocation must be
  // complete: reading an open invocation rejects as an eager (unbound) read,
  // and reading an unknown invocation rejects as forged.
  observe(token: unknown, artifactId: unknown, invocationId: unknown): WitnessObservationFacts {
    const record = this.requireBinding(token, artifactId);
    if (typeof invocationId !== "string" || !INVOCATION_PATTERN.test(invocationId)) {
      throw this.reject("stale-handle", "observation names no witnessed invocation");
    }
    const invocation = this.invocations.get(invocationId);
    if (invocation === undefined || invocation.artifactId !== record.artifactId) {
      throw this.reject("stale-handle", "observation has no matching witnessed invocation");
    }
    if (invocation.completion === null || invocation.binding === null) {
      throw this.reject("permission", "invocation is not complete; facts are unbound");
    }
    this.observationCount += 1;
    const facts = Object.freeze({
      artifact: record.artifactId,
      adapter_export: record.exportName,
      invocation: invocationId,
      completion: invocation.completion,
      binding: invocation.binding,
    });
    checkFactsInert(facts, "witness observation facts");
    return facts;
  }

  private mintToken(seq: number): string {
    const digest = createHash("sha256").update(`${this.nonce}:${seq}`, "utf8").digest("hex");
    return `cwt:${seq}:${digest.slice(0, 16)}`;
  }

  // Verify the (token, artifact) binding. Unknown tokens, malformed tokens,
  // unknown artifacts, and token/artifact mismatches all reject: none of
  // them proves a generated-path call.
  private requireBinding(token: unknown, artifactId: unknown): ArtifactRecord {
    if (typeof token !== "string" || !TOKEN_PATTERN.test(token)) {
      throw this.reject("not-found", "no instrumented artifact for this token");
    }
    if (typeof artifactId !== "string" || !ARTIFACT_PATTERN.test(artifactId)) {
      throw this.reject("not-found", "no instrumented artifact for this id");
    }
    const record = this.byToken.get(token);
    if (record === undefined || record.artifactId !== artifactId) {
      throw this.reject("not-found", "token names no instrumented artifact");
    }
    return record;
  }

  private reject(
    kind:
      | "invalid-request"
      | "unsupported-capability"
      | "changed-input"
      | "not-found"
      | "permission"
      | "stale-handle"
      | "resource-limit",
    message: string,
  ): NativeSchemaError {
    this.rejectedCount += 1;
    return new NativeSchemaError("rejected", kind, message);
  }
}
