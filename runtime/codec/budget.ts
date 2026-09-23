// These are private codec controls. Authored APIs cannot raise the fixed depth
// or node limits; transport adapters may replace only the byte limit.
export const standaloneBytes = 8_388_608;
export const maxDepth = 64;
export const maxNodes = 1_000_000;
export type Reason =
  | "utf8"
  | "unicode_scalar"
  | "byte_range"
  | "invalid_json"
  | "invalid_toml"
  | "invalid_yaml"
  | "invalid_json5"
  | "duplicate_member"
  | "missing_member"
  | "extra_member"
  | "type"
  | "integer_token"
  | "nonfinite"
  | "variant_tag"
  | "cycle"
  | "depth_limit"
  | "node_limit"
  | "byte_limit"
  | "media_type"
  | "charset";
export class CodecIssue extends Error {
  constructor(
    readonly path: string,
    readonly reason: Reason,
  ) {
    super(reason);
  }
}
export function reject(path: string, reason: Reason): never {
  throw new CodecIssue(path, reason);
}
export class Budget {
  remaining: number;
  nodes = 0;
  constructor(readonly bytes = standaloneBytes) {
    if (!Number.isSafeInteger(bytes) || bytes < 1 || bytes > 67_108_864)
      throw new TypeError("invalid private codec budget");
    this.remaining = bytes;
  }
  charge(size: number, path: string): void {
    if (!Number.isSafeInteger(size) || size < 0) throw new TypeError("invalid codec charge");
    if (size > this.remaining) reject(path, "byte_limit");
    this.remaining -= size;
  }
  visit(depth: number, path: string): void {
    if (depth > maxDepth) reject(path, "depth_limit");
    if (++this.nodes > maxNodes) reject(path, "node_limit");
  }
}
export function childPath(parent: string, key: string | number): string {
  return parent + "/" + String(key).replaceAll("~", "~0").replaceAll("/", "~1");
}
