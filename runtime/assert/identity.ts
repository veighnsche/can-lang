// Digest-based callable identity constructors over node:crypto. Portable
// lineage (frames, paths, ordering) lives in ./lineage.ts so browser
// code can track invocations without hashing; only the server harness
// mints digests, through the reservation functions that preserve exact
// occurrence semantics.
import { createHash } from "node:crypto";
import {
  registerCallableInstance,
  reserveCallableOccurrence,
  siteParts,
  type CallableIdentity,
  type InvocationIdentity,
} from "./lineage.ts";

export type { CallableIdentity } from "./lineage.ts";

export function callableIdentity(
  parentIdentity: InvocationIdentity,
  site: string,
  captures: readonly unknown[],
): CallableIdentity {
  const reserved = reserveCallableOccurrence(parentIdentity, site);
  // The digest identifies the creation path, not a serialization of private
  // receiver/near values. Frozen capture references stay in the private receipt.
  const key = createHash("sha256")
    .update(
      "can-callable-instance-v1\0" + JSON.stringify([reserved.path, site, reserved.occurrence]),
    )
    .digest("hex");
  return registerCallableInstance(reserved.family, site, reserved.occurrence, key, captures);
}

// Pure initialization precedes assertion roots. These immutable creation
// receipts may be embedded in multiple isolated roots; they own no queue state.
const initializationOccurrences = new Map<string, number>();
export function initializationCallableIdentity(
  site: string,
  captures: readonly unknown[],
): CallableIdentity {
  siteParts(site);
  const visit = initializationOccurrences.get(site) ?? 0;
  if (!Number.isSafeInteger(visit + 1)) throw new TypeError("callable occurrence overflow");
  initializationOccurrences.set(site, visit + 1);
  const key = createHash("sha256")
    .update("can-initialization-callable-v1\0" + JSON.stringify([site, visit]))
    .digest("hex");
  return registerCallableInstance(null, site, visit, key, captures);
}
