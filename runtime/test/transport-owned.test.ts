import { test, expect } from "bun:test";
import { nativeOperation, cleanupOperation } from "../transport/owned.ts";
import { Deadline, transportProblem } from "../transport/deadline.ts";
import {
  runOwnedRoot,
  withScope,
  launchOwned,
  registerResource,
  useResource,
  type OwnerDiagnostic,
} from "../owner.ts";
import { success } from "../completion.ts";
function deferred<T>() {
  let resolve!: (v: T) => void;
  const promise = new Promise<T>((r) => (resolve = r));
  return { promise, resolve };
}
test("deadline returns before native settlement but root retains the actual work", async () => {
  const native = deferred<number>(),
    timed = deferred<void>();
  let done = false;
  const root = runOwnedRoot(async () => {
    const deadline = new Deadline(10);
    try {
      await nativeOperation(() => native.promise, [], deadline);
      throw new Error("unexpected settlement");
    } catch (cause) {
      expect(transportProblem(cause)).toEqual({ kind: "timeout", milliseconds: 10 });
    } finally {
      deadline.dispose();
    }
    timed.resolve();
    return success(1n);
  }).then((result) => {
    done = true;
    return result;
  });
  await timed.promise;
  await Promise.resolve();
  expect(done).toBe(false);
  native.resolve(9);
  expect((await root).completion.kind).toBe("ok");
});
test("cleanup rejection is observed without replacing an already chosen outcome", async () => {
  const diagnostics: OwnerDiagnostic[] = [];
  const result = await runOwnedRoot(
    () => {
      cleanupOperation(async () => {
        throw new Error("cleanup fault");
      });
      return success(1n);
    },
    (d) => {
      diagnostics.push(d);
    },
  );
  expect(result.completion).toMatchObject({ kind: "ok", value: 1n });
  expect(result.cleanupFailed).toBe(false);
  expect(diagnostics).toMatchObject([{ phase: "late", category: "native_exception" }]);
});

test("native continuations finish under the existing owner while its scope drains", async () => {
  const returned = deferred<void>(),
    proceed = deferred<void>();
  let used = false;
  const root = runOwnedRoot(() =>
    withScope(async () => {
      const resource = registerResource("scoped", {}, () => success(undefined), {
        scopeManaged: true,
      });
      const group = launchOwned([
        {
          captures: [resource],
          run: async () => {
            await proceed.promise;
            return nativeOperation(async () =>
              useResource(resource, "scoped", () => {
                used = true;
                return success(undefined);
              }),
            );
          },
        },
      ]);
      group.publish([]);
      returned.resolve();
      return success(undefined);
    }),
  );
  await returned.promise;
  await Promise.resolve();
  await Promise.resolve();
  proceed.resolve();
  expect((await root).completion.kind).toBe("ok");
  expect(used).toBe(true);
});
