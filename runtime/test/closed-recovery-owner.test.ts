import { test, expect } from "bun:test";
import { ambientOwnerPresent, runOwnedRoot, withScope } from "../owner.ts";
import { success } from "../completion.ts";

test("ambientOwnerPresent is false outside any owned root", () => {
  expect(ambientOwnerPresent()).toBe(false);
});

test("ambientOwnerPresent is true inside a root, across awaits, and false after it settles", async () => {
  const seen: boolean[] = [];
  await runOwnedRoot(
    async () => {
      seen.push(ambientOwnerPresent());
      await Promise.resolve();
      seen.push(ambientOwnerPresent());
      return success(undefined);
    },
    () => {},
  );
  expect(seen).toEqual([true, true]);
  expect(ambientOwnerPresent()).toBe(false);
});

test("ambientOwnerPresent stays true under a closed scope while the root store is present", async () => {
  let inside = false,
    afterScope = false;
  await runOwnedRoot(
    async () => {
      await withScope(async () => success(undefined));
      afterScope = ambientOwnerPresent();
      inside = true;
      return success(undefined);
    },
    () => {},
  );
  expect([inside, afterScope]).toEqual([true, true]);
  expect(ambientOwnerPresent()).toBe(false);
});
