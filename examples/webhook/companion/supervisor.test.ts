// F05 companion supervisor tests: clean passthrough, bounded
// restarts with backoff, and loud death past the restart budget.
import { test, expect } from "bun:test";
import { defaultSupervisorOptions, runSupervised } from "./supervisor.ts";

test("clean completion is not a restart", async () => {
  let runs = 0;
  let restarts = 0;
  await runSupervised(
    async () => {
      runs += 1;
    },
    defaultSupervisorOptions({
      sleep: async () => {
        throw new Error("must not sleep on clean exit");
      },
      onRestart: () => {
        restarts += 1;
      },
    }),
  );
  expect(runs).toBe(1);
  expect(restarts).toBe(0);
});

test("flaky tasks restart with backoff until they succeed", async () => {
  let runs = 0;
  const sleeps: number[] = [];
  const seen: number[] = [];
  await runSupervised(
    async () => {
      runs += 1;
      if (runs < 3) throw new Error(`boom-${runs}`);
    },
    defaultSupervisorOptions({
      backoffBaseMs: 1000,
      backoffMaxMs: 30000,
      sleep: async (ms: number) => {
        sleeps.push(ms);
      },
      onRestart: (restart: number) => {
        seen.push(restart);
      },
    }),
  );
  expect(runs).toBe(3);
  expect(sleeps).toEqual([1000, 2000]);
  expect(seen).toEqual([1, 2]);
});

test("tasks past the restart budget die with the last failure", async () => {
  let runs = 0;
  const sleeps: number[] = [];
  const failure = await runSupervised(
    async () => {
      runs += 1;
      throw new Error(`boom-${runs}`);
    },
    defaultSupervisorOptions({
      maxRestarts: 2,
      backoffBaseMs: 1000,
      backoffMaxMs: 1500,
      sleep: async (ms: number) => {
        sleeps.push(ms);
      },
    }),
  ).then(
    () => "survived",
    (cause: unknown) => cause,
  );
  expect(runs).toBe(3);
  expect(sleeps).toEqual([1000, 1500]);
  expect((failure as Error).message).toBe("boom-3");
});

test("zero restart budget surfaces the first failure", async () => {
  await expect(
    runSupervised(
      async () => {
        throw new Error("immediate");
      },
      defaultSupervisorOptions({ maxRestarts: 0 }),
    ),
  ).rejects.toThrow("immediate");
});
