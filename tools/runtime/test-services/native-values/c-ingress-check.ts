// K04 bounded self-check: fixed ingress seam mechanics only.
//
// Run with: bun tools/runtime/test-services/native-values/c-ingress-check.ts
// Local controls only. No services, transports, timers, or live runtimes.
// Positive fixtures mirror the current emitter output shapes from
// compiler/internal/emit/program_entry.go, runtime_core.go and modules.go
// (relative specifiers, $can aliases, supervisor bodies); negatives mutate
// one seam property each.
import { strict as assert } from "node:assert";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { checkModule, runtimeRootsOf, type EntryKind, type IngressManifest } from "./c-ingress.ts";

const here = dirname(fileURLToPath(import.meta.url));
const manifest = JSON.parse(
  readFileSync(join(here, "c-ingress.manifest.json"), "utf8"),
) as IngressManifest;

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectClean(kind: EntryKind, path: string, source: string): void {
  assert.deepEqual(checkModule(manifest, kind, path, source), []);
}

function expectProblem(kind: EntryKind, path: string, source: string, want: string): void {
  const problems = checkModule(manifest, kind, path, source);
  assert.ok(
    problems.some((problem) => problem.includes(want)),
    `expected ${want} in ${JSON.stringify(problems)}`,
  );
}

const EXEC_OK = `import { runEntry as $canRunEntry } from "./runtime/entry.ts";
import { $canInitialize as $canInitialize } from "./program/state.ts";
import { configureDiagnostics as $canConfigureDiagnostics } from "./runtime/diagnostics.ts";
import { $canMainFn as $canMain } from "./packages/app/main.ts";
process.exitCode = await $canRunEntry(() => {$canConfigureDiagnostics(import.meta.url); $canInitialize();}, $canMain, process.argv.slice(2));
`;

const ROOT_OK = `import { runAssertionRoot as $canRunAssertionRoot } from "./runtime/assert/runner.ts";
import { $canInitialize as $canInitialize } from "./program/state.ts";
import { configureDiagnostics as $canConfigureDiagnostics } from "./runtime/diagnostics.ts";
import { $canCase as $canCase0 } from "./assertions/aa.ts";
import { $canCase as $canCase1 } from "./assertions/bb.ts";
process.exitCode = await $canRunAssertionRoot([$canCase0, $canCase1], () => {$canConfigureDiagnostics(import.meta.url); $canInitialize();}, process.argv.slice(2));
`;

const RUNTIME_EDGES = [
  `import { array as $canArray } from "../runtime/collections/array.ts";`,
  `import { callContext as $canCallContext, scopeRequest as $canScopeRequest } from "../runtime/assert/context.ts";`,
  `import { settle as $canCoordinateSettle } from "../runtime/coordination.ts";`,
  `import type { Participant as $canParticipant } from "../runtime/owner.ts";`,
  `import { byteLength as $canByteLength } from "../runtime/bytes.ts";`,
  `import { createImage as $canCreateImage } from "../runtime/image.ts";`,
  `import { ownCallable as $canOwnCallable } from "../runtime/callable.ts";`,
  `import { withFixture as $canWithFixture } from "../runtime/assert/fixtures.ts";`,
  `import { policyBase as $canPolicyBase } from "../runtime/assert/policy.ts";`,
  `import { success as $canSuccess, invoke as $canInvoke } from "../runtime/completion.ts";`,
  `import type { Completion as $canCompletion } from "../runtime/completion.ts";`,
  `import { record as $canRecord } from "../runtime/data.ts";`,
  `import { index as $canIndex } from "../runtime/primitive.ts";`,
  `import { captureStandard as $canCaptureStandard } from "../runtime/failure.ts";`,
  `import { sha256 as $canSHA256 } from "../runtime/platform/crypto/primitives.ts";`,
];
const CASE_OK = `${RUNTIME_EDGES.join("\n")}
import { $canInitialize as $canInitialize } from "../program/state.ts";
import { $canHelper as $canHelper } from "../packages/app/helper.ts";
export const $canCase = Object.freeze({root: Object.freeze({}), actual: $canActual, expected: $canExpected});
`;

check("executable-ok", () => {
  expectClean("executable", "entry.ts", EXEC_OK);
});
check("executable-second-runtime", () => {
  expectProblem(
    "executable",
    "entry.ts",
    `${EXEC_OK}import { x as $canX } from "./runtime2/entry.ts";\n`,
    "second runtime root",
  );
});
check("executable-dynamic-import", () => {
  expectProblem(
    "executable",
    "entry.ts",
    `${EXEC_OK}const late = await import("./runtime/evil.ts");\n`,
    "dynamic import()",
  );
});
check("executable-bare-specifier", () => {
  expectProblem(
    "executable",
    "entry.ts",
    EXEC_OK.replace('"./runtime/entry.ts"', '"runtime/entry.ts"'),
    "non-relative specifier",
  );
});
check("executable-unaliased", () => {
  expectProblem(
    "executable",
    "entry.ts",
    EXEC_OK.replace("{ runEntry as $canRunEntry }", "{ runEntry }"),
    "unaliased binding",
  );
});
check("executable-missing-prefix", () => {
  expectProblem(
    "executable",
    "entry.ts",
    EXEC_OK.replace("$canMainFn as $canMain", "$canMainFn as main"),
    "misses the $can prefix",
  );
});
check("executable-export", () => {
  expectProblem("executable", "entry.ts", `${EXEC_OK}export const $canExtra = 1;\n`, "exports");
});
check("executable-body", () => {
  expectProblem(
    "executable",
    "entry.ts",
    EXEC_OK.replace("process.exitCode = await", "process.exitCode = "),
    "supervisor shape",
  );
});
check("executable-missing-edge", () => {
  expectProblem(
    "executable",
    "entry.ts",
    EXEC_OK.split("\n")
      .filter((line) => !line.includes("diagnostics.ts"))
      .join("\n"),
    "missing fixed edge",
  );
});
check("root-ok", () => {
  expectClean("assertion_root", "entry.ts", ROOT_OK);
});
check("root-no-case", () => {
  expectProblem(
    "assertion_root",
    "entry.ts",
    ROOT_OK.split("\n")
      .filter((line) => !line.includes("assertions/"))
      .join("\n"),
    "at least one case edge",
  );
});
check("root-case-outside", () => {
  expectProblem(
    "assertion_root",
    "entry.ts",
    ROOT_OK.replace('"./assertions/bb.ts"', '"./packages/bb.ts"'),
    "outside assertions/",
  );
});
check("case-ok", () => {
  expectClean("assertion_case", "assertions/9f.ts", CASE_OK);
});
check("case-missing-completion-type", () => {
  expectProblem(
    "assertion_case",
    "assertions/9f.ts",
    CASE_OK.split("\n")
      .filter((line) => !line.includes("import type { Completion"))
      .join("\n"),
    "missing runtime edge completion.ts (type-only)",
  );
});
check("case-extra-runtime", () => {
  expectProblem(
    "assertion_case",
    "assertions/9f.ts",
    `${CASE_OK}import { byteLength as $canByteLength2 } from "../runtime/bytes.ts";\n`,
    "unexpected runtime edge",
  );
});
check("case-outside-seam", () => {
  expectProblem(
    "assertion_case",
    "assertions/9f.ts",
    `${CASE_OK}import { text as $canText } from "../runtime/text.ts";\n`,
    "target outside the seam",
  );
});
check("case-wrong-export", () => {
  expectProblem(
    "assertion_case",
    "assertions/9f.ts",
    CASE_OK.replace("export const $canCase", "export const $canWrong"),
    "must export exactly $canCase",
  );
});
check("case-escape", () => {
  expectProblem(
    "assertion_case",
    "assertions/9f.ts",
    `${CASE_OK}import { x as $canX } from "../../evil.ts";\n`,
    "escapes the generation root",
  );
});
check("same-realm-closure", () => {
  assert.deepEqual(runtimeRootsOf(manifest, "entry.ts", EXEC_OK), ["runtime"]);
  assert.deepEqual(runtimeRootsOf(manifest, "entry.ts", ROOT_OK), ["runtime"]);
  assert.deepEqual(runtimeRootsOf(manifest, "assertions/9f.ts", CASE_OK), ["runtime"]);
});

console.log(`${passed.length} ingress checks passed`);
