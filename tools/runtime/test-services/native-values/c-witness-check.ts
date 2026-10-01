// K05 bounded self-check: actual C-entry witness hook and its controls.
//
// Run with: bun tools/runtime/test-services/native-values/c-witness-check.ts
// Local controls only. No services, transports, timers, or live runtimes.
// Every loop below is bounded by a small constant.
//
// Fixtures are real emitter output committed in-repo, pinned by sha256 so any
// drift fails loudly: the executable artifact is byte-exact, and the case
// artifact carries exactly one inserted import line (the runtime/image.ts
// edge the current emitter adds unconditionally; the unrepaired bytes are the
// stale-artifact negative control). Adapters run through the real supervisors
// (runtime/entry.ts runEntry, runtime/assert/runner.ts runAssertion) driven by
// the witness hook; nothing here calls begin/complete (that seam is gone).

import { strict as assert } from "node:assert";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { checkFactsInert, NativeSchemaError } from "./schema.ts";
import {
  C_WITNESS_EXPORTS,
  CAdapterWitness,
  digestBinding,
  MAX_WITNESS_ARTIFACTS,
  MAX_WITNESS_INVOCATIONS,
  NATIVE_C_WITNESS_SCHEMA_VERSION,
  type CWitnessArtifactKind,
} from "./c-witness.ts";
import { success, type Completion } from "../../../../runtime/completion.ts";

const here = dirname(fileURLToPath(import.meta.url));
const manifestSource = readFileSync(join(here, "c-ingress.manifest.json"), "utf8");
const manifest = JSON.parse(manifestSource) as {
  readonly signatures: Readonly<Record<string, string>>;
};

function signatureOf(exportName: string): string {
  const signature = manifest.signatures[exportName];
  assert.ok(typeof signature === "string" && signature.length > 0);
  return signature as string;
}

// --- Real generated-artifact fixtures ----------------------------------------

const EVIDENCE_ROOT = join(
  here,
  "..",
  "..",
  "..",
  "..",
  "docs",
  "syntax-taste",
  "evidence",
  "2026-09-26",
  "full-review-2cb1bc3",
  "core-probes",
  "recurse-small",
);
const EXEC_PATH = "entry.ts";
const EXEC_SOURCE = readFileSync(join(EVIDENCE_ROOT, "prod", "entry.ts"), "utf8");
const CASE_FILENAME = "6abe8e13774b07642245601f17e48741d3a9c7cf581b258e1238c7911615f0fb.ts";
const CASE_PATH = `assertions/${CASE_FILENAME}`;
const CASE_RAW = readFileSync(join(EVIDENCE_ROOT, "assert", "assertions", CASE_FILENAME), "utf8");

function sha256Hex(data: string): string {
  return createHash("sha256").update(data, "utf8").digest("hex");
}

// Pinned bytes: any drift in the committed evidence fails here, not silently.
assert.equal(
  sha256Hex(EXEC_SOURCE),
  "381aae74c0822da5c281cdad8b304cff437481f50831668ec64fbef70f76cea0",
);
assert.equal(
  sha256Hex(CASE_RAW),
  "6a1128ba8f226230855e241a925e424ee04566dc946d2239b29f9e1106aac96f",
);

// The one edge the current emitter adds unconditionally
// (compiler/internal/emit/runtime_core.go:12). The repair is exactly one line:
// the anchor occurs once, so the splice cannot duplicate or drop anything.
const IMAGE_EDGE = 'import { createImage as $canCreateImage } from "../runtime/image.ts";';
const IMAGE_ANCHOR =
  'import { sha256 as $canSHA256 } from "../runtime/platform/crypto/primitives.ts";';
assert.equal(CASE_RAW.split(IMAGE_ANCHOR).length, 2);
const CASE_SOURCE = CASE_RAW.replace(IMAGE_ANCHOR, `${IMAGE_EDGE}\n${IMAGE_ANCHOR}`);
assert.equal(CASE_SOURCE.length, CASE_RAW.length + IMAGE_EDGE.length + 1);

// Independent artifact-digest recomputation: flat preimage, three lines of
// node:crypto, no witness internals.
function artifactDigestOf(path: string, kind: CWitnessArtifactKind, source: string): string {
  return `sha256:${sha256Hex(`${path}\0${kind}\0${source}`)}`;
}

function execArtifact(): { path: string; kind: CWitnessArtifactKind; source: string } {
  return { path: EXEC_PATH, kind: "executable", source: EXEC_SOURCE };
}

function caseArtifact(): { path: string; kind: CWitnessArtifactKind; source: string } {
  return { path: CASE_PATH, kind: "assertion_case", source: CASE_SOURCE };
}

function caseRoot(name: string): {
  package: string;
  declaration: string;
  name: string;
} {
  return { package: "can.k05", declaration: "witness", name };
}

// --- Check harness ------------------------------------------------------------

const passed: string[] = [];
async function check(name: string, body: () => void | Promise<void>): Promise<void> {
  await body();
  passed.push(name);
}

function expectRejectSync(body: () => unknown, kind: string): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof NativeSchemaError)) {
      assert.fail("expected a NativeSchemaError");
    }
    assert.equal(error.outcome, "rejected");
    assert.equal(error.kind, kind);
    return;
  }
  assert.fail(`expected rejection with kind ${kind}`);
}

async function expectRejectAsync(promise: Promise<unknown>, kind: string): Promise<void> {
  try {
    await promise;
  } catch (error) {
    if (!(error instanceof NativeSchemaError)) {
      assert.fail("expected a NativeSchemaError");
    }
    assert.equal(error.outcome, "rejected");
    assert.equal(error.kind, kind);
    return;
  }
  assert.fail(`expected rejection with kind ${kind}`);
}

// --- Positive: actual invocation through the real hook -------------------------

await check("positive-executable-invocation", async () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument(
    execArtifact(),
    "$canMain",
    signatureOf("$canMain"),
    manifest,
  );
  assert.equal(artifact.artifact, "cartifact1");
  assert.ok(artifact.token.startsWith("cwt:1:"));
  assert.equal(artifact.artifact_digest, artifactDigestOf(EXEC_PATH, "executable", EXEC_SOURCE));
  assert.ok(artifact.adapter_binding.startsWith("$canFunction1 from ./packages/"));
  assert.ok(artifact.adapter_binding.endsWith(".ts"));
  checkFactsInert(artifact, "artifact facts");

  let calls = 0;
  const seenArgs: string[][] = [];
  const done = await witness.invokeExecutable(
    artifact.token,
    artifact.artifact,
    artifact.artifact_digest,
    (args: readonly string[]) => {
      calls += 1;
      seenArgs.push([...args]);
      return success(undefined);
    },
    ["alpha"],
  );
  assert.equal(calls, 1);
  assert.deepEqual(seenArgs, [["alpha"]]);
  assert.equal(done.invocation, "cinv1");
  assert.equal(done.completion, "completed");
  assert.equal(done.kind, "executable");
  assert.equal(done.exit_code, 0);
  assert.equal(done.report_lines, 0);
  assert.ok(done.binding.startsWith("sha256:"));
  checkFactsInert(done, "completion facts");

  const seen = witness.observe(
    artifact.token,
    artifact.artifact,
    artifact.artifact_digest,
    done.invocation,
  );
  assert.equal(seen.artifact, artifact.artifact);
  assert.equal(seen.adapter_export, "$canMain");
  assert.equal(seen.adapter_binding, artifact.adapter_binding);
  assert.equal(seen.artifact_digest, artifact.artifact_digest);
  assert.equal(seen.invocation, done.invocation);
  assert.equal(seen.completion, "completed");
  assert.equal(seen.binding, done.binding);
  assert.equal(seen.kind, "executable");
  checkFactsInert(seen, "observation facts");

  assert.deepEqual(witness.counters(), {
    artifacts: 1,
    invocations: 1,
    completions: 1,
    observations: 1,
    rejected: 0,
  });
});

await check("positive-executable-failed", async () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument(
    execArtifact(),
    "$canMain",
    signatureOf("$canMain"),
    manifest,
  );
  let calls = 0;
  const done = await witness.invokeExecutable(
    artifact.token,
    artifact.artifact,
    artifact.artifact_digest,
    async (): Promise<Completion<void>> => {
      calls += 1;
      throw new Error("boom");
    },
  );
  assert.equal(calls, 1);
  assert.equal(done.completion, "failed");
  assert.equal(done.kind, "executable");
  assert.equal(done.exit_code, 1);
  assert.equal(done.report_lines, 1);
  const seen = witness.observe(
    artifact.token,
    artifact.artifact,
    artifact.artifact_digest,
    done.invocation,
  );
  assert.equal(seen.completion, "failed");
  assert.equal(seen.binding, done.binding);
});

await check("positive-binding-pins-hook", async () => {
  const witness = new CAdapterWitness();
  const exec = witness.instrument(execArtifact(), "$canMain", signatureOf("$canMain"), manifest);
  const doneExec = await witness.invokeExecutable(
    exec.token,
    exec.artifact,
    exec.artifact_digest,
    () => success(undefined),
  );
  assert.equal(
    doneExec.binding,
    digestBinding({
      schema_version: NATIVE_C_WITNESS_SCHEMA_VERSION,
      hook: "runtime/entry.ts runEntry",
      artifact: exec.artifact,
      adapter_export: exec.adapter_export,
      adapter_binding: exec.adapter_binding,
      artifact_digest: exec.artifact_digest,
      signature: signatureOf("$canMain"),
      token: exec.token,
      invocation: doneExec.invocation,
      completion: doneExec.completion,
      exit_code: 0,
      report_lines: 0,
    }),
  );

  const kase = witness.instrument(caseArtifact(), "$canCase", signatureOf("$canCase"), manifest);
  assert.equal(kase.adapter_binding, "$canCase");
  assert.equal(kase.artifact_digest, artifactDigestOf(CASE_PATH, "assertion_case", CASE_SOURCE));
  const doneCase = await witness.invokeCase(kase.token, kase.artifact, kase.artifact_digest, {
    root: caseRoot("pins-hook"),
    actual: async () => success(1),
    expected: async () => success(1),
  });
  assert.equal(
    doneCase.binding,
    digestBinding({
      schema_version: NATIVE_C_WITNESS_SCHEMA_VERSION,
      hook: "runtime/assert/runner.ts runAssertion",
      artifact: kase.artifact,
      adapter_export: kase.adapter_export,
      adapter_binding: kase.adapter_binding,
      artifact_digest: kase.artifact_digest,
      signature: signatureOf("$canCase"),
      token: kase.token,
      invocation: doneCase.invocation,
      completion: doneCase.completion,
      passed: true,
      reason: null,
    }),
  );
});

await check("positive-case-pass", async () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument(
    caseArtifact(),
    "$canCase",
    signatureOf("$canCase"),
    manifest,
  );
  const done = await witness.invokeCase(
    artifact.token,
    artifact.artifact,
    artifact.artifact_digest,
    {
      root: caseRoot("pass"),
      actual: async () => success(1),
      expected: async () => success(1),
    },
  );
  assert.equal(done.completion, "completed");
  assert.equal(done.kind, "assertion_case");
  assert.equal(done.passed, true);
  assert.equal(done.reason, null);
  const seen = witness.observe(
    artifact.token,
    artifact.artifact,
    artifact.artifact_digest,
    done.invocation,
  );
  assert.equal(seen.completion, "completed");
  assert.equal(seen.binding, done.binding);
  checkFactsInert(seen, "case observation facts");
});

await check("positive-case-rejected", async () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument(
    caseArtifact(),
    "$canCase",
    signatureOf("$canCase"),
    manifest,
  );
  const done = await witness.invokeCase(
    artifact.token,
    artifact.artifact,
    artifact.artifact_digest,
    {
      root: caseRoot("rejected"),
      actual: async () => success(1),
      expected: async () => {
        throw new Error("expected blew up");
      },
    },
  );
  assert.equal(done.completion, "rejected");
  assert.equal(done.kind, "assertion_case");
  assert.equal(done.passed, false);
  assert.equal(done.reason, "expected evaluation failed");
  const seen = witness.observe(
    artifact.token,
    artifact.artifact,
    artifact.artifact_digest,
    done.invocation,
  );
  assert.equal(seen.completion, "rejected");
  assert.equal(seen.binding, done.binding);
});

await check("positive-case-failed", async () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument(
    caseArtifact(),
    "$canCase",
    signatureOf("$canCase"),
    manifest,
  );
  const mismatch = await witness.invokeCase(
    artifact.token,
    artifact.artifact,
    artifact.artifact_digest,
    {
      root: caseRoot("mismatch"),
      actual: async () => success(1),
      expected: async () => success(2),
    },
  );
  assert.equal(mismatch.completion, "failed");
  assert.equal(mismatch.kind, "assertion_case");
  assert.equal(mismatch.passed, false);
  assert.equal(mismatch.reason, "outcome mismatch");

  const harness = await witness.invokeCase(
    artifact.token,
    artifact.artifact,
    artifact.artifact_digest,
    {
      root: { ...caseRoot("harness"), links: ["never-used"] },
      actual: async () => success(1),
      expected: async () => success(1),
    },
  );
  assert.equal(harness.completion, "failed");
  assert.equal(harness.kind, "assertion_case");
  assert.equal(harness.passed, false);
  assert.equal(harness.reason, "harness violation");
  assert.notEqual(mismatch.binding, harness.binding);
});

await check("positive-bindings-differ-per-invocation", async () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument(
    execArtifact(),
    "$canMain",
    signatureOf("$canMain"),
    manifest,
  );
  const first = await witness.invokeExecutable(
    artifact.token,
    artifact.artifact,
    artifact.artifact_digest,
    () => success(undefined),
  );
  const second = await witness.invokeExecutable(
    artifact.token,
    artifact.artifact,
    artifact.artifact_digest,
    () => success(undefined),
  );
  assert.notEqual(first.invocation, second.invocation);
  assert.notEqual(first.binding, second.binding);
});

await check("token-model-partial", async () => {
  // Preserved token/occurrence mechanics as partial evidence: per-instance
  // secret nonce, table-lookup verification, cross-instance failure.
  const first = new CAdapterWitness();
  const second = new CAdapterWitness();
  const one = first.instrument(execArtifact(), "$canMain", signatureOf("$canMain"), manifest);
  const two = second.instrument(execArtifact(), "$canMain", signatureOf("$canMain"), manifest);
  assert.match(one.token, /^cwt:1:[0-9a-f]{16}$/);
  assert.notEqual(one.token, two.token);
  // Same artifact bytes, same digest, but the token names no binding here.
  assert.equal(one.artifact_digest, two.artifact_digest);
  await expectRejectAsync(
    second.invokeExecutable(one.token, two.artifact, two.artifact_digest, () => success(undefined)),
    "not-found",
  );
  assert.equal(second.counters().invocations, 0);
});

// --- Bypass: uninstrumented artifacts and fabricated tokens -------------------

await check("bypass-uninvoked-adapter", async () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument(
    execArtifact(),
    "$canMain",
    signatureOf("$canMain"),
    manifest,
  );
  // A directly called (unwitnessed) adapter produces no facts: the witness
  // never saw an invocation, so there is nothing to observe.
  let calls = 0;
  const adapter = () => {
    calls += 1;
    return success(undefined);
  };
  adapter();
  assert.equal(calls, 1);
  assert.equal(witness.counters().invocations, 0);
  expectRejectSync(
    () => witness.observe(artifact.token, artifact.artifact, artifact.artifact_digest, "cinv1"),
    "stale-handle",
  );
});

await check("bypass-uninstrumented-digest", async () => {
  const witness = new CAdapterWitness();
  const exec = witness.instrument(execArtifact(), "$canMain", signatureOf("$canMain"), manifest);
  const kase = witness.instrument(caseArtifact(), "$canCase", signatureOf("$canCase"), manifest);
  const unknownDigest = `sha256:${"0".repeat(64)}`;
  // Well-formed but never-minted digest: no binding, on invoke or observe.
  await expectRejectAsync(
    witness.invokeExecutable(exec.token, exec.artifact, unknownDigest, () => success(undefined)),
    "not-found",
  );
  expectRejectSync(
    () => witness.observe(exec.token, exec.artifact, unknownDigest, "cinv1"),
    "not-found",
  );
  // Real digest, wrong artifact: the triple was never minted together.
  await expectRejectAsync(
    witness.invokeExecutable(exec.token, exec.artifact, kase.artifact_digest, () =>
      success(undefined),
    ),
    "not-found",
  );
  await expectRejectAsync(
    witness.invokeCase(kase.token, kase.artifact, exec.artifact_digest, {
      root: caseRoot("swap"),
      actual: async () => success(1),
      expected: async () => success(1),
    }),
    "not-found",
  );
  assert.equal(witness.counters().invocations, 0);
});

await check("bypass-fabricated-token", async () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument(
    execArtifact(),
    "$canMain",
    signatureOf("$canMain"),
    manifest,
  );
  const fabricated = "cwt:1:0123456789abcdef";
  assert.notEqual(fabricated, artifact.token);
  await expectRejectAsync(
    witness.invokeExecutable(fabricated, artifact.artifact, artifact.artifact_digest, () =>
      success(undefined),
    ),
    "not-found",
  );
  expectRejectSync(
    () => witness.observe(fabricated, artifact.artifact, artifact.artifact_digest, "cinv1"),
    "not-found",
  );
});

await check("bypass-token-artifact-mismatch", async () => {
  const witness = new CAdapterWitness();
  const exec = witness.instrument(execArtifact(), "$canMain", signatureOf("$canMain"), manifest);
  const kase = witness.instrument(caseArtifact(), "$canCase", signatureOf("$canCase"), manifest);
  assert.notEqual(exec.token, kase.token);
  await expectRejectAsync(
    witness.invokeExecutable(kase.token, exec.artifact, exec.artifact_digest, () =>
      success(undefined),
    ),
    "not-found",
  );
  expectRejectSync(
    () => witness.observe(kase.token, exec.artifact, exec.artifact_digest, "cinv1"),
    "not-found",
  );
});

await check("bypass-malformed-shapes", async () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument(
    execArtifact(),
    "$canMain",
    signatureOf("$canMain"),
    manifest,
  );
  for (const token of ["", "cwt:0:0123456789abcdef", "cwt:1:xyz", "cartifact1", 42, null]) {
    await expectRejectAsync(
      witness.invokeExecutable(token, artifact.artifact, artifact.artifact_digest, () =>
        success(undefined),
      ),
      "not-found",
    );
  }
  for (const digest of ["", "sha256:xyz", "sha256:0123456789abcdef", 42, null]) {
    await expectRejectAsync(
      witness.invokeExecutable(artifact.token, artifact.artifact, digest, () => success(undefined)),
      "not-found",
    );
  }
  for (const id of ["", "cartifact0", "cinv1", 42, null]) {
    await expectRejectAsync(
      witness.invokeExecutable(artifact.token, id, artifact.artifact_digest, () =>
        success(undefined),
      ),
      "not-found",
    );
  }
  assert.equal(witness.counters().invocations, 0);
});

// --- Wrong export / signature vs the manifest ----------------------------------

await check("wrong-export-outside-vocab", () => {
  const witness = new CAdapterWitness();
  for (const name of ["$canEvil", "main", "runEntry", "runAssertionRoot"]) {
    expectRejectSync(
      () => witness.instrument(execArtifact(), name, signatureOf("$canMain"), manifest),
      "unsupported-capability",
    );
  }
});

await check("wrong-export-kind-mismatch", async () => {
  const witness = new CAdapterWitness();
  // $canMain needs executable bytes; $canCase needs assertion-case bytes.
  expectRejectSync(
    () => witness.instrument(caseArtifact(), "$canMain", signatureOf("$canMain"), manifest),
    "unsupported-capability",
  );
  expectRejectSync(
    () => witness.instrument(execArtifact(), "$canCase", signatureOf("$canCase"), manifest),
    "unsupported-capability",
  );
  // And each hook only drives its own kind.
  const exec = witness.instrument(execArtifact(), "$canMain", signatureOf("$canMain"), manifest);
  const kase = witness.instrument(caseArtifact(), "$canCase", signatureOf("$canCase"), manifest);
  await expectRejectAsync(
    witness.invokeExecutable(kase.token, kase.artifact, kase.artifact_digest, () =>
      success(undefined),
    ),
    "unsupported-capability",
  );
  await expectRejectAsync(
    witness.invokeCase(exec.token, exec.artifact, exec.artifact_digest, {
      root: caseRoot("kind"),
      actual: async () => success(1),
      expected: async () => success(1),
    }),
    "unsupported-capability",
  );
});

await check("wrong-export-missing-from-manifest", () => {
  const witness = new CAdapterWitness();
  const partial = JSON.parse(manifestSource) as { signatures: Record<string, string> };
  delete partial.signatures["$canCase"];
  expectRejectSync(
    () => witness.instrument(caseArtifact(), "$canCase", signatureOf("$canCase"), partial),
    "unsupported-capability",
  );
});

await check("wrong-signature-differs-from-manifest", () => {
  const witness = new CAdapterWitness();
  expectRejectSync(
    () => witness.instrument(execArtifact(), "$canMain", "(bogus) => void", manifest),
    "changed-input",
  );
  for (const bad of ["", "x".repeat(513), 42, null]) {
    expectRejectSync(
      () => witness.instrument(execArtifact(), "$canMain", bad, manifest),
      "invalid-request",
    );
  }
});

await check("wrong-artifact-seam-dirty", () => {
  const witness = new CAdapterWitness();
  const signature = signatureOf("$canMain");
  expectRejectSync(
    () =>
      witness.instrument(
        {
          path: EXEC_PATH,
          kind: "executable",
          source: `${EXEC_SOURCE}export const $canExtra = 1;\n`,
        },
        "$canMain",
        signature,
        manifest,
      ),
    "changed-input",
  );
  expectRejectSync(
    () =>
      witness.instrument(
        {
          path: EXEC_PATH,
          kind: "executable",
          source: `${EXEC_SOURCE}const late = await import("./runtime/evil.ts");\n`,
        },
        "$canMain",
        signature,
        manifest,
      ),
    "changed-input",
  );
  assert.equal(witness.counters().artifacts, 0);
});

await check("wrong-artifact-stale", () => {
  // The committed case bytes predate the manifest's image.ts edge: the real
  // checker rejects them, so the witness must too.
  const witness = new CAdapterWitness();
  try {
    witness.instrument(
      { path: CASE_PATH, kind: "assertion_case", source: CASE_RAW },
      "$canCase",
      signatureOf("$canCase"),
      manifest,
    );
  } catch (error) {
    if (!(error instanceof NativeSchemaError)) {
      assert.fail("expected a NativeSchemaError");
    }
    assert.equal(error.outcome, "rejected");
    assert.equal(error.kind, "changed-input");
    assert.ok(error.message.includes("image.ts"));
    assert.equal(witness.counters().artifacts, 0);
    return;
  }
  assert.fail("expected the stale artifact to reject");
});

// --- Eager reads -----------------------------------------------------------------

await check("eager-read-pending", async () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument(
    execArtifact(),
    "$canMain",
    signatureOf("$canMain"),
    manifest,
  );
  let release!: (completion: Completion<void>) => void;
  const gate = new Promise<Completion<void>>((resolve) => {
    release = resolve;
  });
  const pending = witness.invokeExecutable(
    artifact.token,
    artifact.artifact,
    artifact.artifact_digest,
    () => gate,
  );
  // The invocation is open but unsettled: facts are unbound.
  assert.deepEqual(witness.counters().invocations, 1);
  assert.deepEqual(witness.counters().completions, 0);
  expectRejectSync(
    () => witness.observe(artifact.token, artifact.artifact, artifact.artifact_digest, "cinv1"),
    "permission",
  );
  release(success(undefined));
  const done = await pending;
  assert.equal(done.completion, "completed");
  const seen = witness.observe(
    artifact.token,
    artifact.artifact,
    artifact.artifact_digest,
    done.invocation,
  );
  assert.equal(seen.binding, done.binding);
});

await check("eager-read-unknown-invocation", () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument(
    execArtifact(),
    "$canMain",
    signatureOf("$canMain"),
    manifest,
  );
  for (const id of ["cinv7", "cinv0", "cartifact1", "", 42, null]) {
    expectRejectSync(
      () => witness.observe(artifact.token, artifact.artifact, artifact.artifact_digest, id),
      "stale-handle",
    );
  }
});

// --- Forged completion: no public seam to forge through ----------------------------

await check("forged-no-public-completion", () => {
  // The revoked model seam is gone: invocations open only inside the hook and
  // complete only from observed supervisor settlement.
  const witness = new CAdapterWitness() as unknown as Record<string, unknown>;
  assert.equal(witness["begin"], undefined);
  assert.equal(witness["complete"], undefined);
  const proto = Object.getPrototypeOf(witness) as Record<string, unknown>;
  assert.equal(proto["begin"], undefined);
  assert.equal(proto["complete"], undefined);
});

await check("forged-observe-never-invoked", () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument(
    execArtifact(),
    "$canMain",
    signatureOf("$canMain"),
    manifest,
  );
  expectRejectSync(
    () => witness.observe(artifact.token, artifact.artifact, artifact.artifact_digest, "cinv1"),
    "stale-handle",
  );
});

await check("forged-observe-wrong-artifact", async () => {
  const witness = new CAdapterWitness();
  const exec = witness.instrument(execArtifact(), "$canMain", signatureOf("$canMain"), manifest);
  const kase = witness.instrument(caseArtifact(), "$canCase", signatureOf("$canCase"), manifest);
  const done = await witness.invokeExecutable(exec.token, exec.artifact, exec.artifact_digest, () =>
    success(undefined),
  );
  // The invocation belongs to the executable artifact, not the case one.
  expectRejectSync(
    () => witness.observe(kase.token, kase.artifact, kase.artifact_digest, done.invocation),
    "stale-handle",
  );
});

// --- Bounds ----------------------------------------------------------------------

await check("bounds-artifact-cap", () => {
  const witness = new CAdapterWitness();
  for (let index = 0; index < MAX_WITNESS_ARTIFACTS; index += 1) {
    const exportName = C_WITNESS_EXPORTS[index % C_WITNESS_EXPORTS.length] as string;
    const artifact = exportName === "$canMain" ? execArtifact() : caseArtifact();
    witness.instrument(artifact, exportName, signatureOf(exportName), manifest);
  }
  expectRejectSync(
    () => witness.instrument(execArtifact(), "$canMain", signatureOf("$canMain"), manifest),
    "resource-limit",
  );
  assert.equal(witness.counters().artifacts, MAX_WITNESS_ARTIFACTS);
});

await check("bounds-invocation-cap", async () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument(
    execArtifact(),
    "$canMain",
    signatureOf("$canMain"),
    manifest,
  );
  for (let index = 0; index < MAX_WITNESS_INVOCATIONS; index += 1) {
    await witness.invokeExecutable(
      artifact.token,
      artifact.artifact,
      artifact.artifact_digest,
      () => success(undefined),
    );
  }
  await expectRejectAsync(
    witness.invokeExecutable(artifact.token, artifact.artifact, artifact.artifact_digest, () =>
      success(undefined),
    ),
    "resource-limit",
  );
  assert.equal(witness.counters().invocations, MAX_WITNESS_INVOCATIONS);
});

await check("bounds-malformed-manifest", () => {
  const witness = new CAdapterWitness();
  const signature = signatureOf("$canMain");
  for (const bad of [null, [], "manifest", {}, { signatures: null }, { signatures: [] }]) {
    expectRejectSync(
      () => witness.instrument(execArtifact(), "$canMain", signature, bad),
      "invalid-request",
    );
  }
  const broken = JSON.parse(manifestSource) as {
    entries: { executable: { body_pattern: string } };
  };
  broken.entries.executable.body_pattern = "(unclosed";
  expectRejectSync(
    () => witness.instrument(execArtifact(), "$canMain", signature, broken),
    "invalid-request",
  );
  assert.equal(witness.counters().artifacts, 0);
});

await check("bounds-input-limits", async () => {
  const witness = new CAdapterWitness();
  const signature = signatureOf("$canMain");
  expectRejectSync(
    () =>
      witness.instrument(
        { path: EXEC_PATH, kind: "executable", source: "x".repeat(1048577) },
        "$canMain",
        signature,
        manifest,
      ),
    "resource-limit",
  );
  expectRejectSync(
    () =>
      witness.instrument(
        { path: "x".repeat(257), kind: "executable", source: EXEC_SOURCE },
        "$canMain",
        signature,
        manifest,
      ),
    "invalid-request",
  );
  expectRejectSync(
    () =>
      witness.instrument(
        { path: EXEC_PATH, kind: "nope", source: EXEC_SOURCE },
        "$canMain",
        signature,
        manifest,
      ),
    "unsupported-capability",
  );

  const artifact = witness.instrument(execArtifact(), "$canMain", signature, manifest);
  const triple = [artifact.token, artifact.artifact, artifact.artifact_digest] as const;
  await expectRejectAsync(
    witness.invokeExecutable(triple[0], triple[1], triple[2], "not-a-function"),
    "invalid-request",
  );
  await expectRejectAsync(
    witness.invokeExecutable(triple[0], triple[1], triple[2], () => success(undefined), "nope"),
    "invalid-request",
  );
  await expectRejectAsync(
    witness.invokeExecutable(
      triple[0],
      triple[1],
      triple[2],
      () => success(undefined),
      Array.from({ length: 9 }, (_, index) => `arg${index}`),
    ),
    "resource-limit",
  );
  await expectRejectAsync(
    witness.invokeExecutable(triple[0], triple[1], triple[2], () => success(undefined), [
      "x".repeat(257),
    ]),
    "invalid-request",
  );

  const kase = witness.instrument(caseArtifact(), "$canCase", signatureOf("$canCase"), manifest);
  const caseTriple = [kase.token, kase.artifact, kase.artifact_digest] as const;
  const goodActual = async () => success(1);
  const goodExpected = async () => success(1);
  for (const root of [
    null,
    { package: "p", declaration: "d" },
    { package: "", declaration: "d", name: "n" },
    { package: "p", declaration: "d", name: "n", links: ["a", "a"] },
    {
      package: "p",
      declaration: "d",
      name: "n",
      links: ["x", "x", "x", "x", "x", "x", "x", "x", "x"],
    },
  ]) {
    await expectRejectAsync(
      witness.invokeCase(caseTriple[0], caseTriple[1], caseTriple[2], {
        root,
        actual: goodActual,
        expected: goodExpected,
      }),
      "invalid-request",
    );
  }
  await expectRejectAsync(
    witness.invokeCase(caseTriple[0], caseTriple[1], caseTriple[2], {
      root: caseRoot("no-actual"),
      expected: goodExpected,
    }),
    "invalid-request",
  );
  await expectRejectAsync(
    witness.invokeCase(caseTriple[0], caseTriple[1], caseTriple[2], {
      root: caseRoot("no-expected"),
      actual: goodActual,
    }),
    "invalid-request",
  );
  assert.equal(witness.counters().invocations, 0);
});

await check("bounds-rejections-counted", async () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument(
    execArtifact(),
    "$canMain",
    signatureOf("$canMain"),
    manifest,
  );
  expectRejectSync(
    () => witness.observe(artifact.token, artifact.artifact, artifact.artifact_digest, "cinv9"),
    "stale-handle",
  );
  await expectRejectAsync(
    witness.invokeExecutable(
      "cwt:1:0123456789abcdef",
      artifact.artifact,
      artifact.artifact_digest,
      () => success(undefined),
    ),
    "not-found",
  );
  assert.equal(witness.counters().rejected, 2);
  checkFactsInert(witness.counters(), "witness counters");
});

console.log(
  JSON.stringify({
    schema: "can.native-test.c-witness-check",
    passed: passed.length,
    checks: passed,
  }),
);
