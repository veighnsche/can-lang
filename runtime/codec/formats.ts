import { Budget, reject, standaloneBytes } from "./budget.ts";
import { decodeText } from "./document.ts";
import { projectValue, type Schema, type IntPolicy } from "./project.ts";

// Parsed-value integers: the native TOML/YAML/JSON5 parsers hand over plain
// doubles, so only finite safe integers project. Unsafe tokens still fail
// closed: every rounded value sits at or above 2^53 (no representable
// double lies between MAX_SAFE_INTEGER and 2^53), and TOML additionally
// fails 9007199254740993 natively before projection runs.
const parsedInt: IntPolicy = (value, _holder, _key, path, budget) => {
  if (typeof value !== "number" || !Number.isSafeInteger(value)) reject(path, "integer_token");
  budget.charge(String(value).length, path);
  return BigInt(value);
};
const dummy: object = Object.create(null);

function projectFormat(
  schema: Schema,
  input: unknown,
  parse: (text: string) => unknown,
  reason: "invalid_toml" | "invalid_yaml" | "invalid_json5",
  bytes: number,
): unknown {
  const text = decodeText(input, bytes);
  const budget = new Budget(bytes);
  let parsed: unknown;
  try {
    parsed = parse(text);
  } catch (cause) {
    if (cause instanceof Error) reject("", reason);
    throw cause;
  }
  return projectValue(schema, parsed, dummy, budget, parsedInt);
}

export function decodeToml(schema: Schema, input: unknown, bytes = standaloneBytes): unknown {
  return projectFormat(schema, input, (text) => Bun.TOML.parse(text), "invalid_toml", bytes);
}
export function decodeYaml(schema: Schema, input: unknown, bytes = standaloneBytes): unknown {
  return projectFormat(schema, input, (text) => Bun.YAML.parse(text), "invalid_yaml", bytes);
}
export function decodeJson5(schema: Schema, input: unknown, bytes = standaloneBytes): unknown {
  return projectFormat(schema, input, (text) => Bun.JSON5.parse(text), "invalid_json5", bytes);
}
