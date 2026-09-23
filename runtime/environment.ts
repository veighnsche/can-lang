// Startup controls never reach Bun. Approved env/auth adapters read the caller
// snapshot instead of process.env, whose HOME/config values belong to the driver.
import { readFileSync, closeSync } from "node:fs";
const source = JSON.parse(readFileSync(3, "utf8"));
closeSync(3);
if (
  source === null ||
  typeof source !== "object" ||
  Array.isArray(source) ||
  Object.values(source).some((value) => typeof value !== "string")
) {
  throw new Error("invalid launcher environment snapshot");
}
const values: Readonly<Record<string, string>> = Object.freeze(
  Object.assign(Object.create(null), source),
);
export function originalEnvironment(name: string): string | undefined {
  return Object.hasOwn(values, name) ? values[name] : undefined;
}
