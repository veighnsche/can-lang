// NT-P28 base owner/channel contract: grant admission shapes, opaque
// handles, framed envelope FIFOs, and the exact finite failure vocabulary.
// test::grant_admit/release/open/send/recv/close/pending lower to this
// adapter pair; issuance verification and the N link arrive in later
// slices, so loopback behavior and closed-vocabulary admission are what
// this file pins. Markers test:: and can.std.test@1 keep the NT-P28
// catalogue package referenced from runtime test evidence.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { success, value, type Completion } from "../completion.ts";
import { array, dataProperty, record } from "../data.ts";
import { ownBytes } from "../bytes.ts";
import { createJSONValueCodec, type JSONValueIds } from "../codec/value.ts";
import { createTestOwner } from "../test-support/owner.ts";
import {
  CHANNEL_KINDS,
  createTestSupport,
  createTestTransport,
} from "../test-support/transport.ts";

const identity = (kind: string, declaration: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration]))
    .digest("hex");
const scalar = (name: string): FailureShape => ({
  identity: identity("primitive", name),
  kind: "primitive",
  declaration: name,
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const text = scalar("str");
const integer = scalar("int");
const declarations = catalogue.errors.filter((error) => error.name.startsWith("test::"));
const codecDecl = catalogue.errors.find((error) => error.name === "codec::invalid_data");
if (codecDecl === undefined) throw Error("codec::invalid_data missing from catalogue");
const S1C_ERRORS = [
  "files::not_found",
  "files::already_exists",
  "files::denied",
  "files::unexpected_kind",
  "files::limit_exceeded",
  "files::io_error",
  "process::spawn_failed",
  "process::timeout",
  "process::output_limit",
  "process::invalid_config",
  "process::io_error",
];
const s1cDecls = S1C_ERRORS.map((name) => {
  const found = catalogue.errors.find((error) => error.name === name);
  if (found === undefined) throw Error(`${name} missing from catalogue`);
  return found;
});
const shapeOf = (id: string, fields: readonly { name: string; type: string }[]): FailureShape => ({
  identity: identity("error", id),
  kind: "error",
  declaration: id,
  arguments: [],
  fields: fields.map((field) => ({
    name: field.name,
    type: field.type === "int" ? integer.identity : text.identity,
  })),
  leaves: [],
  inputs: [],
  errors: [],
});
const errorShapes: Map<string, FailureShape> = new Map(
  [...declarations, codecDecl, ...s1cDecls].map((error) => [
    error.name,
    shapeOf(error.identity, error.fields),
  ]),
);
const domain = createDomainRuntime({
  declarations: [...declarations, codecDecl, ...s1cDecls].map((error) => ({
    identity: error.identity,
    name: error.name,
    parameters: 0,
  })),
  shapes: [text, integer, ...errorShapes.values()],
});
const err = (name: string): string => {
  const shape = errorShapes.get(name);
  if (shape === undefined) throw Error(`no shape for ${name}`);
  return shape.identity;
};

const prefix = "test:json_";
const ids = Object.fromEntries(
  ["null", "bool", "int", "float", "string", "array", "object", "member"].map((kind) => [
    kind,
    prefix + kind,
  ]),
) as JSONValueIds;
const codec = createJSONValueCodec(domain, err("codec::invalid_data"), ids);

function rig() {
  const owner = createTestOwner(domain, { invalidGrant: err("test::invalid_grant") });
  const transport = createTestTransport(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      closedHandle: err("test::closed_handle"),
      transportFailure: err("test::transport_failure"),
      channelFull: err("test::channel_full"),
      detachedTransport: err("test::detached_transport"),
      channelEmpty: err("test::channel_empty"),
      invalidKind: err("test::invalid_kind"),
    },
    codec,
    owner,
    ids.object,
  );
  return { owner, transport };
}

const GRANT = "ng1-0123456789abcdef0123456789abcdef";
const MEMBER = (name: string, content: unknown) =>
  record(ids.member, [
    ["name", name],
    ["value", content],
  ]);
const envelope = () =>
  record(ids.object, [
    ["members", array([MEMBER("op", record(ids.string, [["value", "probe"]]))])],
  ]);

async function failed(completion: Promise<Completion<never>>) {
  const result = await completion;
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain failure");
  return domainFailureDiagnostics(result.value);
}

test("can.std.test@1 grant admission mints opaque handles and rejects malformed grants", async () => {
  const { owner } = rig();
  const admitted = await owner.admitGrant(GRANT);
  expect(admitted.kind).toBe("ok");
  const handle = value(admitted);
  expect(Object.isFrozen(handle)).toBe(true);
  expect(JSON.stringify(handle)).not.toContain("ng1-");
  for (const [grant, reason] of [
    ["", "empty"],
    ["ng1-short", "bad-length"],
    [`xx1-${"0".repeat(32)}`, "bad-prefix"],
    [`ng1-${"z".repeat(32)}`, "bad-charset"],
  ] as const) {
    const details = await failed(owner.admitGrant(grant) as Promise<Completion<never>>);
    expect(details.declaration.name).toBe("test::invalid_grant");
    expect(details.payload).toMatchObject({ reason });
  }
});

test("test::grant_release is idempotent and release cascades to channels", async () => {
  const { owner, transport } = rig();
  const handle = value(await owner.admitGrant(GRANT));
  const channel = value(await transport.openChannel(handle, "dispatch"));
  expect(value(await owner.releaseGrant(handle))).toBe(undefined);
  expect(value(await owner.releaseGrant(handle))).toBe(undefined);
  const stale = await failed(
    transport.openChannel(handle, "dispatch") as Promise<Completion<never>>,
  );
  expect(stale.declaration.name).toBe("test::stale_handle");
  expect(stale.payload).toMatchObject({ handle: "owner#1" });
  const sendStale = await failed(
    transport.sendEnvelope(channel, envelope()) as Promise<Completion<never>>,
  );
  expect(sendStale.declaration.name).toBe("test::stale_handle");
  expect(sendStale.payload).toMatchObject({ handle: "channel#1" });
});

test("test::channel_open admits the closed kind vocabulary", async () => {
  const { owner, transport } = rig();
  const handle = value(await owner.admitGrant(GRANT));
  expect([...CHANNEL_KINDS]).toEqual(["dispatch", "report"]);
  for (const kind of CHANNEL_KINDS) {
    expect((await transport.openChannel(handle, kind)).kind).toBe("ok");
  }
  const details = await failed(transport.openChannel(handle, "exec") as Promise<Completion<never>>);
  expect(details.declaration.name).toBe("test::invalid_kind");
  expect(details.payload).toMatchObject({ kind: "exec" });
});

test("test::channel_send/recv/pending round trip framed envelopes", async () => {
  const { owner, transport } = rig();
  const handle = value(await owner.admitGrant(GRANT));
  const channel = value(await transport.openChannel(handle, "dispatch"));
  expect(value(await transport.pendingDepth(channel))).toBe(0n);
  const bytes = value(await transport.sendEnvelope(channel, envelope()));
  expect(typeof bytes).toBe("bigint");
  expect(bytes > 0n).toBe(true);
  expect(value(await transport.pendingDepth(channel))).toBe(1n);
  const decoded = value(await transport.recvEnvelope(channel));
  expect(value(await transport.pendingDepth(channel))).toBe(0n);
  const members = dataProperty(decoded, "members") as readonly unknown[];
  expect(members.length).toBe(1);
  expect(dataProperty(members[0], "name")).toBe("op");
  expect(dataProperty(dataProperty(members[0], "value"), "value")).toBe("probe");
});

test("test::channel_close drains then seals; empty recv names channel_empty", async () => {
  const { owner, transport } = rig();
  const handle = value(await owner.admitGrant(GRANT));
  const channel = value(await transport.openChannel(handle, "report"));
  const empty = await failed(transport.recvEnvelope(channel) as Promise<Completion<never>>);
  expect(empty.declaration.name).toBe("test::channel_empty");
  await transport.sendEnvelope(channel, envelope());
  expect(value(await transport.closeChannel(channel))).toBe(undefined);
  expect(value(await transport.closeChannel(channel))).toBe(undefined);
  expect((await transport.recvEnvelope(channel)).kind).toBe("ok");
  const sealed = await failed(transport.recvEnvelope(channel) as Promise<Completion<never>>);
  expect(sealed.declaration.name).toBe("test::closed_handle");
  const sendSealed = await failed(
    transport.sendEnvelope(channel, envelope()) as Promise<Completion<never>>,
  );
  expect(sendSealed.declaration.name).toBe("test::closed_handle");
});

test("test::channel_full caps depth without retaining the overflow", async () => {
  const { owner, transport } = rig();
  const handle = value(await owner.admitGrant(GRANT));
  const channel = value(await transport.openChannel(handle, "dispatch"));
  for (let i = 0; i < 64; i++) {
    expect((await transport.sendEnvelope(channel, envelope())).kind).toBe("ok");
  }
  const full = await failed(
    transport.sendEnvelope(channel, envelope()) as Promise<Completion<never>>,
  );
  expect(full.declaration.name).toBe("test::channel_full");
  expect(value(await transport.pendingDepth(channel))).toBe(64n);
});

test("detach severs send/recv but keeps depth and close observable", async () => {
  const { owner, transport } = rig();
  const handle = value(await owner.admitGrant(GRANT));
  const channel = value(await transport.openChannel(handle, "dispatch"));
  await transport.sendEnvelope(channel, envelope());
  transport.detach();
  transport.detach();
  const sendCut = await failed(
    transport.sendEnvelope(channel, envelope()) as Promise<Completion<never>>,
  );
  expect(sendCut.declaration.name).toBe("test::detached_transport");
  const recvCut = await failed(transport.recvEnvelope(channel) as Promise<Completion<never>>);
  expect(recvCut.declaration.name).toBe("test::detached_transport");
  expect(value(await transport.pendingDepth(channel))).toBe(1n);
  expect(value(await transport.closeChannel(channel))).toBe(undefined);
});

test("foreign handles are stale on fallible ops and loud on release/close", async () => {
  const first = rig();
  const second = rig();
  const handle = value(await first.owner.admitGrant(GRANT));
  const channel = value(await first.transport.openChannel(handle, "dispatch"));
  const stale = await failed(
    second.transport.sendEnvelope(channel, envelope()) as Promise<Completion<never>>,
  );
  expect(stale.declaration.name).toBe("test::stale_handle");
  await expect(second.owner.releaseGrant(handle)).rejects.toThrow(TypeError);
  await expect(second.transport.closeChannel(channel)).rejects.toThrow(TypeError);
});

test("raw thenables and plain objects are rejected as channel handles", async () => {
  const { transport } = rig();
  // oxlint-disable no-thenable -- The impostor deliberately exposes then to prove raw thenables are rejected.
  const rawThenable = { then: () => undefined };
  // oxlint-enable no-thenable
  const plainObject = { queue: [] };
  for (const impostor of [rawThenable, plainObject, null, undefined, 42, "channel#1"]) {
    await expect(transport.sendEnvelope(impostor, envelope())).rejects.toThrow(TypeError);
    await expect(transport.recvEnvelope(impostor)).rejects.toThrow(TypeError);
    await expect(transport.closeChannel(impostor)).rejects.toThrow(TypeError);
    await expect(transport.pendingDepth(impostor)).rejects.toThrow(TypeError);
  }
});

test("codec and frame faults map onto test::transport_failure reasons", async () => {
  const { owner, transport } = rig();
  const handle = value(await owner.admitGrant(GRANT));
  const channel = value(await transport.openChannel(handle, "dispatch"));
  // A mistagged record fails the real codec encode.
  const mistagged = record("bogus", [["members", []]]);
  const encodeFault = await failed(
    transport.sendEnvelope(channel, mistagged) as Promise<Completion<never>>,
  );
  expect(encodeFault.declaration.name).toBe("test::transport_failure");
  expect(encodeFault.payload).toMatchObject({ reason: "encode" });
  expect(value(await transport.pendingDepth(channel))).toBe(0n);
  // A stub byte source stands in for the future N link; decode below is the
  // real codec. Malformed frames fail decode; wellformed non-object frames
  // fail the object guard. Neither fault retains a frame.
  const byteSource = (text: string) => ({
    encode: async () => success(ownBytes(new TextEncoder().encode(text))),
    decode: codec.decode,
  });
  const malformed = createTestTransport(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      closedHandle: err("test::closed_handle"),
      transportFailure: err("test::transport_failure"),
      channelFull: err("test::channel_full"),
      detachedTransport: err("test::detached_transport"),
      channelEmpty: err("test::channel_empty"),
      invalidKind: err("test::invalid_kind"),
    },
    byteSource('{"members":'),
    owner,
    ids.object,
  );
  const badChannel = value(await malformed.openChannel(handle, "report"));
  await malformed.sendEnvelope(badChannel, envelope());
  const decodeFault = await failed(
    malformed.recvEnvelope(badChannel) as Promise<Completion<never>>,
  );
  expect(decodeFault.payload).toMatchObject({ reason: "decode" });
  const scalar = createTestTransport(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      closedHandle: err("test::closed_handle"),
      transportFailure: err("test::transport_failure"),
      channelFull: err("test::channel_full"),
      detachedTransport: err("test::detached_transport"),
      channelEmpty: err("test::channel_empty"),
      invalidKind: err("test::invalid_kind"),
    },
    byteSource("42"),
    owner,
    ids.object,
  );
  const scalarChannel = value(await scalar.openChannel(handle, "report"));
  await scalar.sendEnvelope(scalarChannel, envelope());
  const shapeFault = await failed(scalar.recvEnvelope(scalarChannel) as Promise<Completion<never>>);
  expect(shapeFault.payload).toMatchObject({ reason: "non-object-frame" });
});

test("createTestSupport composes one shared owner table for $canTest", async () => {
  const support = createTestSupport(
    domain,
    {
      invalidGrant: err("test::invalid_grant"),
      staleHandle: err("test::stale_handle"),
      closedHandle: err("test::closed_handle"),
      transportFailure: err("test::transport_failure"),
      channelFull: err("test::channel_full"),
      detachedTransport: err("test::detached_transport"),
      channelEmpty: err("test::channel_empty"),
      invalidKind: err("test::invalid_kind"),
      unknownTool: err("test::unknown_tool"),
      invalidPath: err("test::invalid_path"),
      invalidName: err("test::invalid_name"),
      notFound: err("files::not_found"),
      alreadyExists: err("files::already_exists"),
      denied: err("files::denied"),
      unexpectedKind: err("files::unexpected_kind"),
      limitExceeded: err("files::limit_exceeded"),
      invalidData: err("codec::invalid_data"),
      ioError: err("files::io_error"),
      spawnFailed: err("process::spawn_failed"),
      timeout: err("process::timeout"),
      outputLimit: err("process::output_limit"),
      invalidConfig: err("process::invalid_config"),
      processIoError: err("process::io_error"),
      toolResult: "test:tool_result",
      receipt: "test:evidence_receipt",
    },
    codec,
    ids.object,
    new Map(),
  );
  expect(Object.isFrozen(support)).toBe(true);
  const handle = value(await support.owner.admitGrant(GRANT));
  const channel = value(await support.transport.openChannel(handle, "dispatch"));
  await support.transport.sendEnvelope(channel, envelope());
  expect(value(await support.transport.pendingDepth(channel))).toBe(1n);
  await support.owner.releaseGrant(handle);
  const stale = await failed(support.transport.pendingDepth(channel) as Promise<Completion<never>>);
  expect(stale.declaration.name).toBe("test::stale_handle");
});

test("oversize frames fail without queueing", async () => {
  const { owner, transport } = rig();
  const handle = value(await owner.admitGrant(GRANT));
  const channel = value(await transport.openChannel(handle, "dispatch"));
  const big = record(ids.object, [
    ["members", array([MEMBER("blob", record(ids.string, [["value", "x".repeat(1048576)]]))])],
  ]);
  const fault = await failed(transport.sendEnvelope(channel, big) as Promise<Completion<never>>);
  expect(fault.payload).toMatchObject({ reason: "frame-too-large" });
  expect(value(await transport.pendingDepth(channel))).toBe(0n);
});
