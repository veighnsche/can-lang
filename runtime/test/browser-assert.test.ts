import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { value, type Completion } from "../completion.ts";
import { assertionContext, scopeRequest, isAssertScope } from "../assert/context.ts";
import { createBrowser, createBrowserState } from "../platform/browser.ts";

// T24 elided harness scope arguments. Every browser handle input can be
// elided in assertions; the inert scope token fails closed as disposed
// (disposals no-op) so asserted handlers exercise their disposal arms.
// Mount takes no handle and stays live; non-token values still throw a
// resource-state fault instead of masquerading as disposal.
const hash = (kind: string, name: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, name]))
    .digest("hex");
const shape = (
  kind: string,
  declaration: string,
  fields: { name: string; type: string }[] = [],
): FailureShape => ({
  identity: hash(kind, declaration),
  kind,
  declaration,
  fields,
  arguments: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const str = shape("primitive", "str"),
  int = shape("primitive", "int");
const declarations = catalogue.errors.filter((e) =>
  [
    "browser::missing_root",
    "browser::disposed",
    "browser::rejected",
    "browser::stale_version",
  ].includes(e.name),
);
const errors = declarations.map((e) =>
  shape(
    "error",
    e.identity,
    e.fields.map((f) => ({ name: f.name, type: f.type === "int" ? int.identity : str.identity })),
  ),
);
const domain = createDomainRuntime({
  declarations: declarations.map((e) => ({ ...e, parameters: 0 })),
  shapes: [str, int, ...errors],
});
const contracts = {
  missingRoot: errors[0]!.identity,
  disposed: errors[1]!.identity,
  rejected: errors[2]!.identity,
  event: "test-browser-event",
};
const stateContracts = {
  disposed: errors[1]!.identity,
  stale: errors[3]!.identity,
  state: "test-browser-state",
  snapshot: "test-browser-snapshot",
};
const failureName = (result: Completion<unknown>): string => {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw new Error("expected domain failure");
  return domainFailureDiagnostics(result.value).declaration.name;
};

test("scope branding marks only the harness token", () => {
  const context = assertionContext({ package: "p", declaration: "d", name: "n" });
  const scope = scopeRequest(context);
  expect(isAssertScope(scope)).toBe(true);
  expect(isAssertScope(scopeRequest(context))).toBe(true);
  expect(isAssertScope({})).toBe(false);
  expect(isAssertScope("scope")).toBe(false);
  expect(isAssertScope(undefined)).toBe(false);
});

test("elided handles fail closed as disposed across the browser surface", async () => {
  const context = assertionContext({ package: "p", declaration: "d", name: "n" });
  const scope = scopeRequest(context);
  const browser = createBrowser(domain, contracts);
  const states = createBrowserState(domain, stateContracts);
  for (const result of [
    await browser.root(scope),
    await browser.openView(scope),
    await browser.createElement(scope, "div"),
    await browser.createText(scope, "hi"),
    await browser.setText(scope, "hi"),
    await browser.setAttribute(scope, "id", "x"),
    await browser.removeAttribute(scope, "id"),
    await browser.appendChild(scope, scope),
    await browser.removeNode(scope),
    await browser.focus(scope),
    await browser.onEvent(scope, scope, "click", async () => {}),
    await browser.setTimeout(scope, 0, async () => {}, context),
    await states.createState(scope, 1),
    await states.readState(scope),
    await states.replaceState(scope, 0, 1),
  ]) {
    expect(failureName(result)).toBe("browser::disposed");
  }
  value(await browser.disposeView(scope));
  value(await browser.disposeApp(scope));
});

test("mount stays live and mistyped values still fault", async () => {
  const context = assertionContext({ package: "p", declaration: "d", name: "n" });
  const scope = scopeRequest(context);
  expect((globalThis as { document?: unknown }).document).toBeUndefined();
  const browser = createBrowser(domain, contracts);
  const states = createBrowserState(domain, stateContracts);
  expect(failureName(await browser.mount("app"))).toBe("browser::missing_root");
  await expect(browser.root({})).rejects.toThrow();
  await expect(states.readState(scope === null ? null : {})).rejects.toThrow();
});
