import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createForm } from "../platform/form.ts";
import { value, type Completion } from "../completion.ts";
import { record, dataProperty } from "../data.ts";
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
  ["form::unknown_field", "form::invalid_name"].includes(e.name),
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
const form = createForm(domain, {
  unknownField: errors[0]!.identity,
  invalidName: errors[1]!.identity,
});
function check(result: Completion, name: string): string {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error();
  const diagnostics = domainFailureDiagnostics(result.value);
  expect(diagnostics.declaration.name).toBe(name);
  return dataProperty(diagnostics.payload ?? result.value, "name") as string;
}
test("checked builders mint tokens for valid names", async () => {
  const coll = value(await form.namedCollection("lines"));
  const field = value(await form.namedField("sku"));
  const row = value(await form.rowKey("a1-_B"));
  expect(value(await form.orderName(coll))).toBe("lines_order");
  expect(value(await form.fieldName(field))).toBe("sku");
  expect(value(await form.inputName(coll, row, field))).toBe("lines[a1-_B][sku]");
});
test("checked builders reject invalid spellings with the offending name", async () => {
  for (const bad of ["Lines", "has space", "bracket[", "", "lines_order!"]) {
    expect(check(await form.namedCollection(bad), "form::unknown_field")).toBe(bad);
    expect(check(await form.namedField(bad), "form::unknown_field")).toBe(bad);
  }
  for (const bad of ["", "a b", "a[b]", "a/b", "x".repeat(65)]) {
    expect(check(await form.rowKey(bad), "form::invalid_name")).toBe(bad);
  }
});
test("renderers refuse forged tokens and mistyped names", async () => {
  const coll = value(await form.namedCollection("lines"));
  const row = value(await form.rowKey("a"));
  const field = value(await form.namedField("sku"));
  for (const forged of [record("forged", []), "lines", null, undefined, 7]) {
    await expect(form.orderName(forged)).rejects.toThrow();
    await expect(form.fieldName(forged)).rejects.toThrow();
    await expect(form.inputName(forged, row, field)).rejects.toThrow();
    await expect(form.inputName(coll, forged, field)).rejects.toThrow();
    await expect(form.inputName(coll, row, forged)).rejects.toThrow();
  }
  // Tokens are single-sorted: a row is not a collection.
  await expect(form.orderName(row)).rejects.toThrow();
  await expect(form.namedCollection(7)).rejects.toThrow(TypeError);
  await expect(form.rowKey(null)).rejects.toThrow(TypeError);
});
