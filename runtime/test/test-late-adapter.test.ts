// NT-I03 late-occurrence adapter contract: the 13 operations drive one
// adapter-held K06 service with owner admission first; facts map to
// nominal records (ints as bigint, nullables as options); revoked
// owners fail test::stale_handle without touching the service; and
// every service rejection maps VERBATIM to late::late_fault{kind,
// reason} so K06 verdicts (swap, replay, unbound lease, caps) stay
// visible to Can rows instead of collapsing into stale_handle.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import type { Completion } from "../completion.ts";
import { dataProperty, recordIdentity } from "../data.ts";
import { createTestOwner } from "../test-support/owner.ts";
import { createLateOccurrence } from "../test-support/slices/i03/late.ts";

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
const declarations = catalogue.errors.filter(
  (error) => error.name === "test::stale_handle" || error.name === "late::late_fault",
);
const shapeOf = (id: string, fields: readonly { name: string; type: string }[]): FailureShape => ({
  identity: identity("error", id),
  kind: "error",
  declaration: id,
  arguments: [],
  fields: fields.map((field) => ({ name: field.name, type: text.identity })),
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
  shapes: [text, ...errorShapes.values()],
});
const err = (name: string): string => {
  const shape = errorShapes.get(name);
  if (shape === undefined) throw Error(`no shape for ${name}`);
  return shape.identity;
};

const SOME = "test:some";
const NONE = "test:none";
const typeId = (name: string): string => `test:type:${name}`;

function rig() {
  const owner = createTestOwner(domain, { invalidGrant: err("test::stale_handle") });
  const late = createLateOccurrence(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      lateFault: err("late::late_fault"),
      some: SOME,
      none: NONE,
      selectFacts: typeId("late::select_facts"),
      enrollFacts: typeId("late::enroll_facts"),
      gateFacts: typeId("late::gate_facts"),
      eventFacts: typeId("late::event_facts"),
      terminalFacts: typeId("late::terminal_facts"),
      observationFacts: typeId("late::observation_facts"),
      leaseFacts: typeId("late::lease_facts"),
      releaseFacts: typeId("late::release_facts"),
      reconcileFacts: typeId("late::reconcile_facts"),
      outcomeFacts: typeId("late::outcome_facts"),
      counters: typeId("late::counters"),
    },
    owner,
  );
  return { owner, late };
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

async function bootstrap() {
  const { owner, late } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  await ok(late.select(handle, "late.alpha"));
  await ok(late.enroll(handle, "worker"));
  await ok(late.enroll(handle, "observer"));
  return { owner, late, handle };
}

test("late::select/enroll/arm_gate map facts and joins", async () => {
  const { owner, late } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const first = await ok(late.select(handle, "late.alpha"));
  expect(field(first, "identity")).toBe("late.alpha");
  expect(field(first, "joined")).toBe(false);
  const joined = await ok(late.select(handle, "late.alpha"));
  expect(field(joined, "joined")).toBe(true);
  const enrolled = await ok(late.enroll(handle, "worker"));
  expect(field(enrolled, "participant")).toBe("worker");
  expect(field(enrolled, "identity")).toBe("late.alpha");
  const gate = await ok(late.armGate(handle, "worker"));
  expect(field(gate, "gate_seq")).toBe(1n);
  expect(field(gate, "joined")).toBe(false);
  const regate = await ok(late.armGate(handle, "worker"));
  expect(field(regate, "joined")).toBe(true);
});

test("late::emit admits with dropped gaps and late flags", async () => {
  const { late, handle } = await bootstrap();
  await ok(late.armGate(handle, "worker"));
  const first = await ok(late.emit(handle, "worker", "fault", 1n));
  expect(field(first, "seq")).toBe(1n);
  expect(field(first, "late")).toBe(true);
  expect(field(first, "kind")).toBe("fault");
  const gapped = await ok(late.emit(handle, "worker", "use", 3n));
  expect([...(field(gapped, "dropped") as readonly unknown[])]).toEqual([2n]);
  const rec = await ok(late.reconcile(handle, "worker"));
  expect(field(rec, "admitted")).toBe(2n);
  expect(field(rec, "late")).toBe(2n);
  expect([...(field(rec, "dropped") as readonly unknown[])]).toEqual([2n]);
  expect(field(rec, "next_seq")).toBe(4n);
});

test("late::witness_terminal binds a digest; replays fault verbatim", async () => {
  const { late, handle } = await bootstrap();
  await ok(late.emit(handle, "worker", "fault", 1n));
  const sealed = await ok(late.witnessTerminal(handle, "worker", "completed"));
  expect(field(sealed, "terminal")).toBe("completed");
  expect(String(field(sealed, "binding"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  const replay = await failed(
    late.witnessTerminal(handle, "worker", "failed") as Promise<Completion<never>>,
  );
  expect(replay.declaration.name).toBe("late::late_fault");
  expect(field(replay.payload, "kind")).toBe("stale-handle");
  const closed = await failed(
    late.emit(handle, "worker", "fault", 2n) as Promise<Completion<never>>,
  );
  expect(field(closed.payload, "kind")).toBe("permission");
});

test("late::observe detects swaps as verbatim wrong-owner faults", async () => {
  const { late, handle } = await bootstrap();
  await ok(late.emit(handle, "worker", "use", 1n));
  const seen = await ok(late.observe(handle, "worker", "worker", 1n));
  expect(field(seen, "late")).toBe(false);
  expect(recordIdentity(field(seen, "terminal"))).toBe(NONE);
  const swap = await failed(
    late.observe(handle, "observer", "worker", 1n) as Promise<Completion<never>>,
  );
  expect(swap.declaration.name).toBe("late::late_fault");
  expect(field(swap.payload, "kind")).toBe("wrong-owner");
  const missing = await failed(
    late.observe(handle, "worker", "worker", 99n) as Promise<Completion<never>>,
  );
  expect(field(missing.payload, "kind")).toBe("stale-handle");
});

test("late leases bind to the holder terminal", async () => {
  const { late, handle } = await bootstrap();
  await ok(late.grantLease(handle, "worker"));
  const early = await failed(
    late.observeLease(handle, "worker", "llease1") as Promise<Completion<never>>,
  );
  expect(early.declaration.name).toBe("late::late_fault");
  expect(field(early.payload, "kind")).toBe("permission");
  await ok(late.emit(handle, "worker", "fault", 1n));
  await ok(late.witnessTerminal(handle, "worker", "completed"));
  const facts = await ok(late.observeLease(handle, "worker", "llease1"));
  expect(field(facts, "lease")).toBe("llease1");
  expect(field(facts, "holder")).toBe("worker");
  expect(field(facts, "seq")).toBe(1n);
  expect(field(facts, "released")).toBe(false);
  const released = await ok(late.releaseLease(handle, "worker", "llease1"));
  expect(field(released, "terminal")).toBe("completed");
  expect(String(field(released, "binding"))).toMatch(/^sha256:/);
  const joined = await ok(late.releaseLease(handle, "worker", "llease1"));
  expect(field(joined, "binding")).toBe(field(released, "binding"));
  const unknown = await failed(
    late.observeLease(handle, "worker", "llease99") as Promise<Completion<never>>,
  );
  expect(field(unknown.payload, "kind")).toBe("not-found");
});

test("late outcome, kill_worker and counters", async () => {
  const { late, handle } = await bootstrap();
  const partial = await ok(late.readOutcome(handle));
  expect(field(partial, "outcome")).toBe("incomplete");
  expect(field(partial, "reason")).toBe("terminal proof incomplete");
  await ok(late.emit(handle, "worker", "fault", 1n));
  await ok(late.emit(handle, "observer", "use", 1n));
  await ok(late.witnessTerminal(handle, "worker", "completed"));
  await ok(late.witnessTerminal(handle, "observer", "rejected"));
  const full = await ok(late.readOutcome(handle));
  expect(field(full, "outcome")).toBe("complete");
  expect(field(full, "worker_dead")).toBe(false);
  const counts = await ok(late.readCounters(handle));
  expect(field(counts, "participants")).toBe(2n);
  expect(field(counts, "admitted")).toBe(2n);
  expect(field(counts, "terminals")).toBe(2n);
  const dead = await ok(late.killWorker(handle));
  expect(field(dead, "outcome")).toBe("incomplete");
  expect(field(dead, "reason")).toBe("worker-dead");
  expect(field(dead, "worker_dead")).toBe(true);
  const closed = await failed(late.enroll(handle, "worker") as Promise<Completion<never>>);
  expect(field(closed.payload, "kind")).toBe("permission");
});

test("late unknown words fault unsupported-capability verbatim", async () => {
  const { late, handle } = await bootstrap();
  for (const call of [
    late.select(handle, "late.gamma"),
    late.enroll(handle, "stranger"),
    late.emit(handle, "worker", "using", 1n),
    late.witnessTerminal(handle, "worker", "done"),
  ]) {
    const fault = await failed(call as Promise<Completion<never>>);
    expect(fault.declaration.name).toBe("late::late_fault");
    expect(field(fault.payload, "kind")).toBe("unsupported-capability");
  }
  const switched = await failed(late.select(handle, "late.beta") as Promise<Completion<never>>);
  expect(field(switched.payload, "kind")).toBe("changed-input");
});

test("late revoked owners fail stale without touching the service", async () => {
  const { owner, late } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  await ok(late.select(handle, "late.alpha"));
  await ok(owner.releaseGrant(handle));
  const stale = await failed(late.enroll(handle, "worker") as Promise<Completion<never>>);
  expect(stale.declaration.name).toBe("test::stale_handle");
  const staleRead = await failed(late.readCounters(handle) as Promise<Completion<never>>);
  expect(staleRead.declaration.name).toBe("test::stale_handle");
});

test("late malformed inputs throw before the service runs", async () => {
  const { late, handle } = await bootstrap();
  await expect(late.enroll(handle, 42)).rejects.toThrow("late::enroll participant");
  await expect(late.emit(handle, "worker", "use", "1")).rejects.toThrow("late::emit seq");
  await expect(late.observe(handle, "worker", "worker", 1)).rejects.toThrow("late::observe seq");
});
