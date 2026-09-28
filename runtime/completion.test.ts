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
  checkedCompletion,
  isCompletion,
  invoke,
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

test("async calls and continuations never assimilate then data", async () => {
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
  expect(invoked).toBe(0);
  expect(Object.getPrototypeOf(nested)).toBeNull();
  expect(Object.isFrozen(nested)).toBe(true);
  expect("then" in nested).toBe(false);
  for (const descriptor of Object.values(Object.getOwnPropertyDescriptors(nested))) {
    expect("value" in descriptor).toBe(true);
    expect(descriptor.writable).toBe(false);
    expect(descriptor.configurable).toBe(false);
  }
});
test("void and distinct failure categories survive invocation with original occurrences", async () => {
  expect(value(await invoke(async () => success(undefined), origin))).toBeUndefined();
  const error = fresh(),
    standard = captureStandard(new Error("broken"), origin);
  for (const occurrence of [error, standard]) {
    const completion = failure(occurrence);
    const recovered = await invoke(() => Promise.resolve(completion), origin);
    expect(recovered).toBe(completion);
    expect(recovered.value).toBe(occurrence);
  }
  expect(domainFailureDiagnostics(error).occurrenceID).not.toBe(
    standardFailureOccurrenceID(standard),
  );
  const caught = await invoke(() => {
    throw standard;
  }, origin);
  expect(caught.value).toBe(standard);
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
  expect(() => failure({} as any)).toThrow();
});

test("carriers expose exactly two frozen null-prototype data keys with fresh branded identity", () => {
  const domainFailure = fresh();
  const standardFailure = captureStandard(new Error("shape"), origin);
  const carriers: Array<{
    carrier: Completion;
    kind: "ok" | "domain" | "standard";
    payload: unknown;
  }> = [
    { carrier: success(1), kind: "ok", payload: 1 },
    { carrier: success(undefined), kind: "ok", payload: undefined },
    { carrier: failure(domainFailure), kind: "domain", payload: domainFailure },
    { carrier: failure(standardFailure), kind: "standard", payload: standardFailure },
  ];
  expect(success(1)).not.toBe(success(1));
  for (const { carrier, kind, payload } of carriers) {
    expect(Object.getPrototypeOf(carrier)).toBeNull();
    expect(Object.isFrozen(carrier)).toBe(true);
    expect(Object.keys(carrier)).toEqual(["kind", "value"]);
    expect(Reflect.ownKeys(carrier)).toEqual(["kind", "value"]);
    expect("then" in carrier).toBe(false);
    for (const [key, expected] of [
      ["kind", kind],
      ["value", payload],
    ] as const) {
      const descriptor = Object.getOwnPropertyDescriptor(carrier, key)!;
      expect("value" in descriptor).toBe(true);
      expect(descriptor.enumerable).toBe(true);
      expect(descriptor.writable).toBe(false);
      expect(descriptor.configurable).toBe(false);
      expect(descriptor.value).toBe(expected);
    }
    expect(isCompletion(carrier)).toBe(true);
    expect(checkedCompletion(carrier)).toBe(carrier);
    expect(carrier.kind).toBe(kind);
    expect(carrier.value).toBe(payload);
  }
  const nullProtoFake = { __proto__: null, kind: "ok", value: 1 };
  expect(isCompletion(nullProtoFake)).toBe(false);
  expect(() => checkedCompletion(nullProtoFake as unknown as Completion)).toThrow();
});

test("carrier admission rejects forged getters and proxies before inspection", () => {
  let getters = 0;
  const forged = {};
  Object.defineProperty(forged, "kind", {
    enumerable: true,
    configurable: true,
    get() {
      getters++;
      return "ok";
    },
  });
  Object.defineProperty(forged, "value", {
    enumerable: true,
    configurable: true,
    get() {
      getters++;
      return 1;
    },
  });
  expect(isCompletion(forged)).toBe(false);
  expect(() => checkedCompletion(forged as Completion)).toThrow();
  expect(getters).toBe(0);

  let traps = 0;
  const genuine = success(1);
  const proxy = new Proxy(genuine, {
    get(target, property, receiver) {
      traps++;
      return Reflect.get(target, property, receiver);
    },
    getOwnPropertyDescriptor(target, property) {
      traps++;
      return Reflect.getOwnPropertyDescriptor(target, property);
    },
    ownKeys(target) {
      traps++;
      return Reflect.ownKeys(target);
    },
  });
  expect(isCompletion(proxy)).toBe(false);
  expect(() => checkedCompletion(proxy as Completion)).toThrow();
  expect(traps).toBe(0);

  const revoked = Proxy.revocable({}, {});
  revoked.revoke();
  expect(isCompletion(revoked.proxy)).toBe(false);
  expect(() => checkedCompletion(revoked.proxy as Completion)).toThrow();
});

test("success preserves hostile then and getter payloads without assimilation", async () => {
  // oxlint-disable no-thenable -- Hostile then/getter payloads prove boxing never assimilates them.
  let thenCalls = 0;
  const thenPayload = {
    then() {
      thenCalls++;
      throw new Error("assimilated");
    },
  };
  let getterCalls = 0;
  const getterPayload = {};
  Object.defineProperty(getterPayload, "then", {
    enumerable: true,
    configurable: true,
    get() {
      getterCalls++;
      throw new Error("assimilated getter");
    },
  });
  // oxlint-enable no-thenable
  for (const payload of [thenPayload, getterPayload]) {
    const carrier = success(payload);
    expect(carrier.value).toBe(payload);
    expect(value(carrier)).toBe(payload);
    const recovered = await invoke(async () => success(payload), origin);
    expect(recovered.value).toBe(payload);
    expect(value(recovered)).toBe(payload);
    expect(value(await Promise.resolve(carrier).then((c) => success(value(c))))).toBe(payload);
  }
  expect(thenCalls).toBe(0);
  expect(getterCalls).toBe(0);
  expect(success(undefined).value).toBeUndefined();
  const domainFailure = fresh();
  const standardFailure = captureStandard(new Error("payload"), origin);
  expect(failure(domainFailure).value).toBe(domainFailure);
  expect(failure(standardFailure).value).toBe(standardFailure);
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
