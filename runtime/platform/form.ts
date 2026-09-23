import { record, array } from "../data.ts";
import { copyBytes, byteLength } from "../bytes.ts";
import { reject, childPath, Budget } from "../codec/budget.ts";
export type FormSchema = Readonly<{
  root: string;
  fields: readonly Readonly<{ name: string; kind: string; some?: string; none?: string }>[];
}>;
export class FormIssue extends Error {
  constructor(readonly reason: "invalid_form_encoding" | "form_missing" | "form_repeated") {
    super("invalid form");
  }
}
const origin = Object.freeze({
  source: "can:form",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
// Validation delegates UTF-8 percent decoding to the native operation before
// URLSearchParams applies its otherwise replacement-tolerant decoding policy.
export function strictParameters(
  text: string,
  encodingReason: "invalid_form_encoding" = "invalid_form_encoding",
): URLSearchParams {
  if (/%(?![0-9a-fA-F]{2})/.test(text)) throw new FormIssue(encodingReason);
  for (const pair of text.split("&")) {
    if (pair === "") continue;
    const separator = pair.indexOf("="),
      key = separator < 0 ? pair : pair.slice(0, separator),
      value = separator < 0 ? "" : pair.slice(separator + 1);
    let name: string;
    try {
      name = decodeURIComponent(key.replaceAll("+", " "));
    } catch (cause) {
      if (!(cause instanceof URIError)) throw cause;
      reject("", "utf8");
    }
    try {
      decodeURIComponent(value.replaceAll("+", " "));
    } catch (cause) {
      if (!(cause instanceof URIError)) throw cause;
      reject(childPath("", name!), "utf8");
    }
  }
  return new URLSearchParams(text);
}
export function decodeForm(schema: FormSchema, input: unknown, limit: number): unknown {
  if (byteLength(input) > BigInt(limit)) reject("", "byte_limit");
  const budget = new Budget(Math.max(1, limit));
  budget.visit(1, "");
  let text: string;
  try {
    text = new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(
      copyBytes(input, origin),
    );
  } catch (cause) {
    if (!(cause instanceof TypeError)) throw cause;
    reject("", "utf8");
  }
  const params = strictParameters(text),
    known = new Set(schema.fields.map((f) => f.name));
  for (const name of params.keys()) if (!known.has(name)) reject(childPath("", name), "type");
  const fields: [string, unknown][] = [];
  for (const field of schema.fields) {
    const values = params.getAll(field.name),
      path = childPath("", field.name);
    budget.visit(field.kind === "str" ? 1 : 2, path);
    if (field.kind === "array") {
      for (let i = 0; i < values.length; i++) budget.visit(2, childPath(path, i));
      fields.push([field.name, array(values)]);
      continue;
    }
    if (values.length > 1) throw new FormIssue("form_repeated");
    if (field.kind === "str") {
      if (values.length === 0) throw new FormIssue("form_missing");
      fields.push([field.name, values[0]]);
    } else if (field.kind === "optional") {
      if (!field.some || !field.none) throw new TypeError("invalid compiler form schema");
      budget.visit(3, path);
      if (values.length !== 0) budget.visit(3, childPath(path, "value"));
      fields.push([
        field.name,
        values.length === 0 ? record(field.none, []) : record(field.some, [["value", values[0]]]),
      ]);
    } else throw new TypeError("unknown compiler form field");
  }
  // Typed fields were derived by the compiler; shared codec error paths and
  // node/depth budgets apply without serializing a second JSON representation.
  return record(schema.root, fields);
}
