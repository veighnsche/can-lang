import { test, expect } from "bun:test";
import { Deadline, transportProblem } from "../transport/deadline.ts";
import { readBody } from "../transport/body.ts";

test("transport stream byte accounting rejects the first excessive chunk", async () => {
  let cancelled = false;
  const deadline = new Deadline(1000);
  const stream = new ReadableStream<Uint8Array>({
    start(c) {
      c.enqueue(new Uint8Array([1, 2]));
      c.enqueue(new Uint8Array([3, 4]));
    },
    cancel() {
      cancelled = true;
    },
  });
  try {
    await readBody(new Response(stream), 3, deadline, () => {});
    throw new Error("accepted oversized body");
  } catch (cause) {
    expect(transportProblem(cause)).toEqual({ kind: "limit", limit: 3 });
  } finally {
    deadline.dispose();
  }
  expect(cancelled).toBe(true);
});
test("transport deadline bounds a stalled body and cancels its stream", async () => {
  let cancelled = false;
  const deadline = new Deadline(20);
  try {
    await readBody(
      new Response(
        new ReadableStream({
          cancel() {
            cancelled = true;
          },
        }),
      ),
      10,
      deadline,
      () => {},
    );
    throw new Error("accepted stalled body");
  } catch (cause) {
    expect(transportProblem(cause)).toEqual({ kind: "timeout" });
  } finally {
    deadline.dispose();
  }
  expect(cancelled).toBe(true);
});
test("transport monotonic deadline rejects synchronous late validation", () => {
  const deadline = new Deadline(2);
  const start = performance.now();
  while (performance.now() - start < 5) {}
  try {
    deadline.check();
    throw new Error("accepted late validation");
  } catch (cause) {
    expect(transportProblem(cause)).toEqual({ kind: "timeout" });
  } finally {
    deadline.dispose();
  }
});
test("transport bodies materialize exact bounded delivered bytes", async () => {
  const deadline = new Deadline(1000);
  try {
    expect([
      ...(await readBody(new Response(new Uint8Array([1, 2, 3])), 3, deadline, () => {})),
    ]).toEqual([1, 2, 3]);
  } finally {
    deadline.dispose();
  }
});

test("deadline expiry takes precedence over an earlier owner cancellation", () => {
  const controller = new AbortController();
  const deadline = new Deadline(2, controller.signal);
  controller.abort();
  const start = performance.now();
  while (performance.now() - start < 5) {}
  try {
    deadline.check();
    throw new Error("accepted cancellation");
  } catch (cause) {
    expect(transportProblem(cause)).toEqual({ kind: "timeout" });
  } finally {
    deadline.dispose();
  }
});
