import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createBrowser, type BrowserDocument } from "../platform/browser.ts";
import { value, type Completion } from "../completion.ts";
import { dataProperty, recordIdentity } from "../data.ts";

// UP13 strict query bootstrap. The location host is a minimal stub: the
// adapter reads only its search string, never the DOM. Every budget,
// malformed, duplicate and alias case below runs against the real native
// URLSearchParams pair semantics.
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
const str = shape("primitive", "str");
const declarations = catalogue.errors.filter((e) =>
  [
    "browser::missing_root",
    "browser::disposed",
    "browser::rejected",
    "browser::invalid_query",
  ].includes(e.name),
);
const errors = declarations.map((e) =>
  shape(
    "error",
    e.identity,
    e.fields.map((f) => ({ name: f.name, type: str.identity })),
  ),
);
const domain = createDomainRuntime({
  declarations: declarations.map((e) => ({ ...e, parameters: 0 })),
  shapes: [str, ...errors],
});
const contracts = {
  missingRoot: errors[0]!.identity,
  disposed: errors[1]!.identity,
  rejected: errors[2]!.identity,
  event: "test-browser-event",
  invalidQuery: errors[3]!.identity,
  some: "test-browser-some",
  none: "test-browser-none",
};
const setup = (search: string) => {
  const host = { location: { search } } as unknown as BrowserDocument;
  return createBrowser(domain, contracts, host);
};
const failureName = (result: Completion<unknown>): string => {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw new Error("expected domain failure");
  return domainFailureDiagnostics(result.value).declaration.name;
};
const failurePayload = (result: Completion<unknown>): unknown => {
  if (result.kind !== "domain") throw new Error("expected domain failure");
  return domainFailureDiagnostics(result.value).payload;
};

test("one occurrence returns some and zero returns none", async () => {
  const one = value(await setup("?invoice=inv-7").queryParameter("invoice"));
  expect(recordIdentity(one)).toBe(contracts.some);
  expect(dataProperty(one, "value")).toBe("inv-7");
  for (const search of ["", "?", "?other=1", "?invoice2=7"]) {
    const absent = value(await setup(search).queryParameter("invoice"));
    expect(recordIdentity(absent)).toBe(contracts.none);
  }
});

test("bare keys, empty segments and semicolons follow native pair semantics", async () => {
  const bare = value(await setup("?invoice").queryParameter("invoice"));
  expect(recordIdentity(bare)).toBe(contracts.some);
  expect(dataProperty(bare, "value")).toBe("");
  const padded = value(await setup("?invoice=7&&&").queryParameter("invoice"));
  expect(dataProperty(padded, "value")).toBe("7");
  const semi = value(await setup("?invoice=7;8").queryParameter("invoice"));
  expect(dataProperty(semi, "value")).toBe("7;8");
  const plus = value(await setup("?invoice=a+b").queryParameter("invoice"));
  expect(dataProperty(plus, "value")).toBe("a b");
});

test("encoded-key aliases count toward duplicate detection", async () => {
  const aliased = await setup("?invoice=7&%69nvoice=8").queryParameter("invoice");
  expect(failureName(aliased)).toBe("browser::invalid_query");
  expect(dataProperty(failurePayload(aliased), "key")).toBe("invoice");
  expect(dataProperty(failurePayload(aliased), "reason")).toBe("duplicate");
  const direct = await setup("?invoice=7&invoice=8").queryParameter("invoice");
  expect(failureName(direct)).toBe("browser::invalid_query");
  expect(dataProperty(failurePayload(direct), "reason")).toBe("duplicate");
  // in+voice decodes to "in voice", not the selected key.
  const spaced = value(await setup("?in+voice=7").queryParameter("invoice"));
  expect(recordIdentity(spaced)).toBe(contracts.none);
});

test("malformed escapes and invalid UTF-8 fail on any pair", async () => {
  for (const search of ["?invoice=%ZZ", "?invoice=%FF", "?invoice=%E2%82", "?%ZZ=1"]) {
    const result = await setup(search).queryParameter("invoice");
    expect(failureName(result)).toBe("browser::invalid_query");
    expect(dataProperty(failurePayload(result), "key")).toBe("invoice");
    expect(dataProperty(failurePayload(result), "reason")).toBe("malformed");
  }
  const foreign = await setup("?other=%ZZ&invoice=7").queryParameter("invoice");
  expect(failureName(foreign)).toBe("browser::invalid_query");
  expect(dataProperty(failurePayload(foreign), "reason")).toBe("malformed");
  const illFormed = await setup("?invoice=\uD800").queryParameter("invoice");
  expect(failureName(illFormed)).toBe("browser::invalid_query");
  expect(dataProperty(failurePayload(illFormed), "reason")).toBe("malformed");
});

test("raw and decoded budgets reject oversized queries", async () => {
  const fitting = `?${"a".repeat(8191)}`;
  expect(new TextEncoder().encode(fitting).byteLength).toBe(8192);
  expect((await setup(fitting).queryParameter("invoice")).kind).toBe("ok");
  const oversized = `?${"a".repeat(8192)}`;
  const raw = await setup(oversized).queryParameter("invoice");
  expect(failureName(raw)).toBe("browser::invalid_query");
  expect(dataProperty(failurePayload(raw), "reason")).toBe("too_large");
  const selected = `?invoice=${"x".repeat(257)}`;
  const over = await setup(selected).queryParameter("invoice");
  expect(failureName(over)).toBe("browser::invalid_query");
  expect(dataProperty(failurePayload(over), "reason")).toBe("too_large");
  const edge = value(await setup(`?invoice=${"x".repeat(256)}`).queryParameter("invoice"));
  expect(dataProperty(edge, "value")).toBe("x".repeat(256));
  const wide = `?other=${"é".repeat(129)}&invoice=7`;
  const foreign = await setup(wide).queryParameter("invoice");
  expect(failureName(foreign)).toBe("browser::invalid_query");
  expect(dataProperty(failurePayload(foreign), "reason")).toBe("too_large");
});

test("literal keys re-check at runtime and failures echo the key", async () => {
  for (const key of ["Invoice", "", "1voice", "in-voice", "in voice", "a".repeat(65)]) {
    const result = await setup("?invoice=7").queryParameter(key);
    expect(failureName(result)).toBe("browser::invalid_query");
    expect(dataProperty(failurePayload(result), "key")).toBe(key);
    expect(dataProperty(failurePayload(result), "reason")).toBe("key");
  }
  await expect(setup("?invoice=7").queryParameter(7n)).rejects.toThrow("invalid browser string");
});

test("host location wins over ambient location, absence reads empty", async () => {
  const ambient = (globalThis as { location?: unknown }).location;
  (globalThis as { location?: unknown }).location = { search: "?invoice=ambient" };
  try {
    const fromAmbient = value(await createBrowser(domain, contracts).queryParameter("invoice"));
    expect(dataProperty(fromAmbient, "value")).toBe("ambient");
    const fromHost = value(await setup("?invoice=host").queryParameter("invoice"));
    expect(dataProperty(fromHost, "value")).toBe("host");
  } finally {
    if (ambient === undefined) delete (globalThis as { location?: unknown }).location;
    else (globalThis as { location?: unknown }).location = ambient;
  }
  const missing = value(await createBrowser(domain, contracts).queryParameter("invoice"));
  expect(recordIdentity(missing)).toBe(contracts.none);
});
