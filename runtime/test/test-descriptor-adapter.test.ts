// NT-I11 descriptor adapter contract: envelopes carry the merge-defined
// descriptor wire shapes, launch wire objects round-trip verbatim, facts
// map to nominal records, and non-completed outcomes return stale_handle
// (wrong-owner/stale-handle/closed-handle/not-found) or verbatim
// descriptor_fault kinds. Dispatch stays stubbed: the default transport
// throws, so nothing here claims live owner behavior.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import type { Completion } from "../completion.ts";
import { array, dataProperty, record } from "../data.ts";
import { createTestOwner } from "../test-support/owner.ts";
import {
  createDescriptorDelivery,
  type DescriptorDispatch,
} from "../test-support/slices/i11/descriptor.ts";

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

const SOME = "test:some";
const NONE = "test:none";
const typeId = (name: string): string => `test:type:${name}`;

type Captured = Readonly<Record<string, unknown>>;
function rig(handler: (request: Captured) => Readonly<Record<string, unknown>>) {
  const requests: Captured[] = [];
  const owner = createTestOwner(domain, { invalidGrant: err("test::stale_handle") });
  const delivery = createDescriptorDelivery(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      descriptorFault: err("descriptor::descriptor_fault"),
      some: SOME,
      none: NONE,
      exit: typeId("descriptor::exit"),
      facts: typeId("descriptor::facts"),
      leaseReport: typeId("descriptor::lease_report"),
      report: typeId("descriptor::report"),
    },
    owner,
    (async (request) => {
      requests.push(request);
      return handler(request);
    }) as DescriptorDispatch,
  );
  return { owner, delivery, requests };
}

const GRANT = "ng1-0123456789abcdef0123456789abcdef";

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

function spec(overrides?: Partial<Record<string, unknown>>): unknown {
  const base: Record<string, unknown> = {
    executable: "/bin/echo",
    args: array(["hi"]),
    env: array([]),
    with_lease: false,
    detached: false,
    dir: "/tmp",
    ...overrides,
  };
  return record(typeId("descriptor::spec"), Object.entries(base) as [string, unknown][]);
}

function completed(facts: Record<string, unknown>) {
  return { outcome: "completed", kind: "ok", facts };
}

function launchFacts(pid: number) {
  return completed({ launch: { pid } });
}

const FACTS_WIRE = {
  offered: true,
  offered_bytes: 3,
  writer_done: true,
  writer_err: "",
  accepted: true,
  malformed: false,
  eof: true,
  reaped: true,
  child_exit: { code: 0, signaled: false, signal: "" },
};

test("descriptor::launch_child mints an opaque launch and digests args", async () => {
  const { owner, delivery, requests } = rig(() => launchFacts(4242));
  const handle = await ok(owner.admitGrant(GRANT));
  const launch = await ok(delivery.launchChild(handle, "op-1", spec()));
  expect(typeof launch).toBe("object");
  expect(requests).toHaveLength(1);
  const request = requests[0] as Record<string, unknown>;
  expect(request["operation"]).toBe("descriptor.launch_child");
  expect(request["schema_version"]).toBe("1");
  expect(request["owner_grant"]).toBe(GRANT);
  expect(request["arguments_digest"]).toMatch(/^sha256:[0-9a-f]{64}$/);
  const args = request["args"] as Record<string, unknown>;
  expect(args["op_id"]).toBe("op-1");
  expect(args["spec"]).toEqual({
    executable: "/bin/echo",
    args: ["hi"],
    env: [],
    with_lease: false,
    detached: false,
    dir: "/tmp",
  });
});

test("descriptor::launch_child passes with_lease through for the owner", async () => {
  const { owner, delivery, requests } = rig(() => launchFacts(1));
  const handle = await ok(owner.admitGrant(GRANT));
  await ok(delivery.launchChild(handle, "op-2", spec({ with_lease: true })));
  const args = requests[0]["args"] as Record<string, unknown>;
  expect((args["spec"] as Record<string, unknown>)["with_lease"]).toBe(true);
});

test("descriptor::read_facts maps observations and the optional exit", async () => {
  const { owner, delivery } = rig((request) =>
    request["operation"] === "descriptor.launch_child"
      ? launchFacts(7)
      : completed({ facts: FACTS_WIRE }),
  );
  const handle = await ok(owner.admitGrant(GRANT));
  const launch = await ok(delivery.launchChild(handle, "op-3", spec()));
  const facts = await ok(delivery.readFacts(handle, launch));
  expect(field(facts, "offered")).toBe(true);
  expect(field(facts, "offered_bytes")).toBe(3n);
  expect(field(facts, "writer_err")).toBe("");
  expect(field(facts, "reaped")).toBe(true);
  const child = field(facts, "child_exit");
  expect(field(child, "value") === undefined ? child : field(child, "value")).toBeDefined();
});

test("descriptor::wait_child maps the exit record", async () => {
  const { owner, delivery } = rig((request) =>
    request["operation"] === "descriptor.launch_child"
      ? launchFacts(9)
      : completed({ exit: { code: 3, signaled: false, signal: "" } }),
  );
  const handle = await ok(owner.admitGrant(GRANT));
  const launch = await ok(delivery.launchChild(handle, "op-4", spec()));
  const exit = await ok(delivery.waitChild(handle, launch, 1000n));
  expect(field(exit, "code")).toBe(3n);
  expect(field(exit, "signaled")).toBe(false);
});

test("descriptor::release_launch maps the report with lease data", async () => {
  const { owner, delivery } = rig((request) =>
    request["operation"] === "descriptor.launch_child"
      ? launchFacts(11)
      : completed({
          report: {
            executable: "/bin/echo",
            facts: { ...FACTS_WIRE, child_exit: null },
            lease: { path: "", inherited: false, held: false, released: false },
            orphan: false,
            clean: true,
            reason: "clean",
          },
        }),
  );
  const handle = await ok(owner.admitGrant(GRANT));
  const launch = await ok(delivery.launchChild(handle, "op-5", spec()));
  const report = await ok(delivery.releaseLaunch(handle, launch));
  expect(field(report, "executable")).toBe("/bin/echo");
  expect(field(report, "clean")).toBe(true);
  expect(field(report, "reason")).toBe("clean");
  const lease = field(report, "lease");
  expect(field(lease, "held")).toBe(false);
});

test("descriptor::expected_ack maps the ack frame bytes", async () => {
  const { owner, delivery } = rig((request) =>
    request["operation"] === "descriptor.launch_child"
      ? launchFacts(13)
      : completed({ ack: [1, 2, 3] }),
  );
  const handle = await ok(owner.admitGrant(GRANT));
  const launch = await ok(delivery.launchChild(handle, "op-6", spec()));
  const ack = (await ok(delivery.expectedAck(handle, launch))) as readonly unknown[];
  expect(ack.map((byte) => String(byte))).toEqual(["1", "2", "3"]);
});

test("descriptor::collect_status and kill_child succeed quietly", async () => {
  const { owner, delivery } = rig((request) =>
    request["operation"] === "descriptor.launch_child" ? launchFacts(15) : completed({}),
  );
  const handle = await ok(owner.admitGrant(GRANT));
  const launch = await ok(delivery.launchChild(handle, "op-7", spec()));
  await ok(delivery.collectStatus(handle, launch, 1000n, 2000n));
  await ok(delivery.killChild(handle, launch));
});

test("descriptor stale kinds return test::stale_handle", async () => {
  for (const kind of ["wrong-owner", "stale-handle", "closed-handle", "not-found"]) {
    const { owner, delivery } = rig((request) =>
      request["operation"] === "descriptor.launch_child"
        ? launchFacts(17)
        : { outcome: "failed", kind, message: `${kind} here`, facts: {} },
    );
    const handle = await ok(owner.admitGrant(GRANT));
    const launch = await ok(delivery.launchChild(handle, `op-stale-${kind}`, spec()));
    const stale = await failed(delivery.readFacts(handle, launch) as Promise<Completion<never>>);
    expect(stale.declaration.name).toBe("test::stale_handle");
  }
});

test("descriptor fault kinds map verbatim with reason fallback", async () => {
  const { owner, delivery } = rig((request) => {
    if (request["operation"] === "descriptor.launch_child") return launchFacts(19);
    if (request["operation"] === "descriptor.wait_child") {
      return { outcome: "deadline", kind: "deadline", message: "wait overran", facts: {} };
    }
    return { outcome: "rejected", kind: "invalid-request", facts: {} };
  });
  const handle = await ok(owner.admitGrant(GRANT));
  const launch = await ok(delivery.launchChild(handle, "op-8", spec()));
  const timeout = await failed(
    delivery.waitChild(handle, launch, 5n) as Promise<Completion<never>>,
  );
  expect(timeout.declaration.name).toBe("descriptor::descriptor_fault");
  expect(field(timeout.payload, "kind")).toBe("deadline");
  expect(field(timeout.payload, "reason")).toBe("wait overran");
  const invalid = await failed(delivery.readFacts(handle, launch) as Promise<Completion<never>>);
  expect(field(invalid.payload, "kind")).toBe("invalid-request");
  expect(field(invalid.payload, "reason")).toBe("rejected/invalid-request");
});

test("descriptor foreign handles are stale; malformed input throws", async () => {
  const { owner, delivery } = rig((request) =>
    request["operation"] === "descriptor.launch_child" ? launchFacts(21) : completed({}),
  );
  const handle = await ok(owner.admitGrant(GRANT));
  const launch = await ok(delivery.launchChild(handle, "op-9", spec()));
  await ok(owner.releaseGrant(handle));
  const stale = await failed(delivery.readFacts(handle, launch) as Promise<Completion<never>>);
  expect(stale.declaration.name).toBe("test::stale_handle");
  const live = await ok(owner.admitGrant(GRANT));
  await expect(delivery.launchChild(live, "op-10", "not-a-spec")).rejects.toThrow(TypeError);
});

test("descriptor default dispatch throws before any owner binds", async () => {
  const unboundOwner = createTestOwner(domain, { invalidGrant: err("test::stale_handle") });
  const unbound = createDescriptorDelivery(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      descriptorFault: err("descriptor::descriptor_fault"),
      some: SOME,
      none: NONE,
      exit: typeId("descriptor::exit"),
      facts: typeId("descriptor::facts"),
      leaseReport: typeId("descriptor::lease_report"),
      report: typeId("descriptor::report"),
    },
    unboundOwner,
  );
  const handle = await ok(unboundOwner.admitGrant(GRANT));
  await expect(unbound.launchChild(handle, "op-11", spec())).rejects.toThrow(
    "descriptor N transport is not bound",
  );
});
