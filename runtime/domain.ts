// Compiler-private typed domain errors: Bun host binding over the canonical
// core. Identity verification hashes synchronously with Node crypto; the
// browser profile overlays runtime/browser/domain.ts instead.
import { createHash } from "node:crypto";
import {
  createDomainRuntimeWithDigest,
  type ErrorPlan,
  type OpaqueAdmission,
} from "./domain-core.ts";

export type {
  DomainDetails,
  DomainFailure,
  DomainRuntime,
  ErrorDeclaration,
  ErrorPlan,
  FailureShape,
  IdentityDigest,
  OpaqueAdmission,
} from "./domain-core.ts";
export {
  concreteTypeDigestInput,
  createDomainRuntimeWithDigest,
  domainFailureDiagnostics,
  isDomainFailure,
  shapeIdentityKey,
} from "./domain-core.ts";

const nodeDigest = (input: string): string => createHash("sha256").update(input).digest("hex");

export function createDomainRuntime(
  plan: ErrorPlan,
  opaqueAdmission?: OpaqueAdmission,
  callableAdmission?: OpaqueAdmission,
) {
  return createDomainRuntimeWithDigest(
    plan,
    (input) => nodeDigest(input),
    opaqueAdmission,
    callableAdmission,
  );
}
