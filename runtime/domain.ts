// Compiler-private typed domain errors. Plans are emitted checked declarations,
// never authored runtime registrations or untrusted wire schemas.
import { createHash } from "node:crypto";
import { catalogue, catalogueTypeShapes } from "./catalogue.ts";
import { dataArray, dataKeys, dataProperty, recordIdentity } from "./data.ts";
import {
  allocateOccurrenceID,
  freezeProvenance,
  isStandardFailure,
  type FailureOrigin,
  type FailureProvenance,
} from "./failure.ts";

export type ErrorDeclaration = Readonly<{
  identity: string;
  name: string;
  parameters: number;
}>;
export type FailureShape = Readonly<{
  identity: string;
  kind: string;
  declaration?: string;
  arguments: readonly string[];
  fields: readonly Readonly<{ name: string; type: string }>[];
  leaves: readonly string[];
  element?: string;
  result?: string;
  inputs: readonly string[];
  errors: readonly string[];
}>;
export type ErrorPlan = Readonly<{
  declarations: readonly ErrorDeclaration[];
  shapes: readonly FailureShape[];
}>;
type Descriptor = Readonly<{ name: string; arguments: readonly Descriptor[] | null }>;
type CatalogueShape = Readonly<{
  name: string;
  identity: string;
  kind: string;
  parameters: readonly { name: string; constraint: string }[];
  fields: readonly { name: string; type: Descriptor }[];
  leaves: readonly Descriptor[];
}>;
declare const domainBrand: unique symbol;
export type DomainFailure = Readonly<{ readonly [domainBrand]: true }>;
export type DomainDetails = Readonly<{
  occurrenceID: bigint;
  declaration: ErrorDeclaration;
  typeIdentity: string;
  typeArguments: readonly string[];
  payload: unknown;
  cause: unknown;
  origin: FailureOrigin;
  provenance: FailureProvenance;
}>;
const occurrences = new WeakMap<object, DomainDetails>();
const objectLike = (value: unknown): value is object =>
  value !== null && (typeof value === "object" || typeof value === "function");
export const isDomainFailure = (value: unknown): value is DomainFailure =>
  objectLike(value) && occurrences.has(value);
export function domainFailureDiagnostics(value: DomainFailure): DomainDetails {
  const found = objectLike(value) ? occurrences.get(value) : undefined;
  if (!found) throw new TypeError("invalid domain occurrence");
  return found;
}
export type OpaqueAdmission = (typeIdentity: string, value: unknown) => boolean;

export function createDomainRuntime(
  plan: ErrorPlan,
  opaqueAdmission?: OpaqueAdmission,
  callableAdmission?: OpaqueAdmission,
) {
  const declarations = new Map<string, ErrorDeclaration>();
  for (const source of plan.declarations) {
    const d = Object.freeze({ ...source });
    if (
      typeof d.identity !== "string" ||
      typeof d.name !== "string" ||
      !Number.isInteger(d.parameters) ||
      d.parameters < 0 ||
      declarations.has(d.identity)
    )
      throw new TypeError("invalid or duplicate error declaration");
    const builtin = catalogue.errors.find((e) => e.identity === d.identity);
    if (builtin) {
      if (builtin.name !== d.name || builtin.parameters.length !== d.parameters)
        throw new TypeError("catalogue declaration mismatch");
    } else if (!d.identity.startsWith("can.project.") || !d.name.includes("::"))
      throw new TypeError("invalid project error declaration");
    declarations.set(d.identity, d);
  }
  const shapes = new Map<string, FailureShape>();
  for (const s of plan.shapes) {
    if (shapes.has(s.identity)) throw new TypeError("duplicate concrete type identity");
    const shape = Object.freeze({
      ...s,
      arguments: Object.freeze([...s.arguments]),
      fields: Object.freeze(s.fields.map((f) => Object.freeze({ ...f }))),
      leaves: Object.freeze([...s.leaves]),
      inputs: Object.freeze([...s.inputs]),
      errors: Object.freeze([...s.errors]),
    });
    shapes.set(shape.identity, shape);
  }
  const definitions = catalogueTypeShapes as readonly CatalogueShape[];
  const byName = new Map(definitions.map((d) => [d.name, d]));
  const byIdentity = new Map(definitions.map((d) => [d.identity, d]));
  const get = (id: string | undefined): FailureShape => {
    const s = id ? shapes.get(id) : undefined;
    if (!s) throw new TypeError("missing concrete failure type");
    return s;
  };
  function matches(
    id: string | undefined,
    descriptor: Descriptor,
    env: Map<string, string>,
  ): boolean {
    if (env.has(descriptor.name)) return id === env.get(descriptor.name);
    const s = get(id);
    const args = descriptor.arguments ?? [];
    if (descriptor.name === "[]")
      return s.kind === "array" && args.length === 1 && matches(s.element, args[0], env);
    const builtin = byName.get(descriptor.name);
    if (!builtin)
      return s.kind === "primitive" && s.declaration === descriptor.name && args.length === 0;
    return (
      s.declaration === builtin.identity &&
      s.arguments.length === args.length &&
      args.every((a, i) => matches(s.arguments[i], a, env))
    );
  }
  for (const s of shapes.values()) {
    let key: unknown;
    if (s.kind === "array") key = ["array", "", get(s.element).identity];
    else if (s.kind === "callable" || s.kind === "choice_arm")
      key = [s.kind, get(s.result).identity, [...s.inputs], [...s.errors]];
    else key = [s.kind, s.declaration ?? "", ...s.arguments];
    const expected = createHash("sha256")
      .update("can-concrete-type-v1\0" + JSON.stringify(key))
      .digest("hex");
    if (expected !== s.identity) throw new TypeError("concrete failure type identity mismatch");
    for (const id of [
      ...s.arguments,
      ...s.inputs,
      ...s.errors,
      ...s.leaves,
      ...s.fields.map((f) => f.type),
    ])
      get(id);
    if (new Set(s.fields.map((f) => f.name)).size !== s.fields.length)
      throw new TypeError("duplicate payload field");
    if (
      s.kind === "variant" &&
      (s.leaves.length === 0 ||
        new Set(s.leaves).size !== s.leaves.length ||
        s.leaves.some((id) => {
          const leaf = get(id);
          return (
            leaf.kind !== "record" &&
            leaf.kind !== "error" &&
            !(leaf.kind === "opaque" && leaf.declaration === "can.prelude@1::standard_failure")
          );
        }))
    )
      throw new TypeError("invalid concrete variant leaves");
    const builtin = s.declaration ? byIdentity.get(s.declaration) : undefined;
    if (builtin) {
      if (
        s.kind !== builtin.kind ||
        s.arguments.length !== builtin.parameters.length ||
        s.fields.length !== builtin.fields.length
      )
        throw new TypeError("catalogue payload shape mismatch");
      const env = new Map(builtin.parameters.map((p, i) => [p.name, s.arguments[i]]));
      if (
        !builtin.fields.every(
          (f, i) => s.fields[i].name === f.name && matches(s.fields[i].type, f.type, env),
        )
      )
        throw new TypeError("catalogue payload field mismatch");
      if (
        s.leaves.length !== builtin.leaves.length ||
        !builtin.leaves.every((leaf) => s.leaves.some((id) => matches(id, leaf, env)))
      )
        throw new TypeError("catalogue variant leaves mismatch");
      for (let i = 0; i < builtin.parameters.length; i++) {
        const arg = get(s.arguments[i]);
        const constraint = builtin.parameters[i].constraint;
        if (
          constraint === "map_key" &&
          !(arg.kind === "primitive" && ["int", "bool", "str"].includes(arg.declaration ?? ""))
        )
          throw new TypeError("invalid map key specialization");
        if (constraint === "failure_variant" && arg.kind !== "variant")
          throw new TypeError("invalid failure variant specialization");
        if (arg.kind === "void") throw new TypeError("void generic payload argument");
      }
    } else if (
      (s.kind === "record" || s.kind === "error" || s.kind === "variant" || s.kind === "opaque") &&
      !s.declaration?.startsWith("can.project.")
    )
      throw new TypeError("unknown nominal payload declaration");
    if (s.kind === "error") {
      const d = declarations.get(s.declaration!);
      if (!d || d.parameters !== s.arguments.length)
        throw new TypeError("unallocated error payload type");
    }
  }
  // Payload checks use only descriptors on non-proxy data. Iterative traversal
  // rejects cycles without recursive JS stack growth; completed shared subtrees
  // are memoized by object and concrete type identity.
  function accepts(typeIdentity: string, value: unknown): boolean {
    type Work = { type: string; value: unknown; exit?: boolean };
    const work: Work[] = [{ type: typeIdentity, value }];
    const active = new WeakSet<object>();
    const checked = new WeakMap<object, Set<string>>();
    try {
      while (work.length) {
        const entry = work.pop()!;
        const s = get(entry.type);
        const v = entry.value;
        if (entry.exit) {
          active.delete(v as object);
          let seen = checked.get(v as object);
          if (!seen) {
            seen = new Set();
            checked.set(v as object, seen);
          }
          seen.add(entry.type);
          continue;
        }
        if (s.kind === "primitive") {
          const expected =
            s.declaration === "int"
              ? "bigint"
              : s.declaration === "float"
                ? "number"
                : s.declaration === "bool"
                  ? "boolean"
                  : s.declaration === "str"
                    ? "string"
                    : "invalid";
          if (typeof v !== expected) return false;
          continue;
        }
        if (s.kind === "variant") {
          const nominal = recordIdentity(v);
          const leaf = s.leaves.find(
            (id) =>
              id === nominal ||
              (get(id).declaration === "can.prelude@1::standard_failure" && isStandardFailure(v)),
          );
          if (!leaf) return false;
          work.push({ type: leaf, value: v });
          continue;
        }
        if (s.kind === "opaque") {
          if (s.declaration === "can.prelude@1::standard_failure") {
            if (!isStandardFailure(v)) return false;
          } else if (!opaqueAdmission?.(s.identity, v)) return false;
          continue;
        }
        if (s.kind === "callable" || s.kind === "choice_arm") {
          if (!callableAdmission?.(s.identity, v)) return false;
          continue;
        }
        if (!objectLike(v) || active.has(v)) return false;
        if (checked.get(v)?.has(s.identity)) continue;
        const keys = dataKeys(v);
        if (!Object.isFrozen(v)) return false;
        active.add(v);
        work.push({ type: s.identity, value: v, exit: true });
        if (s.kind === "array") {
          const values = dataArray(v);
          for (let i = values.length - 1; i >= 0; i--)
            work.push({ type: s.element!, value: values[i] });
        } else if (s.kind === "record" || s.kind === "error") {
          if (
            recordIdentity(v) !== s.identity ||
            keys.filter((k) => typeof k === "string").length !== s.fields.length
          )
            return false;
          for (let i = s.fields.length - 1; i >= 0; i--) {
            const f = s.fields[i];
            work.push({ type: f.type, value: dataProperty(v, f.name) });
          }
        } else return false;
      }
      return true;
    } catch {
      return false;
    }
  }
  function create(
    typeIdentity: string,
    payload: unknown,
    origin: FailureOrigin,
    cause?: unknown,
    provenance?: unknown,
  ): DomainFailure {
    const s = get(typeIdentity);
    const declaration = declarations.get(s.declaration ?? "");
    if (s.kind !== "error" || !declaration || !accepts(typeIdentity, payload))
      throw new TypeError("invalid domain error payload");
    const token = Object.freeze(Object.create(null));
    occurrences.set(
      token,
      Object.freeze({
        occurrenceID: allocateOccurrenceID(),
        declaration,
        typeIdentity,
        typeArguments: Object.freeze([...s.arguments]),
        payload,
        cause,
        origin: Object.freeze({ ...origin, invocation: Object.freeze([...origin.invocation]) }),
        provenance: freezeProvenance(provenance),
      }),
    );
    return token;
  }
  function checkBound(occurrence: DomainFailure, allowed: readonly string[]): DomainFailure {
    const details = domainFailureDiagnostics(occurrence);
    const known = declarations.get(details.declaration.identity);
    if (
      !known ||
      known.name !== details.declaration.name ||
      known.parameters !== details.declaration.parameters ||
      !allowed.includes(details.typeIdentity)
    )
      throw new TypeError("undeclared escaping domain error");
    return occurrence;
  }
  return Object.freeze({ accepts, create, checkBound });
}
