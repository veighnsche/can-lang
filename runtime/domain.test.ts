import { expect, test } from "bun:test";
import { createHash } from "node:crypto";
import {
  createDomainRuntime,
  createDomainRuntimeWithDigest,
  domainFailureDiagnostics,
  isDomainFailure,
  type FailureShape,
} from "./domain";
import { record, array } from "./data";
import { catalogue, errorIdentity, validateErrorIdentity } from "./catalogue";
import { captureStandard, standardFailureOccurrenceID } from "./failure";

// Independent small plans exercise native boundary behavior. The Go integration
// test also serializes real checked plans for every catalogue error declaration.
const shape = (
  kind: string,
  declaration: string,
  args: readonly string[] = [],
  fields: FailureShape["fields"] = [],
  leaves: readonly string[] = [],
): FailureShape => ({
  identity: createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration, ...args]))
    .digest("hex"),
  kind,
  declaration,
  arguments: args,
  fields,
  leaves,
  inputs: [],
  errors: [],
});
const integer = shape("primitive", "int"),
  text = shape("primitive", "str");
const builtin = catalogue.errors.find((e) => e.name === "codec::invalid_data")!;
const codec = shape(
  "error",
  builtin.identity,
  [],
  [
    { name: "path", type: text.identity },
    { name: "reason", type: text.identity },
  ],
);
const project = {
  identity: "can.project.root::app::failed",
  name: "app::failed",
  parameters: 1,
};
const intError = shape(
  "error",
  project.identity,
  [integer.identity],
  [{ name: "value", type: integer.identity }],
);
const strError = shape(
  "error",
  project.identity,
  [text.identity],
  [{ name: "value", type: text.identity }],
);
const plan = {
  declarations: [{ ...builtin, parameters: 0 }, project],
  shapes: [integer, text, codec, intError, strError],
};
const origin = { source: "app/main.can", start: 10, end: 20, invocation: ["main", "native0"] };

test("domain occurrences preserve exact payload, allocation and private origin", () => {
  const runtime = createDomainRuntime(plan);
  const payload = record(intError.identity, [["value", 1n]]);
  const cause = { credential: "secret" };
  const first = runtime.create(intError.identity, payload, origin, cause);
  const second = runtime.create(intError.identity, payload, origin, cause);
  const details = domainFailureDiagnostics(first);
  expect(details.declaration.identity).toBe(project.identity);
  expect(details.declaration.name).toBe("app::failed");
  expect(details.typeArguments).toEqual([integer.identity]);
  expect(details.payload).toBe(payload);
  expect(details.cause).toBe(cause);
  expect(details.origin).toEqual(origin);
  expect(details.occurrenceID).not.toBe(domainFailureDiagnostics(second).occurrenceID);
  expect(details.occurrenceID).not.toBe(
    standardFailureOccurrenceID(captureStandard(cause, origin)),
  );
  expect(runtime.checkBound(first, [intError.identity])).toBe(first);
  expect(() => runtime.checkBound(first, [strError.identity])).toThrow("undeclared escaping");
  expect(isDomainFailure(first)).toBe(true);
  expect(isDomainFailure(new Proxy(first, {}))).toBe(false);
  expect(isDomainFailure(payload)).toBe(false);
  expect(Object.isFrozen(first)).toBe(true);
  expect(Reflect.ownKeys(first)).toEqual([]);
  expect(JSON.stringify(first)).toBe("{}");
  const other = createDomainRuntime({
    ...plan,
    declarations: [
      { ...builtin, parameters: 0 },
      { ...project, name: "app::other" },
    ],
  });
  expect(() => other.checkBound(first, [intError.identity])).toThrow("undeclared escaping");
  const mutable = { ...origin, invocation: ["original"] };
  const captured = runtime.create(intError.identity, payload, mutable);
  mutable.invocation[0] = "changed";
  expect(domainFailureDiagnostics(captured).origin.invocation).toEqual(["original"]);
});

test("malformed payloads and hostile identity metadata fail without getters or traps", () => {
  const runtime = createDomainRuntime(plan);
  let invoked = 0;
  const trap = () => {
    invoked++;
    throw new Error("secret");
  };
  const forged = {
    get path() {
      return trap();
    },
    get reason() {
      return trap();
    },
  };
  const proxy = new Proxy(forged, {
    get: trap,
    ownKeys: trap,
    getOwnPropertyDescriptor: trap,
    getPrototypeOf: trap,
    isExtensible: trap,
  });
  const revoked = Proxy.revocable({}, {});
  revoked.revoke();
  for (const value of [
    forged,
    proxy,
    revoked.proxy,
    null,
    { path: "x", reason: "y" },
    record(codec.identity, [["path", "x"]]),
    record(codec.identity, [
      ["path", 1n],
      ["reason", "y"],
    ]),
    record(codec.identity, [
      ["path", "x"],
      ["reason", "y"],
      ["extra", "secret"],
    ]),
    record(intError.identity, [
      ["path", "x"],
      ["reason", "y"],
    ]),
  ]) {
    expect(runtime.accepts(codec.identity, value)).toBe(false);
    expect(() => runtime.create(codec.identity, value, origin)).toThrow(
      "invalid domain error payload",
    );
  }
  const valid = record(codec.identity, [
    ["path", "root"],
    ["reason", "bad"],
  ]);
  expect(runtime.accepts(codec.identity, valid)).toBe(true);
  expect(runtime.accepts(codec.identity, { ...valid })).toBe(false);
  const identity = errorIdentity("codec::invalid_data");
  for (const value of [
    proxy,
    revoked.proxy,
    {
      ...identity,
      get name() {
        return trap();
      },
    },
    { ...identity, typeArguments: new Proxy([], { get: trap, ownKeys: trap }) },
    { ...identity, extra: "secret" },
  ])
    expect(() => validateErrorIdentity(value as any)).toThrow();
  const args: string[] = [];
  Object.defineProperty(args, "0", { get: trap });
  expect(() => errorIdentity("all_failed", args)).toThrow();
  expect(() => errorIdentity("all_failed", Array(1))).toThrow();
  expect(invoked).toBe(0);
});

test("compiler plan rejects wrong declarations, catalogue fields, and variant cycles", () => {
  expect(() =>
    createDomainRuntime({
      ...plan,
      declarations: [...plan.declarations, { ...project }],
    }),
  ).toThrow("duplicate");
  expect(() =>
    createDomainRuntime({
      ...plan,
      declarations: [{ ...builtin, name: "codec::tampered", parameters: 0 }],
    }),
  ).toThrow("catalogue declaration mismatch");
  expect(() =>
    createDomainRuntime({
      ...plan,
      shapes: plan.shapes.map((s) =>
        s === codec
          ? {
              ...s,
              fields: [
                { name: "path", type: integer.identity },
                { name: "reason", type: text.identity },
              ],
            }
          : s,
      ),
    }),
  ).toThrow("catalogue payload field mismatch");
  const loop = shape("variant", "can.project.root::app::loop");
  expect(() =>
    createDomainRuntime({
      ...plan,
      shapes: [...plan.shapes, { ...loop, leaves: [loop.identity] }],
    }),
  ).toThrow("invalid concrete variant leaves");
  const none = shape("record", "can.std.option@1::none");
  const some = shape(
    "record",
    "can.std.option@1::some",
    [integer.identity],
    [{ name: "value", type: integer.identity }],
  );
  const option = shape(
    "variant",
    "can.std.option@1::value",
    [integer.identity],
    [],
    [none.identity, some.identity],
  );
  expect(() =>
    createDomainRuntime({ ...plan, shapes: [...plan.shapes, none, some, option] }),
  ).not.toThrow();
  expect(() =>
    createDomainRuntime({
      ...plan,
      shapes: [
        ...plan.shapes,
        none,
        some,
        { ...option, leaves: [none.identity, intError.identity] },
      ],
    }),
  ).toThrow("catalogue variant leaves mismatch");
});

test("same error kind under two qualified identities composes without numeric allocation", () => {
  const left = {
    identity: "can.project.lineage/shop_left/model::failed",
    name: "model::failed",
    parameters: 0,
  };
  const right = {
    identity: "can.project.lineage/shop_right/model::failed",
    name: "model::failed",
    parameters: 0,
  };
  const leftShape = shape("error", left.identity, [], [{ name: "reason", type: text.identity }]);
  const rightShape = shape("error", right.identity, [], [{ name: "reason", type: text.identity }]);
  const runtime = createDomainRuntime({
    declarations: [left, right],
    shapes: [text, leftShape, rightShape],
  });
  const payload = record(leftShape.identity, [["reason", "left"]]);
  const occurrence = runtime.create(leftShape.identity, payload, origin);
  expect(domainFailureDiagnostics(occurrence).declaration.identity).toBe(left.identity);
  expect(runtime.checkBound(occurrence, [leftShape.identity])).toBe(occurrence);
  expect(() => runtime.checkBound(occurrence, [rightShape.identity])).toThrow(
    "undeclared escaping",
  );
});

test("recursive payloads admit shared immutable data and reject circular native values", () => {
  const tree = shape("record", "can.project.root::app::tree");
  const children = {
    ...shape("array", "", [tree.identity]),
    arguments: [],
    element: tree.identity,
  };
  const treeShape = { ...tree, fields: [{ name: "children", type: children.identity }] };
  const treeError = shape(
    "error",
    project.identity,
    [tree.identity],
    [{ name: "value", type: tree.identity }],
  );
  const runtime = createDomainRuntime({
    declarations: [project],
    shapes: [treeShape, children, treeError],
  });
  const leaf = record(tree.identity, [["children", array([])]]);
  const shared = record(tree.identity, [["children", array([leaf, leaf])]]);
  expect(runtime.accepts(treeError.identity, record(treeError.identity, [["value", shared]]))).toBe(
    true,
  );
  const backing: unknown[] = [];
  const cyclic = record(tree.identity, [["children", backing]]);
  backing.push(cyclic);
  Object.freeze(backing);
  expect(runtime.accepts(tree.identity, cyclic)).toBe(false);
  expect(runtime.accepts(tree.identity, record(tree.identity, [["children", []]]))).toBe(false);
  const sparse = Array(2);
  Object.freeze(sparse);
  expect(runtime.accepts(children.identity, sparse)).toBe(false);
});

test("repeated factories keep separate snapshots and fresh occurrences", () => {
  const declarations = [{ ...builtin, parameters: 0 }, { ...project }];
  const shapes = [integer, text, codec, intError, strError];
  const first = createDomainRuntime({ declarations, shapes });
  const second = createDomainRuntime({ declarations, shapes });
  (declarations[1] as { name: string }).name = "app::mutated";
  declarations.push({
    identity: "can.project.root::app::extra",
    name: "app::extra",
    parameters: 0,
  });
  shapes.push({ ...intError });
  const payload = record(intError.identity, [["value", 1n]]);
  const a = first.create(intError.identity, payload, origin);
  const b = second.create(intError.identity, payload, origin);
  expect(domainFailureDiagnostics(a).declaration.name).toBe("app::failed");
  expect(domainFailureDiagnostics(b).declaration.name).toBe("app::failed");
  expect(domainFailureDiagnostics(a).occurrenceID).not.toBe(
    domainFailureDiagnostics(b).occurrenceID,
  );
  expect(first.checkBound(b, [intError.identity])).toBe(b);
  expect(first.accepts(intError.identity, payload)).toBe(true);
  expect(second.accepts(intError.identity, payload)).toBe(true);
});

test("warm factories preserve declaration and digest rejection precedence", () => {
  const tampered = { ...codec, identity: "0".repeat(64) };
  expect(() =>
    createDomainRuntime({
      declarations: [...plan.declarations, { ...project }],
      shapes: plan.shapes.map((s) => (s === codec ? tampered : s)),
    }),
  ).toThrow("duplicate");
  const unknownBad = {
    ...shape("record", "can.unknown@1::nope"),
    identity: "1".repeat(64),
  };
  expect(() => createDomainRuntime({ ...plan, shapes: [...plan.shapes, unknownBad] })).toThrow(
    "concrete failure type identity mismatch",
  );
  const unknown = shape("record", "can.unknown@1::nope");
  expect(() => createDomainRuntime({ ...plan, shapes: [...plan.shapes, unknown] })).toThrow(
    "unknown nominal payload declaration",
  );
  expect(() =>
    createDomainRuntime({
      ...plan,
      declarations: [{ ...builtin, name: "codec::tampered", parameters: 0 }],
    }),
  ).toThrow("catalogue declaration mismatch");
});

test("every factory validates every shape digest independently", () => {
  const expected = new Set(plan.shapes.map((s) => s.identity));
  for (let factory = 0; factory < 2; factory++) {
    const claimed: string[] = [];
    const runtime = createDomainRuntimeWithDigest(plan, (input, identity) => {
      claimed.push(identity);
      return createHash("sha256").update(input).digest("hex");
    });
    expect(new Set(claimed)).toEqual(expected);
    const payload = record(codec.identity, [
      ["path", "p"],
      ["reason", "r"],
    ]);
    expect(runtime.accepts(codec.identity, payload)).toBe(true);
  }
});

test("rejected plans leave later factories unaffected", () => {
  expect(() =>
    createDomainRuntime({
      ...plan,
      declarations: [{ ...builtin, name: "codec::wrong", parameters: 0 }],
    }),
  ).toThrow("catalogue declaration mismatch");
  const tampered = { ...codec, identity: "2".repeat(64) };
  expect(() =>
    createDomainRuntime({
      ...plan,
      shapes: plan.shapes.map((s) => (s === codec ? tampered : s)),
    }),
  ).toThrow("identity mismatch");
  const runtime = createDomainRuntime(plan);
  const payload = record(codec.identity, [
    ["path", "p"],
    ["reason", "r"],
  ]);
  expect(runtime.accepts(codec.identity, payload)).toBe(true);
});
