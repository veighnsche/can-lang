// Browser-profile typed domain errors. Concrete type identities are sealed by
// one async WebCrypto verification over the identical canonical input bytes
// before any domain use; synchronous construction admits only verified plans.
// There is no sync hash and no handwritten digest in this profile.
import {
  concreteTypeDigestInput,
  createDomainRuntimeWithDigest,
  shapeIdentityKey,
  type ErrorPlan,
  type FailureShape,
  type OpaqueAdmission,
} from "../domain-core.ts";

export type {
  DomainDetails,
  DomainFailure,
  DomainRuntime,
  ErrorDeclaration,
  ErrorPlan,
  FailureShape,
  IdentityDigest,
  OpaqueAdmission,
} from "../domain-core.ts";
export {
  concreteTypeDigestInput,
  createDomainRuntimeWithDigest,
  domainFailureDiagnostics,
  isDomainFailure,
  shapeIdentityKey,
} from "../domain-core.ts";

const verified = new WeakSet<object>();
const trustingDigest = (_input: string, claimed: string): string => claimed;
const encoder = new TextEncoder();

function bytesToHex(buffer: ArrayBuffer): string {
  return [...new Uint8Array(buffer)].map((byte) => byte.toString(16).padStart(2, "0")).join("");
}

async function webDigestHex(input: string): Promise<string> {
  const subtle = globalThis.crypto?.subtle;
  if (!subtle) throw new TypeError("browser identity verification requires crypto.subtle");
  return bytesToHex(await subtle.digest("SHA-256", encoder.encode(input)));
}

// Verify every sealed concrete identity, then validate declarations, shapes
// and catalogue agreement exactly like the canonical constructor. Identity
// mismatches report before structural violations when a plan has both.
export async function verifyErrorPlan(plan: ErrorPlan): Promise<ErrorPlan> {
  if (plan === null || typeof plan !== "object" || !Array.isArray(plan.shapes))
    throw new TypeError("invalid error plan");
  const shapes = new Map<string, FailureShape>();
  for (const shape of plan.shapes) {
    if (shape === null || typeof shape !== "object" || typeof shape.identity !== "string")
      throw new TypeError("invalid concrete failure type");
    shapes.set(shape.identity, shape);
  }
  const resolve = (id: string | undefined): FailureShape => {
    const found = id ? shapes.get(id) : undefined;
    if (!found) throw new TypeError("missing concrete failure type");
    return found;
  };
  for (const shape of shapes.values()) {
    let input: string;
    try {
      input = concreteTypeDigestInput(shapeIdentityKey(shape, resolve));
    } catch {
      throw new TypeError("concrete failure type identity mismatch");
    }
    if ((await webDigestHex(input)) !== shape.identity)
      throw new TypeError("concrete failure type identity mismatch");
  }
  createDomainRuntimeWithDigest(plan, trustingDigest);
  verified.add(plan);
  return plan;
}

export function createDomainRuntime(
  plan: ErrorPlan,
  opaqueAdmission?: OpaqueAdmission,
  callableAdmission?: OpaqueAdmission,
) {
  if (plan === null || typeof plan !== "object" || !verified.has(plan))
    throw new TypeError("unverified error plan");
  return createDomainRuntimeWithDigest(plan, trustingDigest, opaqueAdmission, callableAdmission);
}
