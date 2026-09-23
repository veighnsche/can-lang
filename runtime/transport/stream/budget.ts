// Private byte/item/queue limits for pull streams. Pure accounting only:
// adapters map violations onto their declared limit failures, and the owner
// keeps every lifecycle rule. This module never touches owner.ts.
export function validMaxItems(max: unknown): max is bigint {
  return typeof max === "bigint" && max >= 1n;
}
export function validMaxBytes(max: unknown): max is bigint {
  return typeof max === "bigint" && max >= 1n;
}
// Count before retaining so a growing source cannot evade the cap.
export function overCap(used: bigint, add: bigint, cap: bigint): boolean {
  return add < 0n || used < 0n || add > cap - used;
}
export function fitsChunk(length: bigint, remaining: bigint): boolean {
  return length >= 0n && length <= remaining;
}
