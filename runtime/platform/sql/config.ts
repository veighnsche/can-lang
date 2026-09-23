// Dialect-specific native option validation. Each pool constructor proves
// its options before any native client exists: wrong shapes are compiler
// defects and throw, while out-of-range values fail as connection errors
// so misconfiguration never reaches the network or filesystem.
import type { Completion } from "../../completion.ts";
import type { SQLFailures } from "./errors.ts";

export function postgresMaxConnections(value: unknown, failures: SQLFailures): { ok: true; max: number } | { ok: false; failure: Completion<never> } {
  if (typeof value !== "bigint") throw new TypeError("invalid compiler sql config");
  if (value < 1n || value > 2147483647n) return { ok: false, failure: failures.connectionFailed("config") };
  return { ok: true, max: Number(value) };
}
