import { test, expect } from "bun:test";
import { success, type Completion } from "../completion.ts";
import {
  rootIdentity,
  invocationIdentity,
  participantIdentities,
  compareInvocations,
} from "../assert/lineage.ts";
import {
  createBarrier,
  reserveFrame,
  startFrame,
  suspendFrame,
  resumeFrame,
  finishFrame,
  abandonFrame,
  fixtureEvent,
  barrierState,
  type Frame,
} from "../assert/barrier.ts";
const root = () => rootIdentity({ package: "p", declaration: "p::main", name: "sample" });
const tick = async () => {
  for (let i = 0; i < 30; i++) await Promise.resolve();
};

test("barrier waits for every reserved participant and orders shared requests by full path", async () => {
  const identity = root(),
    owner = createBarrier(identity),
    participants = participantIdentities(identity, "p::main#0", [[0], [1]]);
  const a = reserveFrame(owner, participants[0]),
    b = reserveFrame(owner, participants[1]);
  const events: string[] = [];
  startFrame(b);
  const second = fixtureEvent(b, () => {
    events.push("B");
    return success("B");
  }).then((value) => {
    finishFrame(b);
    return value;
  });
  await tick();
  expect(events).toEqual([]);
  expect(barrierState(owner).pending).toBe(1);
  startFrame(a);
  await tick();
  expect(events).toEqual([]);
  const first = fixtureEvent(a, () => {
    events.push("A");
    return success("A");
  }).then((value) => {
    finishFrame(a);
    return value;
  });
  expect((await first).kind).toBe("ok");
  expect((await second).kind).toBe("ok");
  expect(events).toEqual(["A", "B"]);
  expect(barrierState(owner).frames).toEqual([]);
});

test("arbitrarily delayed pure descendants keep fixture allocation behind a proven barrier", async () => {
  const identity = root(),
    owner = createBarrier(identity),
    participants = participantIdentities(identity, "p::main#0", [[0], [1]]);
  const a = reserveFrame(owner, participants[0]),
    b = reserveFrame(owner, participants[1]);
  startFrame(a);
  startFrame(b);
  const events: string[] = [];
  const second = fixtureEvent(b, () => {
    events.push("B");
    return success("B");
  }).then((value) => {
    finishFrame(b);
    return value;
  });
  const child = reserveFrame(owner, invocationIdentity(participants[0], "p::pure#0"));
  suspendFrame(a);
  startFrame(child);
  for (let i = 0; i < 4; i++) await tick();
  expect(events).toEqual([]);
  finishFrame(child);
  resumeFrame(a);
  const first = fixtureEvent(a, () => {
    events.push("A1");
    return success("A1");
  }).then(async () => {
    const value = await fixtureEvent(a, () => {
      events.push("A2");
      return success("A2");
    });
    finishFrame(a);
    return value;
  });
  await Promise.all([first, second]);
  expect(events).toEqual(["A1", "A2", "B"]);
});

test("native selection continuation stays active before a late fixture can release", async () => {
  const identity = root(),
    owner = createBarrier(identity),
    participants = participantIdentities(identity, "p::main#0", [[0], [1]]);
  const a = reserveFrame(owner, participants[0]),
    b = reserveFrame(owner, participants[1]);
  startFrame(a);
  startFrame(b);
  const events: string[] = [];
  let gate!: Frame;
  const second = fixtureEvent(b, () => {
    events.push("late B");
    return success("B");
  }).then((value) => {
    finishFrame(b);
    return value;
  });
  const first = fixtureEvent(a, () => success("A")).then((value) => {
    // Native coordination must make this transition atomically when its adapter
    // observes a completion that can settle the aggregate.
    gate = reserveFrame(owner, invocationIdentity(identity, "p::selection#0"));
    startFrame(gate);
    finishFrame(a);
    return value;
  });
  await first;
  await tick();
  expect(events).toEqual([]);
  events.push("native continuation");
  finishFrame(gate);
  await second;
  expect(events).toEqual(["native continuation", "late B"]);
});

test("conformance can exercise reverse ordering without changing authored default", async () => {
  const identity = root(),
    owner = createBarrier(identity, (a, b) => -compareInvocations(a, b));
  const participants = participantIdentities(identity, "p::main#0", [[0], [1], [2]]),
    events: number[] = [];
  const ready = participants.map((value) => reserveFrame(owner, value));
  ready.forEach(startFrame);
  await Promise.all(
    ready.map((value, index) =>
      fixtureEvent(value, () => {
        events.push(index);
        return success(index);
      }).then((result) => {
        finishFrame(value);
        return result;
      }),
    ),
  );
  expect(events).toEqual([2, 1, 0]);
});

test("barriers reject forged and duplicate identities and invalid transitions", async () => {
  const identity = root(),
    owner = createBarrier(identity),
    value = reserveFrame(owner, identity);
  expect(() => reserveFrame(owner, identity)).toThrow("already reserved");
  expect(() => reserveFrame(owner, root())).toThrow("different assertion roots");
  expect(() => startFrame({} as Frame)).toThrow("invalid assertion frame");
  expect(() => fixtureEvent(value, () => success(undefined))).toThrow("outside a running");
  startFrame(value);
  expect(() => startFrame(value)).toThrow("transition");
  const malformed = fixtureEvent(value, () => ({ kind: "ok", value: 1 }) as Completion);
  await expect(malformed).rejects.toThrow();
  finishFrame(value);
  const abandoned = reserveFrame(owner, invocationIdentity(identity, "p::unused#0"));
  abandonFrame(abandoned);
  expect(barrierState(owner).frames).toEqual([]);
});

test("invalid conformance ordering rejects every pending request without stranding frames", async () => {
  const identity = root(),
    owner = createBarrier(identity, () => NaN);
  const ready = participantIdentities(identity, "p::main#0", [[0], [1]]).map((value) =>
    reserveFrame(owner, value),
  );
  ready.forEach(startFrame);
  const outcomes = await Promise.allSettled(
    ready.map((value) =>
      fixtureEvent(value, () => success(undefined)).finally(() => finishFrame(value)),
    ),
  );
  expect(outcomes.map((value) => value.status)).toEqual(["rejected", "rejected"]);
  expect(barrierState(owner)).toEqual({ pending: 0, frames: [] });
  expect(() => participantIdentities(identity, "p::main#1", [[1], [0]])).toThrow(
    "flattened source order",
  );
  expect(() => participantIdentities(identity, "p::main#1", [[0], [0, 0]])).toThrow(
    "flattened source order",
  );
});

test("fixture arbitration stays within a logarithmic comparison budget while draining and reinserting", async () => {
  for (const count of [1000, 4000]) {
    let comparisons = 0;
    const identity = root(),
      owner = createBarrier(identity, (a, b) => {
        comparisons++;
        return compareInvocations(a, b);
      });
    const ready = participantIdentities(
      identity,
      "p::scale#0",
      Array.from({ length: count }, (_, index) => [index]),
    ).map((value) => reserveFrame(owner, value));
    ready.forEach(startFrame);
    const events: number[] = [];
    // Reverse arrivals require ordering work. The first frame then repeatedly
    // rejoins ahead of the retained queue, exercising incremental insertion too.
    const outcomes = ready.toReversed().map((value, reverseIndex) => {
      const index = count - 1 - reverseIndex;
      return (async () => {
        const visits = index === 0 ? count : 1;
        for (let visit = 0; visit < visits; visit++)
          await fixtureEvent(value, () => {
            events.push(index);
            return success(index);
          });
        finishFrame(value);
      })();
    });
    await Promise.all(outcomes);
    expect(events).toEqual([
      ...Array(count).fill(0),
      ...Array.from({ length: count - 1 }, (_, index) => index + 1),
    ]);
    expect(comparisons).toBeLessThan(4 * (2 * count - 1) * Math.ceil(Math.log2(count + 1)));
    expect(barrierState(owner)).toEqual({ pending: 0, frames: [] });
  }
});

test("comparison failures during heap removal reject every retained request and allow recovery", async () => {
  const identity = root();
  let armed = false,
    afterArm = 0;
  const cause = new Error("comparison failed during removal");
  const owner = createBarrier(identity, (a, b) => {
    if (armed && ++afterArm === 2) throw cause;
    return compareInvocations(a, b);
  });
  const ready = participantIdentities(
    identity,
    "p::failure#0",
    Array.from({ length: 16 }, (_, index) => [index]),
  ).map((value) => reserveFrame(owner, value));
  ready.forEach(startFrame);
  const events: number[] = [];
  const outcomes = await Promise.allSettled(
    ready.map((value, index) =>
      fixtureEvent(value, () => {
        events.push(index);
        armed = true;
        return success(index);
      }).finally(() => finishFrame(value)),
    ),
  );
  expect(events).toEqual([0]);
  expect(outcomes[0].status).toBe("fulfilled");
  for (const outcome of outcomes.slice(1)) {
    expect(outcome.status).toBe("rejected");
    if (outcome.status === "rejected") expect(outcome.reason).toBe(cause);
  }
  expect(barrierState(owner)).toEqual({ pending: 0, frames: [] });
  armed = false;
  const next = reserveFrame(owner, invocationIdentity(identity, "p::recovery#0"));
  startFrame(next);
  expect((await fixtureEvent(next, () => success(7))).kind).toBe("ok");
  finishFrame(next);
  expect(barrierState(owner)).toEqual({ pending: 0, frames: [] });
});

test("equal conformance priorities retain arrival order", async () => {
  const identity = root(),
    owner = createBarrier(identity, () => 0);
  const ready = participantIdentities(identity, "p::ties#0", [[0], [1], [2], [3]]).map((value) =>
    reserveFrame(owner, value),
  );
  ready.forEach(startFrame);
  const events: number[] = [];
  await Promise.all(
    [2, 0, 3, 1].map((index) =>
      fixtureEvent(ready[index], () => {
        events.push(index);
        return success(index);
      }).finally(() => finishFrame(ready[index])),
    ),
  );
  expect(events).toEqual([2, 0, 3, 1]);
});
