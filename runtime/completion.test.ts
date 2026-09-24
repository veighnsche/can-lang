import { expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { record } from "./data.ts";
import { createDomainRuntime, domainFailureDiagnostics } from "./domain.ts";
import {
  captureStandard,
  standardFailureDiagnostics,
  standardFailureOccurrenceID,
} from "./failure.ts";
import {
  success,
  failure,
  value,
  element,
  elementValue,
  checkedCompletion,
  invoke,
  dispatch,
  toPromise,
  fromPromise,
  type Completion,
} from "./completion.ts";
const origin = { source: "test.can", start: 1, end: 2, invocation: ["test"] };
const declaration = {
  identity: "can.project.root::app::missing",
  name: "app::missing",
  parameters: 0,
};
const identity = createHash("sha256")
  .update("can-concrete-type-v1\0" + JSON.stringify(["error", declaration.identity]))
  .digest("hex");
const domain = createDomainRuntime({
  declarations: [declaration],
  shapes: [
    {
      identity,
      kind: "error",
      declaration: declaration.identity,
      arguments: [],
      fields: [],
      leaves: [],
      inputs: [],
      errors: [],
    },
  ],
});
const fresh = () => domain.create(identity, record(identity, []), origin);

test("async calls, continuations, elements and native Promise bridges never assimilate then data", async () => {
  let invoked = 0;
  const payload = record("then-data", [
    [
      "then",
      () => {
        invoked++;
        throw new Error("assimilated data");
      },
    ],
    ["value", 3n],
  ]);
  const f = async () => success(payload);
  const callback = async () => success(value(await invoke(f, origin)));
  const nested = await invoke(callback, origin);
  expect(value(nested)).toBe(payload);
  expect(value(await Promise.resolve(nested).then((c) => success(value(c))))).toBe(payload);
  const inputs = [element(payload), element(payload)];
  const mapped = await Array.fromAsync(inputs, async (e) => element(elementValue(e)));
  expect(mapped.map(elementValue)).toEqual([payload, payload]);
  const promises = [toPromise(f()), toPromise(callback())];
  expect((await Promise.all(promises)).map(value)).toEqual([payload, payload]);
  expect(value(await fromPromise(Promise.any(promises), origin))).toBe(payload);
  expect(value(await fromPromise(Promise.race(promises), origin))).toBe(payload);
  for (const settled of await Promise.allSettled(promises)) {
    expect(settled.status).toBe("fulfilled");
    if (settled.status === "fulfilled") expect(value(settled.value)).toBe(payload);
  }
  expect(invoked).toBe(0);
  for (const carrier of [nested, ...mapped]) {
    expect(Object.getPrototypeOf(carrier)).toBeNull();
    expect(Object.isFrozen(carrier)).toBe(true);
    expect("then" in carrier).toBe(false);
    for (const descriptor of Object.values(Object.getOwnPropertyDescriptors(carrier))) {
      expect("value" in descriptor).toBe(true);
      expect(descriptor.writable).toBe(false);
      expect(descriptor.configurable).toBe(false);
    }
  }
});
test("void and distinct failure categories round trip with original occurrences", async () => {
  expect(value(await invoke(async () => success(undefined), origin))).toBeUndefined();
  const error = fresh(),
    standard = captureStandard(new Error("broken"), origin);
  for (const occurrence of [error, standard]) {
    const completion = failure(occurrence);
    const recovered = await fromPromise(toPromise(Promise.resolve(completion)), origin);
    expect(recovered).toBe(completion);
    expect(recovered.value).toBe(occurrence);
  }
  expect(domainFailureDiagnostics(error).occurrenceID).not.toBe(
    standardFailureOccurrenceID(standard),
  );
  const winner = await Promise.any([
    toPromise(Promise.resolve(failure(error))),
    toPromise(Promise.resolve(success(4n))),
  ]);
  expect(value(winner)).toBe(4n);
  const caught = await invoke(() => {
    throw standard;
  }, origin);
  expect(caught.value).toBe(standard);
});
test("selected handler failure is never redispatched and nested handlers complete locally", async () => {
  const first = fresh(),
    second = fresh();
  let visits = 0;
  const handlers = {
    ok: () => success(0n),
    domain: new Map([
      [
        identity,
        () => {
          visits++;
          return failure(second);
        },
      ],
    ]),
  };
  const result = await dispatch(failure(first), handlers, origin);
  expect(result.kind).toBe("domain");
  expect(result.value).toBe(second);
  expect(visits).toBe(1);
  const standard = captureStandard("handler failed", origin);
  const failed = await dispatch(
    success(1n),
    {
      ok: () => {
        throw standard;
      },
      domain: new Map(),
      standard: () => {
        throw new Error("redispatched");
      },
    },
    origin,
  );
  expect(failed.value).toBe(standard);
  const nested = await dispatch(
    success(2n),
    {
      ok: async (n) => {
        const local = await dispatch(
          success(n),
          { ok: (v) => success(v + 1n), domain: new Map() },
          origin,
        );
        return success(value(local) + 1n);
      },
      domain: new Map(),
    },
    origin,
  );
  expect(value(nested)).toBe(4n);
});
test("forged carriers and immediate unboxed thenables are refused without executing data", async () => {
  // oxlint-disable no-thenable -- A hostile then member proves invoke rejects the raw value before assimilation.
  let invoked = 0;
  const raw = {
    then() {
      invoked++;
    },
  };
  // oxlint-enable no-thenable
  const result = await invoke((() => raw) as any, origin);
  expect(result.kind).toBe("standard");
  expect(invoked).toBe(0);
  const revoked = Proxy.revocable({}, {});
  revoked.revoke();
  for (const forged of [{ kind: "ok", value: 1 }, new Proxy(success(1), {}), revoked.proxy])
    expect(() => checkedCompletion(forged as Completion)).toThrow();
  expect(() => elementValue(success(1) as any)).toThrow();
  expect(() => failure({} as any)).toThrow();
});

test("returned and rejected synthetic failures acquire one checked boundary without new occurrences", async () => {
  for (const mode of ["immediate", "awaited", "rejected"] as const) {
    const original = { source: "can:adapter", start: 0, end: 0, invocation: [] };
    const occurrence = captureStandard("private", original),
      carrier = failure(occurrence);
    const result = await invoke(
      () =>
        mode === "immediate"
          ? carrier
          : mode === "awaited"
            ? Promise.resolve(carrier)
            : Promise.reject(occurrence),
      origin,
    );
    expect(result.kind).toBe("standard");
    expect(result.value).toBe(occurrence);
    expect(standardFailureDiagnostics(occurrence).origin).toEqual(original);
    expect(standardFailureDiagnostics(occurrence).boundaryOrigin).toEqual(origin);
    await invoke(() => carrier, { source: "outer.can", start: 10, end: 20, invocation: [] });
    expect(standardFailureDiagnostics(occurrence).boundaryOrigin).toEqual(origin);
  }
});
