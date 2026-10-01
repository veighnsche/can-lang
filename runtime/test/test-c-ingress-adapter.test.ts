// NT-I02 C-ingress adapter contract: parse_module maps the K04 parse wire
// shapes to nominal records, check_module maps the Can manifest to the seam
// checker and returns problem strings verbatim, unknown entry kinds throw,
// and revoked owners fail test::stale_handle without touching the checker.
// Compute is pure and local: no transport is involved, and the K05 witness
// stays host-side with no catalogue operations.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import type { Completion } from "../completion.ts";
import { array, dataArray, dataProperty, record } from "../data.ts";
import { createTestOwner } from "../test-support/owner.ts";
import { createCIngress } from "../test-support/slices/i02/c.ts";

const identity = (kind: string, declaration: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration]))
    .digest("hex");
const scalar = (name: string): FailureShape => ({
  identity: identity("primitive", name),
  kind: "primitive",
  declaration: name,
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const text = scalar("str");
const declarations = catalogue.errors.filter((error) => error.name === "test::stale_handle");
const shapeOf = (id: string, fields: readonly { name: string; type: string }[]): FailureShape => ({
  identity: identity("error", id),
  kind: "error",
  declaration: id,
  arguments: [],
  fields: fields.map((field) => ({ name: field.name, type: text.identity })),
  leaves: [],
  inputs: [],
  errors: [],
});
const errorShapes: Map<string, FailureShape> = new Map(
  declarations.map((error) => [error.name, shapeOf(error.identity, error.fields)]),
);
const domain = createDomainRuntime({
  declarations: declarations.map((error) => ({
    identity: error.identity,
    name: error.name,
    parameters: 0,
  })),
  shapes: [text, ...errorShapes.values()],
});
const err = (name: string): string => {
  const shape = errorShapes.get(name);
  if (shape === undefined) throw Error(`no shape for ${name}`);
  return shape.identity;
};

const SOME = "test:some";
const NONE = "test:none";
const typeId = (name: string): string => `test:type:${name}`;

function rig() {
  const owner = createTestOwner(domain, { invalidGrant: err("test::stale_handle") });
  const ingress = createCIngress(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      some: SOME,
      none: NONE,
      parsedModule: typeId("c::parsed_module"),
      parsedEdge: typeId("c::parsed_edge"),
      parsedName: typeId("c::parsed_name"),
    },
    owner,
  );
  return { owner, ingress };
}

const GRANT = "ng1-0123456789abcdef0123456789abcdef";

async function ok<T>(completion: Promise<Completion<T>>): Promise<T> {
  const result = await completion;
  expect(result.kind).toBe("ok");
  if (result.kind !== "ok") throw Error("expected ok");
  return result.value;
}

async function failed(completion: Promise<Completion<never>>) {
  const result = await completion;
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain failure");
  return domainFailureDiagnostics(result.value);
}

const field = (value: unknown, name: string): unknown => dataProperty(value, name);
const some = (value: unknown): unknown => record(SOME, [["value", value]]);
const none = (): unknown => record(NONE, []);

function binding(exported: string, local: string): unknown {
  return record(typeId("c::name_binding"), [
    ["exported", exported],
    ["local", local],
  ]);
}

function fixedEdge(runtimeModule: unknown, resolved: unknown, names: readonly unknown[]): unknown {
  return record(typeId("c::fixed_edge"), [
    ["runtime_module", runtimeModule],
    ["resolved", resolved],
    ["type_only", false],
    ["names", array([...names])],
  ]);
}

function manifestEntry(path: string, bodyPattern: string, edges: readonly unknown[]): unknown {
  return record(typeId("c::manifest_entry"), [
    ["path", path],
    ["body_pattern", bodyPattern],
    ["fixed_edges", array([...edges])],
  ]);
}

function manifest(overrides?: {
  programRoots?: readonly string[];
  runtimeModules?: readonly string[];
  executable?: unknown;
  assertionRoot?: unknown;
  runtimeEdges?: readonly unknown[];
  resolvedEdges?: readonly unknown[];
  exports?: readonly string[];
}): unknown {
  const runtimeEdges = (overrides?.runtimeEdges ?? []).map((edge) => edge);
  const resolvedEdges = (overrides?.resolvedEdges ?? []).map((edge) => edge);
  return record(typeId("c::manifest"), [
    ["local_prefix", "$can"],
    ["program_roots", array([...(overrides?.programRoots ?? [])])],
    ["runtime_modules", array([...(overrides?.runtimeModules ?? [])])],
    ["executable", overrides?.executable ?? manifestEntry("main.ts", ".*", [])],
    ["assertion_root", overrides?.assertionRoot ?? manifestEntry("assert.ts", ".*", [])],
    [
      "assertion_case",
      record(typeId("c::case_requirements"), [
        ["runtime_edges", array(runtimeEdges)],
        ["resolved_edges", array(resolvedEdges)],
        ["exports", array([...(overrides?.exports ?? [])])],
      ]),
    ],
  ]);
}

function runtimeEdge(runtimeModule: string, typeOnly: boolean): unknown {
  return record(typeId("c::runtime_edge"), [
    ["runtime_module", runtimeModule],
    ["type_only", typeOnly],
  ]);
}

test("c::parse_module maps imports, exports, dynamic flag and body", async () => {
  const { owner, ingress } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const parsed = await ok(
    ingress.parseModule(
      handle,
      'import { x as $canX } from "./rt.ts";\nconst n = import("late");\nexport const $canCase = 1;',
    ),
  );
  const imports = dataArray(field(parsed, "imports"), "imports");
  expect(imports).toHaveLength(1);
  expect(field(imports[0], "specifier")).toBe("./rt.ts");
  expect(field(imports[0], "type_only")).toBe(false);
  expect(field(imports[0], "side_effect")).toBe(false);
  const names = dataArray(field(imports[0], "names"), "names");
  expect(names).toHaveLength(1);
  expect(field(names[0], "exported")).toBe("x");
  expect(field(names[0], "local")).toBe("$canX");
  expect(field(names[0], "aliased")).toBe(true);
  expect([...(field(parsed, "exports") as readonly unknown[])]).toEqual(["$canCase"]);
  expect(field(parsed, "dynamic_import")).toBe(true);
  expect(field(parsed, "body")).toBe('const n = import("late");');
});

test("c::parse_module flags type-only and side-effect edges", async () => {
  const { owner, ingress } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const parsed = await ok(
    ingress.parseModule(handle, 'import type { t as $canT } from "./t.ts";\nimport "./run.ts";'),
  );
  const imports = dataArray(field(parsed, "imports"), "imports");
  expect(imports).toHaveLength(2);
  expect(field(imports[0], "type_only")).toBe(true);
  expect(field(imports[0], "side_effect")).toBe(false);
  expect(field(imports[1], "type_only")).toBe(false);
  expect(field(imports[1], "side_effect")).toBe(true);
  expect(field(parsed, "dynamic_import")).toBe(false);
});

test("c::check_module accepts a clean assertion case", async () => {
  const { owner, ingress } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const problems = (await ok(
    ingress.checkModule(
      handle,
      manifest({
        programRoots: ["assertions/"],
        runtimeModules: ["completion.ts"],
        runtimeEdges: [runtimeEdge("completion.ts", false)],
        exports: [],
      }),
      "assertion_case",
      "assertions/case1.ts",
      'import { x as $canX } from "../rt/completion.ts";',
    ),
  )) as readonly unknown[];
  expect([...problems]).toEqual([]);
});

test("c::check_module reports seam violations verbatim", async () => {
  const { owner, ingress } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const problems = (await ok(
    ingress.checkModule(
      handle,
      manifest({ programRoots: ["assertions/"], exports: ["$canCase"] }),
      "assertion_case",
      "assertions/case2.ts",
      'import { y } from "bare";\nconst n = import("late");\nexport const $canCase = 1;',
    ),
  )) as readonly unknown[];
  expect(problems).toContain("dynamic import() is not a seam import");
  expect(problems).toContain("non-relative specifier bare");
});

test("c::check_module rejects unaliased and unprefixed bindings", async () => {
  const { owner, ingress } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const problems = (await ok(
    ingress.checkModule(
      handle,
      manifest({ programRoots: [""], exports: [] }),
      "assertion_case",
      "assertions/case3.ts",
      'import { y } from "./other.ts";',
    ),
  )) as readonly unknown[];
  expect(problems).toContain("unaliased binding y from ./other.ts");
  expect(problems).toContain("local y misses the $can prefix");
});

test("c::check_module rejects a second runtime root", async () => {
  const { owner, ingress } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const problems = (await ok(
    ingress.checkModule(
      handle,
      manifest({ programRoots: ["assertions/"], runtimeModules: ["m.ts"], exports: [] }),
      "assertion_case",
      "assertions/case4.ts",
      'import { a as $canA } from "../r1/m.ts";\nimport { b as $canB } from "../r2/m.ts";',
    ),
  )) as readonly unknown[];
  expect(problems).toContain("second runtime root swaps the runtime: r1, r2");
});

test("c::check_module maps manifest options onto fixed edges", async () => {
  const { owner, ingress } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const clean = manifest({
    programRoots: [""],
    runtimeModules: ["completion.ts"],
    executable: manifestEntry("main.ts", ".*", [
      fixedEdge(some("completion.ts"), none(), [binding("x", "$canX")]),
    ]),
  });
  const problems = (await ok(
    ingress.checkModule(
      handle,
      clean,
      "executable",
      "main.ts",
      'import { x as $canX } from "./rt/completion.ts";\nimport { main as $canMain } from "./prog.ts";',
    ),
  )) as readonly unknown[];
  expect([...problems]).toEqual([]);
  const missing = (await ok(
    ingress.checkModule(
      handle,
      clean,
      "executable",
      "main.ts",
      'import { main as $canMain } from "./prog.ts";',
    ),
  )) as readonly unknown[];
  expect(missing).toContain("missing fixed edge completion.ts");
});

test("c::check_module enforces the executable supervisor shape", async () => {
  const { owner, ingress } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const problems = (await ok(
    ingress.checkModule(
      handle,
      manifest({ programRoots: [""], exports: [] }),
      "executable",
      "wrong.ts",
      'import { main as $canMain } from "./prog.ts";',
    ),
  )) as readonly unknown[];
  expect(problems).toContain("executable entry must be main.ts");
});

test("c::check_module rejects unknown entry kinds before reading the manifest", async () => {
  const { owner, ingress } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  await expect(
    ingress.checkModule(handle, "not-a-manifest", "executables", "main.ts", ""),
  ).rejects.toThrow('c entry kind "executables" is not admitted');
});

test("c::check_module rejects malformed manifests, options and sources", async () => {
  const { owner, ingress } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  await expect(
    ingress.checkModule(handle, "not-a-manifest", "executable", "main.ts", ""),
  ).rejects.toThrow(TypeError);
  await expect(ingress.parseModule(handle, 42)).rejects.toThrow("c::parse_module source");
  const badOption = manifest({
    executable: manifestEntry("main.ts", ".*", [
      fixedEdge(record("test:bogus", []), none(), [binding("x", "$canX")]),
    ]),
  });
  await expect(ingress.checkModule(handle, badOption, "executable", "main.ts", "")).rejects.toThrow(
    "needs an option",
  );
});

test("c revoked owners fail stale without touching the checker", async () => {
  const { owner, ingress } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  await ok(owner.releaseGrant(handle));
  const staleParse = await failed(ingress.parseModule(handle, "") as Promise<Completion<never>>);
  expect(staleParse.declaration.name).toBe("test::stale_handle");
  const staleCheck = await failed(
    ingress.checkModule(handle, "not-a-manifest", "executable", "main.ts", "") as Promise<
      Completion<never>
    >,
  );
  expect(staleCheck.declaration.name).toBe("test::stale_handle");
});
