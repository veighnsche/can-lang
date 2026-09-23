// Compiler-private wrapper policy dispatch. Base invocations run untouched;
// assertion rows may inject one origin-tagged failure that skips the wrapped
// operation and starts policy lookup from a fresh private occurrence.
// Injection is reported as policy-fixture evidence, never request evidence.
import {
  contextOwner,
  fixtureIndex,
  policyFixtureEvidence,
  registerFixtureTable,
  violation,
  type AssertionContext,
} from "./context.ts";
import { record } from "../data.ts";
import { domainFailureDiagnostics, isDomainFailure, type createDomainRuntime } from "../domain.ts";
import { type FailureOrigin } from "../failure.ts";
import { failure, type Completion } from "../completion.ts";
type Domain = ReturnType<typeof createDomainRuntime>;
export type PolicyKey = Readonly<{ origin: "native" | "emitted"; identity: string }>;
const pending = new WeakMap<object, Map<string, Completion<never>>>();
export function provideInjection(
  context: AssertionContext,
  operation: string,
  origin: string,
  identity: string,
  payload: unknown,
  failed: string,
  domain: Domain,
  where: FailureOrigin,
): void {
  if (origin !== "native" && origin !== "emitted")
    throw violation(context, "malformed fixture", where);
  const owner = contextOwner(context);
  let table = pending.get(owner);
  if (table === undefined) {
    table = new Map();
    pending.set(owner, table);
  }
  if (table.has(operation)) throw violation(context, "malformed fixture", where);
  const completion =
    origin === "native"
      ? failure(
          domain.create(
            failed,
            record(failed, [["detail", payload]]),
            where,
            domain.create(identity, payload, where, undefined, { boundary: "native", operation }),
            { boundary: "native", operation },
          ),
        )
      : failure(
          domain.create(identity, payload, where, undefined, { boundary: "emitted", operation }),
        );
  table.set(operation, completion);
  registerFixtureTable(context, "can:policy:" + operation, 1, where);
}
export async function policyBase<T>(
  context: AssertionContext | undefined,
  operation: string,
  where: FailureOrigin,
  invoke: () => Promise<Completion<T>>,
): Promise<Completion<T>> {
  if (context === undefined) return invoke();
  const injected = pending.get(contextOwner(context))?.get(operation);
  if (injected === undefined) return invoke();
  fixtureIndex(context, "can:policy:" + operation, 1, where);
  policyFixtureEvidence(context);
  return injected as Completion<T>;
}
export function policyKey(completion: Completion<unknown>, failed: string): PolicyKey {
  const token = (completion as { value: unknown }).value;
  if (!isDomainFailure(token)) throw new TypeError("policy lookup needs a domain failure");
  const details = domainFailureDiagnostics(token);
  if (details.typeIdentity === failed && details.provenance.boundary === "native") {
    if (!isDomainFailure(details.cause))
      return { origin: "native", identity: details.typeIdentity };
    return { origin: "native", identity: domainFailureDiagnostics(details.cause).typeIdentity };
  }
  if (details.provenance.boundary === "native")
    return { origin: "native", identity: details.typeIdentity };
  return { origin: "emitted", identity: details.typeIdentity };
}
export function policyLeaf(completion: Completion<unknown>): unknown {
  const token = (completion as { value: unknown }).value;
  if (!isDomainFailure(token)) throw new TypeError("policy leaf needs a domain failure");
  const cause = domainFailureDiagnostics(token).cause;
  if (!isDomainFailure(cause)) throw new TypeError("policy leaf needs a normalized failure");
  return domainFailureDiagnostics(cause).payload;
}
