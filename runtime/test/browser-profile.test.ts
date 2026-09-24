// UP07 browser-profile conformance. The sealed portable production profile is
// the seven browser alternates below plus their resolved closure: every edge
// to a host-coupled canonical module resolves through OVERLAY at browser
// bundle time. This suite pins the overlay table, the resolved module
// inventory, the export surfaces, host-predicate and digest differentials,
// and executes the full profile inside a staged overlaid tree (including a
// zero-alias Bun.build for the browser target).
import { expect, test } from "bun:test";
import { createHash } from "node:crypto";
import {
  cpSync,
  existsSync,
  mkdtempSync,
  mkdirSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { pathToFileURL } from "node:url";
import { GenMapping, addMapping, toEncodedMap } from "../../tools/runtime/vendor/source-maps.mjs";
import {
  concreteTypeDigestInput,
  shapeIdentityKey,
  type ErrorPlan,
  type FailureShape,
} from "../domain-core.ts";
import * as canonicalOwner from "../owner.ts";
import * as browserOwner from "../browser/owner.ts";
import * as canonicalDomain from "../domain.ts";
import * as browserDomain from "../browser/domain.ts";
import * as canonicalDiagnostics from "../diagnostics.ts";
import * as browserDiagnostics from "../browser/diagnostics.ts";
import * as canonicalCallable from "../callable.ts";
import * as browserCallable from "../browser/callable.ts";
import * as canonicalCoordination from "../coordination.ts";
import * as browserCoordination from "../browser/coordination.ts";
import * as canonicalEntry from "../entry.ts";
import * as browserEntry from "../browser/entry.ts";
import * as canonicalReflect from "../reflect.ts";
import * as browserReflect from "../browser/reflect.ts";
import { dataKeys, dataProperty } from "../data.ts";
import { describeNativeFailure } from "../failure.ts";

const RUNTIME = join(dirname(import.meta.path), "..");

// Bundle-time transitive specifier overlay for browser builds. Any edge to a
// canonical key resolves to its alternate instead; the walker below applies
// the same rule, so the inventory it pins is the resolved browser closure.
const OVERLAY: Readonly<Record<string, string>> = Object.freeze({
  "runtime/reflect.ts": "runtime/browser/reflect.ts",
  "runtime/domain.ts": "runtime/browser/domain.ts",
  "runtime/diagnostics.ts": "runtime/browser/diagnostics.ts",
  "runtime/owner.ts": "runtime/browser/owner.ts",
  "runtime/callable.ts": "runtime/browser/callable.ts",
  "runtime/coordination.ts": "runtime/browser/coordination.ts",
  "runtime/entry.ts": "runtime/browser/entry.ts",
});
const ROOTS = Object.freeze(Object.values(OVERLAY));

// Resolved browser production inventory for the core profile. Any drift fails
// loudly: module changes merge through the coordinator, never per lane.
const INVENTORY = Object.freeze([
  "runtime/browser/callable.ts",
  "runtime/browser/coordination.ts",
  "runtime/browser/diagnostics.ts",
  "runtime/browser/domain.ts",
  "runtime/browser/entry.ts",
  "runtime/browser/owner.ts",
  "runtime/browser/reflect.ts",
  "runtime/catalogue.ts",
  "runtime/completion.ts",
  "runtime/data.ts",
  "runtime/diagnostics-core.ts",
  "runtime/domain-core.ts",
  "runtime/failure.ts",
  "runtime/owner-core.ts",
  "runtime/primitive.ts",
  "runtime/vendor/trace-mapping.ts",
]);

function normalize(from: string, specifier: string): string {
  const parts = join(dirname(from), specifier).split("/");
  const out: string[] = [];
  for (const part of parts) {
    if (part === "" || part === ".") continue;
    if (part === "..") out.pop();
    else out.push(part);
  }
  return out.join("/");
}

function walkProfile(): { files: string[]; problems: string[] } {
  const seen = new Set<string>();
  const problems: string[] = [];
  const queue = [...ROOTS];
  const transpiler = new Bun.Transpiler({ loader: "ts" });
  while (queue.length) {
    const file = queue.pop()!;
    if (seen.has(file)) continue;
    seen.add(file);
    const source = readFileSync(join(dirname(RUNTIME), file), "utf8");
    for (const edge of transpiler.scanImports(source) as { kind: string; path: string }[]) {
      if (edge.kind === "dynamic-import" || edge.kind === "require-call") {
        problems.push(`${file}: ${edge.kind} ${edge.path}`);
        continue;
      }
      if (edge.path.startsWith("node:") || edge.path.startsWith("bun:")) {
        problems.push(`${file}: native edge ${edge.path}`);
        continue;
      }
      if (!edge.path.startsWith("./") && !edge.path.startsWith("../")) {
        problems.push(`${file}: non-relative edge ${edge.path}`);
        continue;
      }
      let resolved = normalize(file, edge.path);
      if (!resolved.endsWith(".ts")) {
        problems.push(`${file}: non-module edge ${edge.path}`);
        continue;
      }
      resolved = OVERLAY[resolved] ?? resolved;
      if (resolved.includes("/assert/") || resolved.endsWith("assert.ts"))
        problems.push(`${file}: assertion edge ${resolved}`);
      if (!seen.has(resolved)) queue.push(resolved);
    }
  }
  return { files: [...seen].sort(), problems };
}

test("browser overlay table is complete and inventory is sealed", () => {
  expect(Object.keys(OVERLAY).sort()).toEqual([
    "runtime/callable.ts",
    "runtime/coordination.ts",
    "runtime/diagnostics.ts",
    "runtime/domain.ts",
    "runtime/entry.ts",
    "runtime/owner.ts",
    "runtime/reflect.ts",
  ]);
  for (const [canonical, alternate] of Object.entries(OVERLAY)) {
    expect(existsSync(join(dirname(RUNTIME), canonical))).toBe(true);
    expect(existsSync(join(dirname(RUNTIME), alternate))).toBe(true);
  }
  const { files, problems } = walkProfile();
  expect(problems).toEqual([]);
  expect(files).toEqual([...INVENTORY]);
  for (const canonical of Object.keys(OVERLAY)) expect(files).not.toContain(canonical);
});

test("browser surfaces expose the profile contract with fail-closed ambient names", async () => {
  const keys = (ns: object) => Object.keys(ns).sort();
  expect(keys(browserOwner)).toEqual([
    "closeResource",
    "closeResourceWithContext",
    "guardCallback",
    "guardCallbackWithContext",
    "launchNative",
    "launchNativeWithContext",
    "launchOwned",
    "launchOwnedWithContext",
    "registerCallableCaptures",
    "registerResource",
    "registerResourceWithContext",
    "resourceStatus",
    "runExplicitRoot",
    "runOwnedRoot",
    "useResource",
    "useResourceWithContext",
    "withScope",
    "withScopeWithContext",
  ]);
  expect(keys(canonicalOwner).length).toBe(18);
  // Ambient names exist only so shared modules instantiate; every ambient
  // call fails closed instead of substituting synchronous context.
  for (const ambient of ["registerResource", "launchOwned", "launchNative", "guardCallback"]) {
    expect(() =>
      (browserOwner as unknown as Record<string, (...args: never[]) => unknown>)[ambient](),
    ).toThrow("ambient ownership is unavailable in the browser profile");
  }
  await expect(browserOwner.useResource(undefined, "k", async () => ({}) as never)).rejects.toThrow(
    "ambient ownership is unavailable in the browser profile",
  );
  await expect(browserOwner.closeResource(undefined, "k")).rejects.toThrow(
    "ambient ownership is unavailable in the browser profile",
  );
  await expect(browserOwner.withScope(async () => ({}) as never)).rejects.toThrow(
    "ambient ownership is unavailable in the browser profile",
  );
  await expect(browserOwner.runOwnedRoot(async () => ({}) as never)).rejects.toThrow(
    "ambient ownership is unavailable in the browser profile",
  );
  expect(keys(browserDomain)).toEqual([
    "concreteTypeDigestInput",
    "createDomainRuntime",
    "createDomainRuntimeWithDigest",
    "domainFailureDiagnostics",
    "isDomainFailure",
    "shapeIdentityKey",
    "verifyErrorPlan",
  ]);
  expect(keys(canonicalDomain)).toEqual([
    "concreteTypeDigestInput",
    "createDomainRuntime",
    "createDomainRuntimeWithDigest",
    "domainFailureDiagnostics",
    "isDomainFailure",
    "shapeIdentityKey",
  ]);
  expect(keys(browserDiagnostics)).toEqual([
    "configureBrowserDiagnostics",
    "configureFromData",
    "diagnosticFrames",
    "locateOrigin",
    "reportBrowserDiagnostic",
  ]);
  expect(keys(browserDiagnostics)).not.toContain("configureDiagnostics");
  expect(keys(canonicalDiagnostics)).toEqual([
    "configureDiagnostics",
    "configureFromData",
    "diagnosticFrames",
    "locateOrigin",
  ]);
  expect(keys(browserCallable)).toEqual(keys(canonicalCallable));
  expect(keys(browserCoordination)).toEqual(["aggregate", "handle", "settleWithContext"]);
  expect(keys(browserCoordination)).not.toContain("settle");
  expect(keys(canonicalCoordination)).toContain("settle");
  expect(keys(browserEntry)).toEqual(["runBrowserEntry"]);
  expect(keys(canonicalEntry)).toEqual(["runEntry"]);
  expect(keys(browserReflect)).toEqual(keys(canonicalReflect));
  expect(keys(browserReflect)).toEqual(["isHostNativeError", "isHostProxy"]);
});

test("host predicates agree on genuine values and diverge only contained", () => {
  for (const value of [{}, [], "x", 1n, null, undefined, () => 0]) {
    expect(browserReflect.isHostProxy(value)).toBe(false);
    expect(canonicalReflect.isHostProxy(value)).toBe(false);
  }
  expect(browserReflect.isHostNativeError(new TypeError("x"))).toBe(true);
  expect(canonicalReflect.isHostNativeError(new TypeError("x"))).toBe(true);
  let traps = 0;
  const trap = (): never => {
    traps++;
    throw new Error("trap");
  };
  const throwing = new Proxy({}, { getPrototypeOf: trap });
  expect(canonicalReflect.isHostProxy(throwing)).toBe(true);
  expect(traps).toBe(0);
  expect(browserReflect.isHostProxy(throwing)).toBe(true);
  expect(traps).toBe(1);
  // Documented miss: a fully silent proxy is invisible to portable checks.
  // Callers still fail closed through contained Reflect errors or brand checks.
  expect(canonicalReflect.isHostProxy(new Proxy({}, {}))).toBe(true);
  expect(browserReflect.isHostProxy(new Proxy({}, {}))).toBe(false);
  expect(describeNativeFailure(new Proxy({}, {}))).toBe("native proxy");
});

test("profile data guards reject proxies with the canonical identity", () => {
  const throwing = new Proxy(
    {},
    {
      ownKeys: () => {
        throw new Error("trap");
      },
      getOwnPropertyDescriptor: () => {
        throw new Error("trap");
      },
    },
  );
  for (const fn of [() => dataKeys(throwing), () => dataProperty(throwing, "a")])
    expect(fn).toThrow("expected non-proxy data object");
});

test("canonical digest bytes match WebCrypto over every shape kind", async () => {
  const nodeHex = (input: string) => createHash("sha256").update(input).digest("hex");
  const subtleHex = async (input: string) => {
    const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(input));
    return [...new Uint8Array(digest)].map((b) => b.toString(16).padStart(2, "0")).join("");
  };
  const byId = new Map<string, FailureShape>();
  const resolve = (id: string | undefined): FailureShape => {
    const found = id ? byId.get(id) : undefined;
    if (!found) throw new Error("missing shape");
    return found;
  };
  const shapes: FailureShape[] = [
    {
      identity: "int",
      kind: "primitive",
      declaration: "int",
      arguments: [],
      fields: [],
      leaves: [],
      inputs: [],
      errors: [],
    },
    {
      identity: "arr",
      kind: "array",
      arguments: [],
      fields: [],
      leaves: [],
      element: "int",
      inputs: [],
      errors: [],
    },
    {
      identity: "fn",
      kind: "callable",
      arguments: [],
      fields: [],
      leaves: [],
      result: "int",
      inputs: ["int"],
      errors: [],
    },
    {
      identity: "rec",
      kind: "record",
      declaration: "can.project.root::app::r",
      arguments: [],
      fields: [{ name: "v", type: "int" }],
      leaves: [],
      inputs: [],
      errors: [],
    },
  ];
  for (const shape of shapes) byId.set(shape.identity, shape);
  for (const shape of shapes) {
    const input = concreteTypeDigestInput(shapeIdentityKey(shape, resolve));
    expect(await subtleHex(input)).toBe(nodeHex(input));
  }
});

function fixturePlan(): ErrorPlan {
  const hash = (key: unknown) =>
    createHash("sha256")
      .update("can-concrete-type-v1\0" + JSON.stringify(key))
      .digest("hex");
  const shape = (kind: string, declaration: string, args: string[] = []) => ({
    identity: hash([kind, declaration, ...args]),
    kind,
    declaration,
    arguments: args,
    fields: [] as { name: string; type: string }[],
    leaves: [] as string[],
    inputs: [] as string[],
    errors: [] as string[],
  });
  const integer = shape("primitive", "int"),
    text = shape("primitive", "str"),
    standard = shape("opaque", "can.prelude@1::standard_failure");
  const fault = shape("error", "can.project.root/app::fault", [integer.identity]);
  fault.fields = [{ name: "value", type: integer.identity }];
  const offline = shape("error", "can.project.root/app::offline");
  offline.fields = [{ name: "message", type: text.identity }];
  const variant = shape("variant", "can.project.root/app::failures");
  variant.leaves = [standard.identity, fault.identity, offline.identity];
  const arrayShape: FailureShape = {
    ...shape("array", ""),
    identity: hash(["array", "", variant.identity]),
    element: variant.identity,
  };
  const combined = shape("error", "can.prelude@1::all_failed", [variant.identity]);
  combined.fields = [{ name: "failures", type: arrayShape.identity }];
  return {
    declarations: [
      { identity: fault.declaration!, name: "app::fault", parameters: 1 },
      { identity: offline.declaration!, name: "app::offline", parameters: 0 },
      { identity: combined.declaration!, name: "all_failed", parameters: 1 },
    ],
    shapes: [integer, text, standard, fault, offline, variant, arrayShape, combined],
  };
}

function fixtureTable(): { index: unknown; maps: Record<string, unknown> } {
  const source = "can.project.root/app/main.can";
  const map = new GenMapping({ file: "app.js" });
  addMapping(map, {
    generated: { line: 1, column: 4 },
    source,
    original: { line: 2, column: 4 },
    name: "call:10:20",
  });
  return {
    index: {
      schemaVersion: 1,
      kind: "can.source-index",
      sources: [
        {
          id: source,
          path: "app/main.can",
          spans: {
            "call:10:20": {
              start: 10,
              end: 20,
              line: 2,
              column: 4,
              endLine: 2,
              endColumn: 14,
              operation: "call",
            },
          },
        },
      ],
      modules: [{ path: "app.js" }],
    },
    maps: { "app.js": toEncodedMap(map) },
  };
}

async function withStagedProfile<T>(fn: (staged: string) => Promise<T>): Promise<T> {
  const root = mkdtempSync(join(tmpdir(), "can-browser-profile-"));
  try {
    cpSync(RUNTIME, join(root, "runtime"), { recursive: true });
    // Bundle-time overlay, staged as forwarding modules: every canonical
    // specifier resolves to its alternate with imports intact.
    for (const [canonical, alternate] of Object.entries(OVERLAY)) {
      const from = join(root, canonical);
      const to = join(root, alternate);
      const relative = "./" + to.slice(dirname(from).length + 1).replace(/\\/g, "/");
      writeFileSync(from, `export * from ${JSON.stringify(relative)};\n`);
    }
    return await fn(root);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
}

test("staged profile executes sealed semantics without Node builtins", async () => {
  await withStagedProfile(async (root) => {
    const driver = await import(
      pathToFileURL(join(root, "runtime/test/browser-profile-staged.ts")).href
    );
    const passed = (await driver.runStagedProfile({
      plan: fixturePlan(),
      table: fixtureTable(),
    })) as string[];
    expect(passed).toEqual([
      "reflect",
      "data",
      "failure",
      "completion",
      "domain",
      "diagnostics",
      "owner",
      "callable",
      "coordination",
      "entry",
      "semantics",
    ]);
  });
}, 60000);

test("staged profile bundles for the browser target without aliases", async () => {
  await withStagedProfile(async (root) => {
    const entry = join(root, "runtime/profile-probe-entry.ts");
    writeFileSync(
      entry,
      [
        'export { verifyErrorPlan, createDomainRuntime } from "./browser/domain.ts";',
        'export { configureBrowserDiagnostics, reportBrowserDiagnostic, diagnosticFrames } from "./browser/diagnostics.ts";',
        'export { runExplicitRoot, launchOwnedWithContext, withScopeWithContext } from "./browser/owner.ts";',
        'export { ownCallable, callableEqual } from "./browser/callable.ts";',
        'export { settleWithContext, handle, aggregate } from "./browser/coordination.ts";',
        'export { runBrowserEntry } from "./browser/entry.ts";',
        'export { isHostProxy, isHostNativeError } from "./browser/reflect.ts";',
        "",
      ].join("\n"),
    );
    mkdirSync(join(root, "out"), { recursive: true });
    const result = await Bun.build({
      entrypoints: [entry],
      target: "browser",
      outdir: join(root, "out"),
    });
    expect(result.success).toBe(true);
    expect(result.outputs.length).toBeGreaterThan(0);
  });
}, 60000);
