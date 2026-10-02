// NT-I12 lease-probe adapter contract: the probe envelope carries the
// merge-defined descriptor.probe_lease shape, the owner verdict maps to
// the nominal descriptor::lease_report record with a verbatim path
// echo, and non-completed outcomes return stale_handle
// (wrong-owner/stale-handle/closed-handle/not-found) or verbatim
// descriptor_fault kinds. A gone lease path is a completed outcome
// with released=true, not an error. Dispatch stays stubbed: the
// default transport throws, so nothing here claims live owner
// behavior.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import type { Completion } from "../completion.ts";
import { dataProperty } from "../data.ts";
import { createTestOwner } from "../test-support/owner.ts";
import { createLeaseProbe, type LeaseProbeDispatch } from "../test-support/slices/i12/lease.ts";

const identity = (kind: string, declaration: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration]))
    .digest("hex");
const scalar = (name: string): FailureShape => ({
  identity: identity("primitive", name),
  kind: "primitive",
  declaration: name,
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const text = scalar("str");
const integer = scalar("int");
const declarations = catalogue.errors.filter(
  (error) => error.name === "test::stale_handle" || error.name === "descriptor::descriptor_fault",
);
const shapeOf = (id: string, fields: readonly { name: string; type: string }[]): FailureShape => ({
  identity: identity("error", id),
  kind: "error",
  declaration: id,
  arguments: [],
  fields: fields.map((field) => ({
    name: field.name,
    type: field.type === "int" ? integer.identity : text.identity,
  })),
  leaves: [],
  inputs: [],
  errors: [],
});
const errorShapes: Map<string, FailureShape> = new Map(
  declarations.map((error) => [error.name, shapeOf(error.identity, error.fields)]),
);
const domain = createDomainRuntime({
  declarations: declarations.map((error) => ({
    identity: error.identity,
    name: error.name,
    parameters: 0,
  })),
  shapes: [text, integer, ...errorShapes.values()],
});
const err = (name: string): string => {
  const shape = errorShapes.get(name);
  if (shape === undefined) throw Error(`no shape for ${name}`);
  return shape.identity;
};

const typeId = (name: string): string => `test:type:${name}`;

type Captured = Readonly<Record<string, unknown>>;
function rig(handler: (request: Captured) => Readonly<Record<string, unknown>>) {
  const requests: Captured[] = [];
  const owner = createTestOwner(domain, { invalidGrant: err("test::stale_handle") });
  const probe = createLeaseProbe(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      descriptorFault: err("descriptor::descriptor_fault"),
      leaseReport: typeId("descriptor::lease_report"),
    },
    owner,
    (async (request) => {
      requests.push(request);
      return handler(request);
    }) as LeaseProbeDispatch,
  );
  return { owner, probe, requests };
}

const GRANT = "ng1-0123456789abcdef0123456789abcdef";
const LEASE_PATH = "/tmp/can-fd4.lease";

async function ok<T>(completion: Promise<Completion<T>>): Promise<T> {
  const result = await completion;
  expect(result.kind).toBe("ok");
  if (result.kind !== "ok") throw Error("expected ok");
  return result.value;
}

async function failed(completion: Promise<Completion<never>>) {
  const result = await completion;
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain failure");
  return domainFailureDiagnostics(result.value);
}

const field = (value: unknown, name: string): unknown => dataProperty(value, name);

function completed(facts: Record<string, unknown>) {
  return { outcome: "completed", kind: "ok", facts };
}

function leaseFacts(path: string, inherited: boolean, held: boolean, released: boolean) {
  return completed({ lease: { path, inherited, held, released } });
}

test("descriptor::probe_lease sends the envelope and maps the held verdict", async () => {
  const { owner, probe, requests } = rig(() => leaseFacts(LEASE_PATH, true, true, false));
  const handle = await ok(owner.admitGrant(GRANT));
  const report = await ok(probe.probeLease(handle, LEASE_PATH));
  expect(requests).toHaveLength(1);
  const request = requests[0] as Record<string, unknown>;
  expect(request["operation"]).toBe("descriptor.probe_lease");
  expect(request["schema_version"]).toBe("1");
  expect(request["owner_grant"]).toBe(GRANT);
  expect(request["arguments_digest"]).toBe(
    "sha256:d6cbf860266e84e1b1ba68f2d42e3819ff75db297cdc3c5b77a41414bd71e677",
  );
  expect(request["args"]).toEqual({ path: LEASE_PATH });
  expect(field(report, "path")).toBe(LEASE_PATH);
  expect(field(report, "inherited")).toBe(true);
  expect(field(report, "held")).toBe(true);
  expect(field(report, "released")).toBe(false);
});

test("descriptor::probe_lease maps a released path as a completed verdict", async () => {
  const { owner, probe } = rig(() => leaseFacts(LEASE_PATH, true, false, true));
  const handle = await ok(owner.admitGrant(GRANT));
  const report = await ok(probe.probeLease(handle, LEASE_PATH));
  expect(field(report, "held")).toBe(false);
  expect(field(report, "released")).toBe(true);
});

test("descriptor probe stale kinds return test::stale_handle", async () => {
  for (const kind of ["wrong-owner", "stale-handle", "closed-handle", "not-found"]) {
    const { owner, probe } = rig(() => ({
      outcome: "failed",
      kind,
      message: `${kind} here`,
      facts: {},
    }));
    const handle = await ok(owner.admitGrant(GRANT));
    const stale = await failed(probe.probeLease(handle, LEASE_PATH) as Promise<Completion<never>>);
    expect(stale.declaration.name).toBe("test::stale_handle");
    expect(field(stale.payload, "handle")).toBe("lease");
  }
});

test("descriptor probe fault kinds map verbatim with reason fallback", async () => {
  const { owner, probe } = rig((request) => {
    const args = request["args"] as Record<string, unknown>;
    if (args["path"] === "/bad") {
      return { outcome: "rejected", kind: "invalid-path", message: "no such lease", facts: {} };
    }
    return { outcome: "rejected", kind: "invalid-request", facts: {} };
  });
  const handle = await ok(owner.admitGrant(GRANT));
  const bad = await failed(probe.probeLease(handle, "/bad") as Promise<Completion<never>>);
  expect(bad.declaration.name).toBe("descriptor::descriptor_fault");
  expect(field(bad.payload, "kind")).toBe("invalid-path");
  expect(field(bad.payload, "reason")).toBe("no such lease");
  const invalid = await failed(probe.probeLease(handle, "/other") as Promise<Completion<never>>);
  expect(field(invalid.payload, "kind")).toBe("invalid-request");
  expect(field(invalid.payload, "reason")).toBe("rejected/invalid-request");
});

test("descriptor probe foreign handles are stale without dispatch; malformed input throws", async () => {
  const { owner, probe, requests } = rig(() => leaseFacts(LEASE_PATH, true, true, false));
  const handle = await ok(owner.admitGrant(GRANT));
  await ok(owner.releaseGrant(handle));
  const stale = await failed(probe.probeLease(handle, LEASE_PATH) as Promise<Completion<never>>);
  expect(stale.declaration.name).toBe("test::stale_handle");
  expect(requests).toHaveLength(0);
  const live = await ok(owner.admitGrant(GRANT));
  await expect(probe.probeLease(live, 42)).rejects.toThrow(TypeError);
});

test("descriptor probe default dispatch throws before any owner binds", async () => {
  const unboundOwner = createTestOwner(domain, { invalidGrant: err("test::stale_handle") });
  const unbound = createLeaseProbe(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      descriptorFault: err("descriptor::descriptor_fault"),
      leaseReport: typeId("descriptor::lease_report"),
    },
    unboundOwner,
  );
  const handle = await ok(unboundOwner.admitGrant(GRANT));
  await expect(unbound.probeLease(handle, LEASE_PATH)).rejects.toThrow(
    "descriptor N transport is not bound",
  );
});
