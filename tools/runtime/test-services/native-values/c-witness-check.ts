// K05 bounded self-check: independent C adapter witness mechanics only.
//
// Run with: bun tools/runtime/test-services/native-values/c-witness-check.ts
// Local controls only. No services, transports, timers, or live runtimes.
// Every loop below is bounded by a small constant.

import { strict as assert } from "node:assert";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { checkFactsInert, NativeSchemaError } from "./schema.ts";
import {
  C_WITNESS_EXPORTS,
  CAdapterWitness,
  checkManifestView,
  MAX_WITNESS_ARTIFACTS,
  MAX_WITNESS_INVOCATIONS,
  type CWitnessCompletion,
  type CWitnessManifestView,
} from "./c-witness.ts";

const here = dirname(fileURLToPath(import.meta.url));
const manifest = checkManifestView(
  JSON.parse(readFileSync(join(here, "c-ingress.manifest.json"), "utf8")) as CWitnessManifestView,
);

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectReject(body: () => unknown, kind: string): void {
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

function signatureOf(exportName: string): string {
  const signature = manifest.signatures[exportName];
  assert.ok(typeof signature === "string" && signature.length > 0);
  return signature as string;
}

// --- Positive control: genuine witnessed invocation passes ------------------

check("positive-witnessed-invocation", () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument("$canMain", signatureOf("$canMain"), manifest);
  assert.equal(artifact.artifact, "cartifact1");
  assert.ok(artifact.token.startsWith("cwt:1:"));
  checkFactsInert(artifact, "artifact facts");

  const begun = witness.begin(artifact.token, artifact.artifact);
  assert.equal(begun.invocation, "cinv1");
  checkFactsInert(begun, "invocation facts");

  const done = witness.complete(artifact.token, artifact.artifact, begun.invocation, "completed");
  assert.ok(done.binding.startsWith("sha256:"));
  checkFactsInert(done, "completion facts");

  const seen = witness.observe(artifact.token, artifact.artifact, begun.invocation);
  assert.equal(seen.artifact, artifact.artifact);
  assert.equal(seen.adapter_export, "$canMain");
  assert.equal(seen.invocation, begun.invocation);
  assert.equal(seen.completion, "completed");
  assert.equal(seen.binding, done.binding);
  checkFactsInert(seen, "observation facts");

  assert.deepEqual(witness.counters(), {
    artifacts: 1,
    invocations: 1,
    completions: 1,
    observations: 1,
    rejected: 0,
  });
});

check("positive-every-export-and-completion", () => {
  const completions: CWitnessCompletion[] = ["completed", "rejected", "failed"];
  C_WITNESS_EXPORTS.forEach((exportName, index) => {
    const witness = new CAdapterWitness();
    const completion = completions[index % completions.length] as CWitnessCompletion;
    const artifact = witness.instrument(exportName, signatureOf(exportName), manifest);
    const begun = witness.begin(artifact.token, artifact.artifact);
    const done = witness.complete(artifact.token, artifact.artifact, begun.invocation, completion);
    const seen = witness.observe(artifact.token, artifact.artifact, begun.invocation);
    assert.equal(seen.completion, completion);
    assert.equal(seen.binding, done.binding);
  });
});

check("positive-bindings-differ-per-invocation", () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument("$canCase", signatureOf("$canCase"), manifest);
  const first = witness.begin(artifact.token, artifact.artifact);
  const second = witness.begin(artifact.token, artifact.artifact);
  const doneFirst = witness.complete(
    artifact.token,
    artifact.artifact,
    first.invocation,
    "completed",
  );
  const doneSecond = witness.complete(
    artifact.token,
    artifact.artifact,
    second.invocation,
    "completed",
  );
  assert.notEqual(doneFirst.binding, doneSecond.binding);
});

// --- Bypass: uninstrumented artifacts and fabricated tokens ------------------

check("bypass-fabricated-token", () => {
  const witness = new CAdapterWitness();
  expectReject(() => witness.begin("cwt:1:0123456789abcdef", "cartifact1"), "not-found");
  expectReject(
    () => witness.complete("cwt:1:0123456789abcdef", "cartifact1", "cinv1", "completed"),
    "not-found",
  );
  expectReject(() => witness.observe("cwt:1:0123456789abcdef", "cartifact1", "cinv1"), "not-found");
});

check("bypass-uninstrumented-artifact", () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument("$canMain", signatureOf("$canMain"), manifest);
  // Real token, wrong artifact id: the pair is not a minted binding.
  expectReject(() => witness.begin(artifact.token, "cartifact2"), "not-found");
  // Well-formed but never-minted artifact id with a fabricated token.
  expectReject(() => witness.begin("cwt:9:0123456789abcdef", "cartifact9"), "not-found");
});

check("bypass-token-artifact-mismatch", () => {
  const witness = new CAdapterWitness();
  const first = witness.instrument("$canMain", signatureOf("$canMain"), manifest);
  const second = witness.instrument("$canCase", signatureOf("$canCase"), manifest);
  assert.notEqual(first.token, second.token);
  expectReject(() => witness.begin(first.token, second.artifact), "not-found");
  expectReject(() => witness.begin(second.token, first.artifact), "not-found");
});

check("bypass-malformed-token-shape", () => {
  const witness = new CAdapterWitness();
  for (const token of ["", "cwt:0:0123456789abcdef", "cwt:1:xyz", "cartifact1", 42, null]) {
    expectReject(() => witness.begin(token, "cartifact1"), "not-found");
  }
});

// --- Wrong export / signature -------------------------------------------------

check("wrong-export-outside-vocab", () => {
  const witness = new CAdapterWitness();
  expectReject(
    () => witness.instrument("$canEvil", signatureOf("$canMain"), manifest),
    "unsupported-capability",
  );
  expectReject(() => witness.instrument("main", "anything", manifest), "unsupported-capability");
});

check("wrong-export-missing-from-manifest", () => {
  const witness = new CAdapterWitness();
  const partial = checkManifestView({ signatures: { $canMain: signatureOf("$canMain") } });
  expectReject(
    () => witness.instrument("$canCase", signatureOf("$canCase"), partial),
    "unsupported-capability",
  );
});

check("wrong-signature-differs-from-manifest", () => {
  const witness = new CAdapterWitness();
  expectReject(() => witness.instrument("$canMain", "(bogus) => void", manifest), "changed-input");
  expectReject(() => witness.instrument("$canMain", "", manifest), "invalid-request");
});

// --- Eager reads ---------------------------------------------------------------

check("eager-read-before-completion", () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument("$canMain", signatureOf("$canMain"), manifest);
  const begun = witness.begin(artifact.token, artifact.artifact);
  expectReject(
    () => witness.observe(artifact.token, artifact.artifact, begun.invocation),
    "permission",
  );
  // The same invocation binds facts once it completes.
  const done = witness.complete(artifact.token, artifact.artifact, begun.invocation, "completed");
  const seen = witness.observe(artifact.token, artifact.artifact, begun.invocation);
  assert.equal(seen.binding, done.binding);
});

check("eager-read-unknown-invocation", () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument("$canMain", signatureOf("$canMain"), manifest);
  expectReject(() => witness.observe(artifact.token, artifact.artifact, "cinv7"), "stale-handle");
  expectReject(() => witness.observe(artifact.token, artifact.artifact, "cinv0"), "stale-handle");
});

// --- Forged local completion ----------------------------------------------------

check("forged-completion-never-begun", () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument("$canMain", signatureOf("$canMain"), manifest);
  expectReject(
    () => witness.complete(artifact.token, artifact.artifact, "cinv1", "completed"),
    "stale-handle",
  );
});

check("forged-completion-wrong-artifact", () => {
  const witness = new CAdapterWitness();
  const first = witness.instrument("$canMain", signatureOf("$canMain"), manifest);
  const second = witness.instrument("$canCase", signatureOf("$canCase"), manifest);
  const begun = witness.begin(first.token, first.artifact);
  expectReject(
    () => witness.complete(second.token, second.artifact, begun.invocation, "completed"),
    "stale-handle",
  );
});

check("forged-completion-replay", () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument("$canMain", signatureOf("$canMain"), manifest);
  const begun = witness.begin(artifact.token, artifact.artifact);
  witness.complete(artifact.token, artifact.artifact, begun.invocation, "completed");
  expectReject(
    () => witness.complete(artifact.token, artifact.artifact, begun.invocation, "failed"),
    "stale-handle",
  );
});

check("forged-completion-unknown-tag", () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument("$canMain", signatureOf("$canMain"), manifest);
  const begun = witness.begin(artifact.token, artifact.artifact);
  expectReject(
    () => witness.complete(artifact.token, artifact.artifact, begun.invocation, "succeeded"),
    "unsupported-capability",
  );
});

// --- Bounds ----------------------------------------------------------------------

check("bounds-artifact-cap", () => {
  const witness = new CAdapterWitness();
  for (let index = 0; index < MAX_WITNESS_ARTIFACTS; index += 1) {
    const exportName = C_WITNESS_EXPORTS[index % C_WITNESS_EXPORTS.length] as string;
    witness.instrument(exportName, signatureOf(exportName), manifest);
  }
  expectReject(
    () => witness.instrument("$canMain", signatureOf("$canMain"), manifest),
    "resource-limit",
  );
  assert.equal(witness.counters().artifacts, MAX_WITNESS_ARTIFACTS);
});

check("bounds-invocation-cap", () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument("$canMain", signatureOf("$canMain"), manifest);
  for (let index = 0; index < MAX_WITNESS_INVOCATIONS; index += 1) {
    witness.begin(artifact.token, artifact.artifact);
  }
  expectReject(() => witness.begin(artifact.token, artifact.artifact), "resource-limit");
  assert.equal(witness.counters().invocations, MAX_WITNESS_INVOCATIONS);
});

check("bounds-malformed-manifest", () => {
  const witness = new CAdapterWitness();
  for (const bad of [null, [], "manifest", {}, { signatures: null }, { signatures: [] }]) {
    expectReject(
      () => witness.instrument("$canMain", signatureOf("$canMain"), bad),
      "invalid-request",
    );
  }
  expectReject(
    () => witness.instrument("$canMain", signatureOf("$canMain"), { signatures: { $canMain: 42 } }),
    "invalid-request",
  );
  expectReject(
    () =>
      witness.instrument("$canMain", signatureOf("$canMain"), {
        signatures: { $canMain: "x".repeat(513) },
      }),
    "invalid-request",
  );
  assert.equal(witness.counters().artifacts, 0);
});

check("bounds-rejections-counted", () => {
  const witness = new CAdapterWitness();
  const artifact = witness.instrument("$canMain", signatureOf("$canMain"), manifest);
  const begun = witness.begin(artifact.token, artifact.artifact);
  expectReject(
    () => witness.observe(artifact.token, artifact.artifact, begun.invocation),
    "permission",
  );
  expectReject(() => witness.begin("cwt:1:0123456789abcdef", "cartifact1"), "not-found");
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
