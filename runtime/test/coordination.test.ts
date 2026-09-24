import { test, expect } from "bun:test";
import { settle, handle, type Handlers, type Selection } from "../coordination.ts";
import { success, failure, type Completion } from "../completion.ts";
import { captureStandard } from "../failure.ts";
import { runOwnedRoot, type Participant, type OwnerDiagnostic } from "../owner.ts";
const origin = { source: "test:coordination", start: 0, end: 0, invocation: [] };
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((r) => {
    resolve = r;
  });
  return { promise, resolve };
}
const tick = async () => {
  for (let i = 0; i < 12; i++) await Promise.resolve();
};
const failed = (message: string) => failure(captureStandard(new Error(message), origin));
function fixture() {
  const inputs = [deferred<Completion>(), deferred<Completion>(), deferred<Completion>()];
  const starts: number[] = [];
  const participants: Participant[] = inputs.map((input, index) => ({
    captures: [],
    run: () => {
      starts.push(index);
      return input.promise;
    },
  }));
  return { inputs, starts, participants };
}
function handlers(events: number[], failureAt = -1): Handlers {
  return {
    each: (index, completion) => {
      events.push(index);
      return index === failureAt ? failed("handler") : completion;
    },
    shared: (index, completion) => {
      events.push(index);
      return completion;
    },
    allFailed: (outcomes) => success(outcomes),
  };
}

test("native all selects first rejection without running success handlers or redispatching late failures", async () => {
  const f = fixture(),
    published = deferred<void>(),
    events: number[] = [],
    diagnostics: OwnerDiagnostic[] = [];
  let selected!: Selection;
  const root = runOwnedRoot(
    async () => {
      selected = await settle("all", f.participants);
      published.resolve();
      return handle(selected, handlers(events), true, origin);
    },
    (d) => {
      diagnostics.push(d);
    },
  );
  expect(f.starts).toEqual([0, 1, 2]);
  f.inputs[2].resolve(success(3));
  await tick();
  const second = failed("second");
  f.inputs[1].resolve(second);
  await published.promise;
  expect(selected).toMatchObject({ kind: "one", index: 1, completion: second });
  f.inputs[0].resolve(failed("late first"));
  const result = await root;
  expect(result.completion).toBe(second);
  expect(events).toEqual([1]);
  expect(diagnostics.map((d) => d.message)).toEqual(["Error: late first"]);
});

test("allSettled waits for all then handles input order and stops at handler failure", async () => {
  for (const failureAt of [-1, 1]) {
    const f = fixture(),
      events: number[] = [];
    const root = runOwnedRoot(async () =>
      handle(await settle("settled", f.participants), handlers(events, failureAt), true, origin),
    );
    f.inputs[2].resolve(success("C"));
    await tick();
    expect(events).toEqual([]);
    f.inputs[0].resolve(success("A"));
    await tick();
    expect(events).toEqual([]);
    f.inputs[1].resolve(success("B"));
    const result = await root;
    expect(events).toEqual(failureAt < 0 ? [0, 1, 2] : [0, 1]);
    if (failureAt < 0)
      expect(result.completion).toMatchObject({ kind: "ok", value: ["A", "B", "C"] });
    else expect(result.completion.kind).toBe("standard");
  }
});

test("any retains original all-failed outcomes in input order", async () => {
  const f = fixture(),
    failures = [failed("A"), failed("B"), failed("C")],
    diagnostics: OwnerDiagnostic[] = [];
  const root = runOwnedRoot(
    async () => {
      const selected = await settle("any", f.participants);
      expect(selected.kind).toBe("all-failed");
      if (selected.kind !== "all-failed") throw Error("wrong native selection");
      selected.outcomes.forEach((value, i) => expect(value).toBe(failures[i]));
      return success(undefined);
    },
    (d) => {
      diagnostics.push(d);
    },
  );
  for (const i of [1, 2, 0]) {
    f.inputs[i].resolve(failures[i]);
    await tick();
  }
  expect((await root).completion.kind).toBe("ok");
  expect(diagnostics).toEqual([]);
});

test("any consumes pre-success failures while retaining late standard diagnostics", async () => {
  const f = fixture(),
    selected = deferred<void>(),
    diagnostics: OwnerDiagnostic[] = [];
  // oxlint-disable no-thenable -- This payload must expose then to detect accidental Promise assimilation.
  const payload = Object.freeze({
    then() {
      throw Error("thenable payload assimilated");
    },
    value: 7,
  });
  // oxlint-enable no-thenable
  const root = runOwnedRoot(
    async () => {
      const result = await settle("any", f.participants);
      selected.resolve();
      expect(result.kind).toBe("one");
      if (result.kind !== "one") throw Error("missing winner");
      expect(result.index).toBe(1);
      return result.completion;
    },
    (d) => {
      diagnostics.push(d);
    },
  );
  f.inputs[0].resolve(failed("consumed"));
  await tick();
  f.inputs[1].resolve(success(payload));
  await selected.promise;
  f.inputs[2].resolve(failed("late"));
  const result = await root;
  expect(result.completion).toMatchObject({ kind: "ok", value: payload });
  expect(diagnostics.map((d) => d.message)).toEqual(["Error: late"]);
});

test("race selects the first completion and no losing handler runs", async () => {
  const f = fixture(),
    selected = deferred<void>(),
    events: number[] = [];
  const winner = failed("winner");
  const root = runOwnedRoot(async () => {
    const choice = await settle("race", f.participants);
    selected.resolve();
    return handle(choice, handlers(events), false, origin);
  });
  f.inputs[1].resolve(winner);
  await selected.promise;
  f.inputs[0].resolve(success("later"));
  f.inputs[2].resolve(success("last"));
  expect((await root).completion).toBe(winner);
  expect(events).toEqual([1]);
});

test("empty native all and allSettled succeed and any selects one empty aggregate", async () => {
  for (const mode of ["all", "settled", "any"] as const) {
    let aggregate = 0;
    const root = await runOwnedRoot(async () =>
      handle(
        await settle(mode, []),
        {
          each: () => {
            throw Error("empty handler");
          },
          shared: () => {
            throw Error("empty shared handler");
          },
          allFailed: (outcomes) => {
            aggregate++;
            expect(outcomes).toEqual([]);
            return success("fallback");
          },
        },
        true,
        origin,
      ),
    );
    expect(root.completion).toMatchObject({ kind: "ok", value: mode === "any" ? "fallback" : [] });
    expect(aggregate).toBe(mode === "any" ? 1 : 0);
  }
});

async function domainFixture() {
  const { createHash } = await import("node:crypto");
  const { createDomainRuntime, domainFailureDiagnostics } = await import("../domain.ts");
  const { record } = await import("../data.ts");
  const { aggregate } = await import("../coordination.ts");
  const { standardFailureDiagnostics } = await import("../failure.ts");
  const hash = (key: unknown) =>
    createHash("sha256")
      .update("can-concrete-type-v1\0" + JSON.stringify(key))
      .digest("hex");
  const shape = (kind: string, declaration: string, args: string[] = []) => ({
    identity: hash([kind, declaration, ...args]),
    kind,
    declaration,
    arguments: args,
    fields: [] as { name: string; type: string }[],
    leaves: [] as string[],
    inputs: [],
    errors: [],
  });
  const integer = shape("primitive", "int"),
    text = shape("primitive", "str"),
    standard = shape("opaque", "can.prelude@1::standard_failure");
  const fault = shape("error", "can.project.root/app::fault", [integer.identity]);
  fault.fields = [{ name: "value", type: integer.identity }];
  const offline = shape("error", "can.project.root/app::offline");
  offline.fields = [{ name: "message", type: text.identity }];
  const variant = shape("variant", "can.project.root/app::failures");
  variant.leaves = [standard.identity, fault.identity, offline.identity];
  const array = {
    ...shape("array", ""),
    identity: hash(["array", "", variant.identity]),
    element: variant.identity,
  };
  const combined = shape("error", "can.prelude@1::all_failed", [variant.identity]);
  combined.fields = [{ name: "failures", type: array.identity }];
  const domain = createDomainRuntime({
    declarations: [
      { identity: fault.declaration, name: "app::fault", parameters: 1 },
      { identity: offline.declaration, name: "app::offline", parameters: 0 },
      { identity: combined.declaration, name: "all_failed", parameters: 1 },
    ],
    shapes: [integer, text, standard, fault, offline, variant, array, combined],
  });
  const cause = { private: "original native cause" },
    snapshot = captureStandard(cause, origin);
  const a = record(fault.identity, [["value", 9007199254740993n]]),
    b = record(offline.identity, [["message", "down"]]);
  const outcomes = [
    failure(snapshot),
    failure(domain.create(fault.identity, a, origin)),
    failure(domain.create(offline.identity, b, origin)),
  ];
  return {
    integer,
    domain,
    combined,
    variant,
    fault,
    a,
    b,
    snapshot,
    cause,
    outcomes,
    domainFailureDiagnostics,
    standardFailureDiagnostics,
    aggregate,
  };
}

test("typed aggregate keeps nominal generic payloads and opaque standard occurrence identity", async () => {
  const {
    integer,
    domain,
    combined,
    variant,
    fault,
    a,
    b,
    snapshot,
    cause,
    outcomes,
    domainFailureDiagnostics,
    standardFailureDiagnostics,
    aggregate,
  } = await domainFixture();
  const f = fixture();
  const root = runOwnedRoot(async () => {
    const selected = await settle("any", f.participants);
    if (selected.kind !== "all-failed") throw Error("missing all-failed selection");
    return aggregate(
      [...selected.outcomes, selected.outcomes[1]],
      domain,
      combined.identity,
      origin,
    );
  });
  for (const index of [1, 2, 0]) {
    f.inputs[index].resolve(outcomes[index]);
    await tick();
  }
  const result = (await root).completion;
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("missing aggregate error");
  const details = domainFailureDiagnostics(result.value);
  expect(details.declaration.name).toBe("all_failed");
  expect(details.typeArguments).toEqual([variant.identity]);
  const values = (details.payload as { failures: unknown[] }).failures;
  expect(values.length).toBe(4);
  expect(values[0]).toBe(snapshot);
  expect(values[1]).toBe(a);
  expect(values[2]).toBe(b);
  expect(values[3]).toBe(a);
  expect(standardFailureDiagnostics(snapshot).cause).toBe(cause);
  expect(Object.isFrozen(values)).toBe(true);
  const original = outcomes[1];
  if (original.kind !== "domain") throw Error("missing original error");
  expect(domainFailureDiagnostics(original.value)).toMatchObject({
    typeIdentity: fault.identity,
    typeArguments: [integer.identity],
    declaration: { name: "app::fault" },
    payload: a,
  });
});

test("all and race select the first domain or standard failure with optional standard handling", async () => {
  const { outcomes } = await domainFixture();
  for (const mode of ["all", "race"] as const)
    for (const standard of [false, true])
      for (const catchStandard of [false, true]) {
        const f = fixture(),
          published = deferred<void>(),
          events: number[] = [],
          diagnostics: OwnerDiagnostic[] = [];
        const winner = standard ? outcomes[0] : outcomes[1];
        const root = runOwnedRoot(
          async () => {
            const selected = await settle(mode, f.participants);
            published.resolve();
            return handle(
              selected,
              {
                each: () => {
                  throw Error("success handler ran after rejection");
                },
                shared: (index, completion) => {
                  events.push(index);
                  expect(completion).toBe(winner);
                  return standard && !catchStandard ? completion : success("handled");
                },
                allFailed: () => {
                  throw Error("unexpected aggregate");
                },
              },
              mode === "all",
              origin,
            );
          },
          (diagnostic) => {
            diagnostics.push(diagnostic);
          },
        );
        if (mode === "all") {
          f.inputs[2].resolve(success("C"));
          await tick();
        }
        f.inputs[1].resolve(winner);
        await published.promise;
        f.inputs[0].resolve(mode === "all" ? outcomes[2] : success("A"));
        if (mode === "race") f.inputs[2].resolve(failed("late C"));
        const result = (await root).completion;
        expect(events).toEqual([1]);
        if (standard && !catchStandard) expect(result).toBe(winner);
        else expect(result).toMatchObject({ kind: "ok", value: "handled" });
        expect(diagnostics.map((value) => value.message)).toEqual(
          mode === "race" ? ["Error: late C"] : [],
        );
      }
});

test("allSettled propagates an unhandled standard slot in input order", async () => {
  const f = fixture(),
    events: number[] = [],
    failure = failed("B");
  const root = runOwnedRoot(async () =>
    handle(await settle("settled", f.participants), handlers(events), true, origin),
  );
  f.inputs[2].resolve(success("C"));
  await tick();
  f.inputs[0].resolve(success("A"));
  await tick();
  expect(events).toEqual([]);
  f.inputs[1].resolve(failure);
  expect((await root).completion).toBe(failure);
  expect(events).toEqual([0, 1]);
});

test("any succeeds last after domain and standard failures without aggregate dispatch", async () => {
  const { outcomes } = await domainFixture();
  const f = fixture(),
    events: number[] = [],
    diagnostics: OwnerDiagnostic[] = [];
  let aggregateCalls = 0;
  const root = runOwnedRoot(
    async () =>
      handle(
        await settle("any", f.participants),
        {
          each: () => {
            throw Error("unexpected each");
          },
          shared: (index, completion) => {
            events.push(index);
            return completion;
          },
          allFailed: () => {
            aggregateCalls++;
            return failed("unexpected aggregate");
          },
        },
        false,
        origin,
      ),
    (diagnostic) => {
      diagnostics.push(diagnostic);
    },
  );
  f.inputs[1].resolve(outcomes[1]);
  await tick();
  f.inputs[0].resolve(outcomes[0]);
  await tick();
  f.inputs[2].resolve(success("C"));
  expect((await root).completion).toMatchObject({ kind: "ok", value: "C" });
  expect(events).toEqual([2]);
  expect(aggregateCalls).toBe(0);
  expect(diagnostics).toEqual([]);
});

test("handler failures escape without participant redispatch or a partial collection", async () => {
  const { outcomes } = await domainFixture();
  for (const failure of [outcomes[1], failed("handler")])
    for (const mode of ["all", "any"] as const) {
      const f = fixture(),
        events: string[] = [];
      const root = runOwnedRoot(async () =>
        handle(
          await settle(mode, f.participants),
          {
            each: (index) => {
              events.push("each" + index);
              return failure;
            },
            shared: () => {
              events.push("shared");
              return success("wrong");
            },
            allFailed: () => {
              events.push("aggregate");
              return failure;
            },
          },
          true,
          origin,
        ),
      );
      for (const index of [2, 0, 1]) {
        f.inputs[index].resolve(mode === "all" ? success(index) : outcomes[index]);
        await tick();
      }
      expect((await root).completion).toBe(failure);
      expect(events).toEqual(mode === "all" ? ["each0"] : ["aggregate"]);
    }
});

test("explicit map conversion preserves snapshot identity and mints one outer aggregate occurrence", async () => {
  const {
    domain,
    combined,
    fault,
    a,
    snapshot,
    cause,
    domainFailureDiagnostics,
    standardFailureDiagnostics,
    aggregate,
  } = await domainFixture();
  const { map } = await import("../collections/array.ts");
  const { failure, success, value } = await import("../completion.ts");
  const { standardFailureOccurrenceID } = await import("../failure.ts");
  const leaves: unknown[] = [snapshot, a, snapshot];
  const converted = value(
    await map(leaves, async (item) => success(item), { origin, site: "p::convert#0" }),
  );
  expect(converted.length).toBe(3);
  expect(converted[0]).toBe(snapshot);
  expect(converted[1]).toBe(a);
  expect(converted[2]).toBe(snapshot);
  expect(converted).not.toBe(leaves);
  expect(Object.isFrozen(converted)).toBe(true);
  const before = standardFailureOccurrenceID(snapshot);
  const inner = domain.create(fault.identity, a, origin);
  const rebuilt = aggregate(
    [failure(snapshot), failure(inner), failure(snapshot)],
    domain,
    combined.identity,
    origin,
  );
  if (rebuilt.kind !== "domain") throw Error("missing rebuilt aggregate");
  const details = domainFailureDiagnostics(rebuilt.value);
  const values = (details.payload as { failures: unknown[] }).failures;
  expect(values.length).toBe(3);
  expect(values[0]).toBe(snapshot);
  expect(values[1]).toBe(a);
  expect(values[2]).toBe(snapshot);
  expect(standardFailureDiagnostics(snapshot).cause).toBe(cause);
  expect(standardFailureOccurrenceID(snapshot)).toBe(before);
  expect(details.occurrenceID).not.toBe(domainFailureDiagnostics(inner).occurrenceID);
});

test("empty native race remains pending past the harness deadline", async () => {
  let completed = false;
  const pending = runOwnedRoot(async () => {
    const result = await settle("race", []);
    completed = true;
    return success(result);
  });
  // This timer is only the harness observation deadline. It is not a participant,
  // settlement ordering mechanism, Can timeout, or cancellation of the race.
  const observed = await Promise.race([
    pending.then(() => "completed"),
    new Promise<string>((resolve) => setTimeout(() => resolve("deadline"), 10)),
  ]);
  expect(observed).toBe("deadline");
  expect(completed).toBe(false);
});

test("coordination retains a losing participant's captured resource until late work finishes", async () => {
  const { registerResource, useResource, resourceStatus } = await import("../owner.ts");
  const late = deferred<void>(),
    published = deferred<void>(),
    events: string[] = [];
  let token!: ReturnType<typeof registerResource>,
    finished = false;
  const root = runOwnedRoot(async () => {
    token = registerResource(
      "coordination-test",
      { value: 7 },
      () => {
        events.push("close");
        return success(undefined);
      },
      { scopeManaged: true },
    );
    const choice = await settle("race", [
      { captures: [token], run: () => success("winner") },
      {
        captures: [token],
        run: async () => {
          await late.promise;
          return useResource(token, "coordination-test", (native) => {
            events.push("late use");
            expect(native).toEqual({ value: 7 });
            return success("late");
          });
        },
      },
    ]);
    published.resolve();
    return handle(choice, handlers([]), false, origin);
  }).then((value) => {
    finished = true;
    return value;
  });
  await published.promise;
  await tick();
  expect(finished).toBe(false);
  expect(events).toEqual([]);
  expect(resourceStatus(token).leases).toBeGreaterThan(0);
  late.resolve();
  expect((await root).completion).toMatchObject({ kind: "ok", value: "winner" });
  expect(events).toEqual(["late use", "close"]);
  expect(resourceStatus(token).state).toBe("closed");
});

test("each mode delegates selection to its corresponding native Promise operation", async () => {
  const originals = {
    all: Promise.all,
    settled: Promise.allSettled,
    any: Promise.any,
    race: Promise.race,
  };
  const calls: string[] = [];
  // Instrument the boundary while forwarding every call to the native operation.
  for (const [mode, name] of [
    ["all", "all"],
    ["settled", "allSettled"],
    ["any", "any"],
    ["race", "race"],
  ] as const) {
    const original = originals[mode];
    Object.defineProperty(Promise, name, {
      configurable: true,
      writable: true,
      value: function (values: Iterable<unknown>) {
        calls.push(mode);
        return Reflect.apply(original, Promise, [values]);
      },
    });
  }
  try {
    for (const mode of ["all", "settled", "any", "race"] as const) {
      calls.length = 0;
      await runOwnedRoot(async () => {
        await settle(mode, [{ captures: [], run: () => success(1) }]);
        return success(undefined);
      });
      // Owner draining also uses allSettled; selection's operation must be first.
      expect(calls[0]).toBe(mode);
    }
  } finally {
    Promise.all = originals.all;
    Promise.allSettled = originals.settled;
    Promise.any = originals.any;
    Promise.race = originals.race;
  }
});
