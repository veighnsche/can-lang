import { test, expect } from "bun:test";
import {
  rootIdentity,
  invocationIdentity,
  participantIdentities,
  invocationPath,
} from "../assert/lineage.ts";
import {
  createBarrier,
  reserveFrame,
  startFrame,
  finishFrame,
  fixtureEvent,
} from "../assert/barrier.ts";
import {
  fixtureQueues,
  registerTable,
  allocateFixture,
  closeQueues,
  FixtureQueueError,
  type FixtureQueues,
} from "../assert/queue.ts";
import { success } from "../completion.ts";
const root = () => rootIdentity({ package: "p", declaration: "p::main", name: "sample" });

test("recursive and concurrent visits share one root lexical FIFO without argument searching", async () => {
  const identity = root(),
    queues = fixtureQueues(identity),
    barrier = createBarrier(identity);
  const participants = participantIdentities(identity, "p::main#0", [[0], [1]]),
    frames = participants.map((value) => reserveFrame(barrier, value));
  frames.forEach(startFrame);
  const assigned: number[] = [];
  // Arrival is reversed, but the scheduler allocates the shared table in path order.
  const visits = [1, 0].map((index) =>
    fixtureEvent(frames[index], () => {
      const invocation = invocationIdentity(participants[index], "p::helper#0");
      const row = allocateFixture(queues, "p::helper#0/when", 3, invocation);
      assigned[index] = row.row;
      return success(invocation);
    }).then((result) => {
      finishFrame(frames[index]);
      return result;
    }),
  );
  const results = await Promise.all(visits);
  expect(assigned).toEqual([0, 1]);
  const first = results[1];
  if (first.kind !== "ok") throw Error("missing first invocation");
  const recursive = invocationIdentity(first.value as ReturnType<typeof root>, "p::helper#0");
  expect(allocateFixture(queues, "p::helper#0/when", 3, recursive).row).toBe(2);
  expect(closeQueues(queues)).toEqual([]);
});

test("allocation errors preserve both full expected and actual invocation paths", () => {
  const identity = root(),
    queues = fixtureQueues(identity),
    call = invocationIdentity(identity, "p::helper#0");
  const allocated = allocateFixture(queues, "p::helper#0/when", 1, call);
  expect(allocated.path).toBe(invocationPath(call));
  const next = invocationIdentity(identity, "p::helper#0");
  try {
    allocateFixture(queues, "p::helper#0/when", 1, next);
    throw Error("accepted exhausted queue");
  } catch (cause) {
    expect(cause).toBeInstanceOf(FixtureQueueError);
    const error = cause as FixtureQueueError;
    expect(error.reason).toBe("missing fixture");
    expect(error.expected.row).toBe(1);
    expect(error.expected.path).toBe(invocationPath(next));
    expect(error.actual).toBe(invocationPath(next));
  }
  expect(() => allocateFixture(queues, "other", 1, call)).toThrow("ambiguous fixture");
  expect(() => registerTable(queues, "p::helper#0/when", 2, next)).toThrow("ambiguous fixture");
});

test("leftovers are checked on explicit close and closed roots cannot consume more fixtures", () => {
  const identity = root(),
    queues = fixtureQueues(identity),
    call = invocationIdentity(identity, "p::helper#0");
  registerTable(queues, "p::helper#0/when", 2, call);
  expect(allocateFixture(queues, "p::helper#0/when", 2, call).row).toBe(0);
  const errors = closeQueues(queues);
  expect(errors.length).toBe(1);
  expect(errors[0].reason).toBe("unused fixture");
  expect(errors[0].expected.row).toBe(1);
  expect(errors[0].actual).toBe(invocationPath(call));
  expect(() =>
    allocateFixture(queues, "p::helper#0/when", 2, invocationIdentity(identity, "p::helper#0")),
  ).toThrow("unexpected live boundary");
});

test("queues reject forged tokens and cannot share allocations between identical-looking roots", () => {
  const a = root(),
    b = root(),
    queues = fixtureQueues(a);
  expect(() => allocateFixture({} as FixtureQueues, "table", 1, a)).toThrow(
    "invalid fixture queues",
  );
  expect(() => allocateFixture(queues, "table", 1, b)).toThrow("different assertion roots");
  expect(allocateFixture(queues, "table", 1, a).row).toBe(0);
  expect(allocateFixture(fixtureQueues(b), "table", 1, b).row).toBe(0);
});
