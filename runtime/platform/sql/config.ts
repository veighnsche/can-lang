// Dialect-specific native option validation. Each pool constructor proves
// its options before any native client exists: wrong shapes are compiler
// defects and throw, while out-of-range values fail as connection errors
// so misconfiguration never reaches the network or filesystem.
import type { Completion } from "../../completion.ts";
import { dataProperty } from "../../data.ts";
import type { SQLFailures } from "./errors.ts";

export function poolMaxConnections(value: unknown, failures: SQLFailures): { ok: true; max: number } | { ok: false; failure: Completion<never> } {
  if (typeof value !== "bigint") throw new TypeError("invalid compiler sql config");
  if (value < 1n || value > 2147483647n) return { ok: false, failure: failures.connectionFailed("config") };
  return { ok: true, max: Number(value) };
}

export type SQLiteFileConfig = {
  readonly filename: string;
  readonly mode: "ro" | "rw" | "rwc";
  readonly busyTimeoutMs: number;
};

export function sqliteFileConfig(path: unknown, options: unknown, failures: SQLFailures): { ok: true; config: SQLiteFileConfig } | { ok: false; failure: Completion<never> } {
  if (typeof path !== "string") throw new TypeError("invalid compiler sql path");
  if (path === "" || path.includes("\0")) return { ok: false, failure: failures.connectionFailed("config") };
  const mode = dataProperty(options, "mode");
  if (mode !== "ro" && mode !== "rw" && mode !== "rwc") return { ok: false, failure: failures.connectionFailed("config") };
  const timeout = dataProperty(options, "busy_timeout_ms");
  if (typeof timeout !== "bigint") throw new TypeError("invalid compiler sql config");
  if (timeout < 0n || timeout > 2147483647n) return { ok: false, failure: failures.connectionFailed("config") };
  return { ok: true, config: { filename: path, mode, busyTimeoutMs: Number(timeout) } };
}
