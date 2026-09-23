import { test, expect } from "bun:test";
import { runEntry } from "../entry.ts";
import { runAssertion } from "../assert/runner.ts";
import { launchOwned, registerResource } from "../owner.ts";
import { success, type Completion } from "../completion.ts";
function deferred<T>() {
  let resolve!: (value: T) => void, reject!: (cause: unknown) => void;
  const promise = new Promise<T>((a, b) => {
    resolve = a;
    reject = b;
  });
  return { promise, resolve, reject };
}

test("CLI waits for late losers yet their diagnostics alone keep status zero", async () => {
  const gate = deferred<Completion>(),
    returned = deferred<void>();
  let done = false;
  const lines: string[] = [];
  const pending = runEntry(
    () => {},
    () => {
      const group = launchOwned([{ captures: [], run: () => gate.promise }]);
      group.publish([]);
      returned.resolve();
      return success(undefined);
    },
    [],
    (line) => {
      lines.push(line);
    },
  ).then((status) => {
    done = true;
    return status;
  });
  await returned.promise;
  await Promise.resolve();
  expect(done).toBe(false);
  gate.reject(new Error("late boom"));
  expect(await pending).toBe(0);
  expect(lines.map((line) => JSON.parse(line))).toMatchObject([
    {
      kind: "can.runtime-diagnostic",
      phase: "late",
      category: "native_exception",
      message: "Error: late boom",
    },
  ]);
});

test("CLI reports omitted close as nonzero even after successful automatic close", async () => {
  let closed = false;
  const lines: string[] = [];
  const status = await runEntry(
    () => {},
    () => {
      registerResource("pool", {}, () => {
        closed = true;
        return success(undefined);
      });
      return success(undefined);
    },
    [],
    (line) => {
      lines.push(line);
    },
  );
  expect(status).toBe(1);
  expect(closed).toBe(true);
  expect(JSON.parse(lines[0])).toMatchObject({ phase: "cleanup", category: "cleanup" });
});

test("assertion result waits for every owned participant before closing context", async () => {
  const gate = deferred<Completion>(),
    returned = deferred<void>();
  let done = false;
  const pending = runAssertion({
    root: { package: "app", declaration: "check", name: "sample" },
    expected: async () => success(1n),
    actual: async () => {
      const group = launchOwned([{ captures: [], run: () => gate.promise }]);
      group.publish([]);
      returned.resolve();
      return success(1n);
    },
  }).then((report) => {
    done = true;
    return report;
  });
  await returned.promise;
  await Promise.resolve();
  expect(done).toBe(false);
  gate.reject(new Error("late assertion task"));
  const report = await pending;
  expect(report.passed).toBe(true);
  expect(report.diagnostics).toMatchObject([
    { phase: "late", message: "Error: late assertion task" },
  ]);
  const next = await runAssertion({
    root: { package: "app", declaration: "check", name: "next" },
    expected: async () => success(2n),
    actual: async () => success(2n),
  });
  expect(next.passed).toBe(true);
  expect(next.diagnostics).toBeUndefined();
});
