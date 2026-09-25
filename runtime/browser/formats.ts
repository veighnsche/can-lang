// Browser-profile codec format decoders. TOML, YAML and JSON5 have no
// browser-native parser, so the capability gate rejects those operations
// at check time with span evidence. The stubs below only satisfy linking
// and fail closed when reached, like scopeRequest in browser production.
import { standaloneBytes } from "../codec/budget.ts";
import type { Schema } from "../codec/project.ts";

const unavailable = (format: string): never => {
  throw new TypeError(`${format} parsing is unavailable in the browser profile`);
};

export function decodeToml(
  _schema: Schema,
  _input: unknown,
  _bytes: number = standaloneBytes,
): unknown {
  return unavailable("TOML");
}

export function decodeYaml(
  _schema: Schema,
  _input: unknown,
  _bytes: number = standaloneBytes,
): unknown {
  return unavailable("YAML");
}

export function decodeJson5(
  _schema: Schema,
  _input: unknown,
  _bytes: number = standaloneBytes,
): unknown {
  return unavailable("JSON5");
}
