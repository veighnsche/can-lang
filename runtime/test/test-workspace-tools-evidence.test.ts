// NT-P28 S1c workspace/tools/evidence contract: owner-scoped scratch
// directories with relative-path confinement, keyed tool dispatch over
// the shared spawn discipline, and sealed evidence bundles with
// closed-vocabulary receipt kinds. Markers test:: and can.std.test@1
// keep the NT-P28 catalogue package referenced from runtime test
// evidence.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { existsSync } from "node:fs";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { success, value, type Completion } from "../completion.ts";
import { array, dataProperty, record } from "../data.ts";
import { isBytes, ownBytes } from "../bytes.ts";
import { createTestOwner } from "../test-support/owner.ts";
import { RECEIPT_KINDS, createTestEvidence } from "../test-support/evidence.ts";
import { createTestTools, type ToolTable } from "../test-support/tools.ts";
import { createTestWorkspace } from "../test-support/workspace.ts";
import { runOwnedRoot } from "../owner.ts";

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
const ERROR_NAMES = [
  ...catalogue.errors.filter((error) => error.name.startsWith("test::")).map((e) => e.name),
  "codec::invalid_data",
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
const decls = ERROR_NAMES.map((name) => {
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
  decls.map((error) => [error.name, shapeOf(error.identity, error.fields)]),
);
const domain = createDomainRuntime({
  declarations: decls.map((error) => ({
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

const GRANT = "ng1-0123456789abcdef0123456789abcdef";
const TOOL_RESULT = "test:tool_result";
const RECEIPT = "test:evidence_receipt";

function rig(tools: ToolTable = new Map()) {
  const owner = createTestOwner(domain, { invalidGrant: err("test::invalid_grant") });
  const workspace = createTestWorkspace(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      closedHandle: err("test::closed_handle"),
      invalidPath: err("test::invalid_path"),
      notFound: err("files::not_found"),
      alreadyExists: err("files::already_exists"),
      denied: err("files::denied"),
      unexpectedKind: err("files::unexpected_kind"),
      limitExceeded: err("files::limit_exceeded"),
      invalidData: err("codec::invalid_data"),
      ioError: err("files::io_error"),
    },
    owner,
  );
  const toolRunner = createTestTools(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      unknownTool: err("test::unknown_tool"),
      notFound: err("files::not_found"),
      denied: err("files::denied"),
      spawnFailed: err("process::spawn_failed"),
      timeout: err("process::timeout"),
      outputLimit: err("process::output_limit"),
      invalidConfig: err("process::invalid_config"),
      processIoError: err("process::io_error"),
      toolResult: TOOL_RESULT,
    },
    owner,
    tools,
  );
  const evidence = createTestEvidence(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      closedHandle: err("test::closed_handle"),
      invalidName: err("test::invalid_name"),
      invalidKind: err("test::invalid_kind"),
      receipt: RECEIPT,
    },
    owner,
  );
  return { owner, workspace, tools: toolRunner, evidence };
}

async function failed(completion: Promise<Completion<never>>) {
  const result = await completion;
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain failure");
  return domainFailureDiagnostics(result.value);
}

async function owned<T>(body: () => Promise<Completion<T>>): Promise<Completion<T>> {
  const root = await runOwnedRoot(body);
  expect(root.cleanupFailed).toBe(false);
  return root.completion;
}

type Over = Partial<{
  cwd: string;
  inherit: boolean;
  env: string[];
  stdin: Uint8Array;
  out: bigint;
  err: bigint;
  deadline: bigint;
  grace: bigint;
}>;
const options = (over: Over = {}) =>
  record("test-options", [
    ["cwd", over.cwd ?? ""],
    ["inherit_env", over.inherit ?? false],
    ["env", array(over.env ?? [])],
    ["stdin", ownBytes(over.stdin ?? new Uint8Array())],
    ["stdout_limit", over.out ?? 1000000n],
    ["stderr_limit", over.err ?? 1000000n],
    ["deadline_ms", over.deadline ?? 0n],
    ["grace_ms", over.grace ?? 100n],
  ]);

function frame(name: string, data: Uint8Array): Uint8Array {
  const nameBytes = new TextEncoder().encode(name);
  const out = new Uint8Array(8 + nameBytes.byteLength + 8 + data.byteLength);
  const view = new DataView(out.buffer);
  view.setBigUint64(0, BigInt(nameBytes.byteLength));
  out.set(nameBytes, 8);
  view.setBigUint64(8 + nameBytes.byteLength, BigInt(data.byteLength));
  out.set(data, 8 + nameBytes.byteLength + 8);
  return out;
}

test("workspace stage/run/read/close roundtrip removes the tree", async () => {
  const { owner, workspace } = rig();
  const root = await runOwnedRoot(async () => {
    const handle = value(await owner.admitGrant(GRANT));
    const ws = value(await workspace.openWorkspace(handle));
    expect(Object.isFrozen(ws)).toBe(true);
    await workspace.workspaceMkdir(ws, "stage/bin", true);
    await workspace.workspaceWriteText(ws, "stage/bin/tool.sh", "#!/bin/sh\necho staged\n", false);
    const back = value(await workspace.workspaceReadText(ws, "stage/bin/tool.sh", 1000n));
    expect(back).toBe("#!/bin/sh\necho staged\n");
    await workspace.closeWorkspace(ws);
    await workspace.closeWorkspace(ws);
    return success(undefined);
  });
  expect(root.cleanupFailed).toBe(false);
});

test("workspace close removes the root from the filesystem", async () => {
  const { owner, workspace } = rig();
  let probed = "";
  await runOwnedRoot(async () => {
    const handle = value(await owner.admitGrant(GRANT));
    const ws = value(await workspace.openWorkspace(handle));
    await workspace.workspaceWriteText(ws, "marker.txt", "x", false);
    // Resolve the root through a delegated failure field: missing files
    // name the resolved absolute path.
    const missing = await failed(
      workspace.workspaceReadText(ws, "nope.txt", 100n) as Promise<Completion<never>>,
    );
    expect(missing.declaration.name).toBe("files::not_found");
    probed = (missing.payload as Record<string, string>)["path"].replace(/nope\.txt$/, "");
    expect(existsSync(probed)).toBe(true);
    await workspace.closeWorkspace(ws);
    return success(undefined);
  });
  expect(probed).not.toBe("");
  expect(existsSync(probed)).toBe(false);
});

test("workspace confinement rejects absolute, escaping and malformed paths", async () => {
  const { owner, workspace } = rig();
  await runOwnedRoot(async () => {
    const handle = value(await owner.admitGrant(GRANT));
    const ws = value(await workspace.openWorkspace(handle));
    for (const [path, reason] of [
      ["", "empty"],
      ["/etc/hosts", "absolute"],
      ["C:\\win", "absolute"],
      ["a\0b", "nul_byte"],
      ["../out", "parent_escape"],
      ["a/../../out", "parent_escape"],
    ] as const) {
      for (const call of [
        workspace.workspaceMkdir(ws, path, false),
        workspace.workspaceWriteText(ws, path, "x", true),
        workspace.workspaceReadText(ws, path, 10n),
      ]) {
        const details = await failed(call as Promise<Completion<never>>);
        expect(details.declaration.name).toBe("test::invalid_path");
        expect(details.payload).toMatchObject({ reason });
      }
    }
    // Dots and doubled separators normalize inside the root.
    await workspace.workspaceWriteText(ws, "./dot//file.txt", "ok", false);
    expect(value(await workspace.workspaceReadText(ws, "dot/file.txt", 10n))).toBe("ok");
    await workspace.closeWorkspace(ws);
    return success(undefined);
  });
});

test("workspace delegated failures mirror the files vocabulary", async () => {
  const { owner, workspace } = rig();
  await runOwnedRoot(async () => {
    const handle = value(await owner.admitGrant(GRANT));
    const ws = value(await workspace.openWorkspace(handle));
    const missing = await failed(
      workspace.workspaceReadText(ws, "nope.txt", 100n) as Promise<Completion<never>>,
    );
    expect(missing.declaration.name).toBe("files::not_found");
    await workspace.workspaceWriteText(ws, "once.txt", "1", false);
    const clash = await failed(
      workspace.workspaceWriteText(ws, "once.txt", "2", false) as Promise<Completion<never>>,
    );
    expect(clash.declaration.name).toBe("files::already_exists");
    await workspace.workspaceWriteText(ws, "once.txt", "2", true);
    await workspace.workspaceMkdir(ws, "dir", false);
    const dirRead = await failed(
      workspace.workspaceReadText(ws, "dir", 100n) as Promise<Completion<never>>,
    );
    expect(dirRead.declaration.name).toBe("files::unexpected_kind");
    const over = await failed(
      workspace.workspaceReadText(ws, "once.txt", 1n) as Promise<Completion<never>>,
    );
    expect(over.declaration.name).toBe("files::limit_exceeded");
    const noParent = await failed(
      workspace.workspaceMkdir(ws, "ghost/child", false) as Promise<Completion<never>>,
    );
    expect(noParent.declaration.name).toBe("files::not_found");
    await workspace.closeWorkspace(ws);
    return success(undefined);
  });
});

test("workspace handles go stale on release and closed ops reject", async () => {
  const { owner, workspace } = rig();
  await runOwnedRoot(async () => {
    const handle = value(await owner.admitGrant(GRANT));
    const ws = value(await workspace.openWorkspace(handle));
    await workspace.workspaceWriteText(ws, "a.txt", "a", false);
    await workspace.closeWorkspace(ws);
    const closed = await failed(
      workspace.workspaceReadText(ws, "a.txt", 10n) as Promise<Completion<never>>,
    );
    expect(closed.declaration.name).toBe("test::closed_handle");
    const ws2 = value(await workspace.openWorkspace(handle));
    await owner.releaseGrant(handle);
    for (const call of [
      workspace.workspaceMkdir(ws2, "x", false),
      workspace.workspaceWriteText(ws2, "x", "y", true),
      workspace.workspaceReadText(ws2, "a.txt", 10n),
      workspace.openWorkspace(handle),
    ]) {
      const details = await failed(call as Promise<Completion<never>>);
      expect(details.declaration.name).toBe("test::stale_handle");
    }
    // Close stays infallible over stale handles.
    await workspace.closeWorkspace(ws2);
    return success(undefined);
  });
});

test("run_tool dispatches registered keys and rejects unknown keys pre-spawn", async () => {
  const tools: ToolTable = new Map([
    ["echo", { executable: "/bin/echo" }],
    ["false", { executable: "/usr/bin/false" }],
  ]);
  const { owner, tools: runner } = rig(tools);
  const unknown = await failed(
    runner.runTool(value(await owner.admitGrant(GRANT)), "nope", [], options()) as Promise<
      Completion<never>
    >,
  );
  expect(unknown.declaration.name).toBe("test::unknown_tool");
  expect(unknown.payload).toMatchObject({ key: "nope" });
  const done = await owned(async () => {
    const handle = value(await owner.admitGrant(GRANT));
    return runner.runTool(handle, "echo", ["hello"], options());
  });
  expect(done.kind).toBe("ok");
  const result = value(done);
  expect(dataProperty(result, "code")).toBe(0n);
  expect(dataProperty(result, "signal")).toBe("");
  const stdout = dataProperty(result, "stdout");
  expect(isBytes(stdout)).toBe(true);
  const failedRun = await owned(async () => {
    const handle = value(await owner.admitGrant(GRANT));
    return runner.runTool(handle, "false", [], options());
  });
  expect(value(failedRun) && dataProperty(value(failedRun), "code")).toBe(1n);
});

test("run_tool inherits the spawn failure vocabulary", async () => {
  const tools: ToolTable = new Map([["echo", { executable: "/bin/echo" }]]);
  const { owner, tools: runner } = rig(tools);
  const handle = value(await owner.admitGrant(GRANT));
  const stale = runner.runTool("bogus-owner", "echo", [], options());
  await expect(stale).rejects.toThrow("test::run_tool needs an owner handle");
  await owner.releaseGrant(handle);
  const dead = await failed(
    runner.runTool(handle, "echo", [], options()) as Promise<Completion<never>>,
  );
  expect(dead.declaration.name).toBe("test::stale_handle");
  const live = value(await owner.admitGrant(GRANT));
  const badCwd = await owned(() =>
    runner.runTool(live, "echo", ["x"], options({ cwd: "/nonexistent-dir-xyz" })),
  );
  expect(badCwd.kind).toBe("domain");
  if (badCwd.kind !== "domain") throw Error("expected domain failure");
  expect(domainFailureDiagnostics(badCwd.value).declaration.name).toBe("files::not_found");
});

test("evidence append/seal roundtrip attests exactly the appended bytes", async () => {
  const { owner, evidence } = rig();
  const handle = value(await owner.admitGrant(GRANT));
  const ev = value(await evidence.openEvidence(handle));
  expect(Object.isFrozen(ev)).toBe(true);
  const first = new TextEncoder().encode("report-body");
  const second = new TextEncoder().encode("n-receipt-body");
  await evidence.appendEvidence(ev, "qualified", ownBytes(first));
  await evidence.appendEvidence(ev, "receipt", ownBytes(second));
  const sealed = value(await evidence.sealEvidence(ev, "qualified-report"));
  const hash = createHash("sha256");
  hash.update(frame("qualified", first));
  hash.update(frame("receipt", second));
  expect(dataProperty(sealed, "digest")).toBe(`sha256:${hash.digest("hex")}`);
  expect(dataProperty(sealed, "kind")).toBe("qualified-report");
  expect(dataProperty(sealed, "bytes")).toBe(BigInt(first.byteLength + second.byteLength));
  expect(dataProperty(sealed, "entries")).toBe(2n);
  for (const kind of RECEIPT_KINDS) {
    const other = value(await evidence.openEvidence(handle));
    await evidence.appendEvidence(other, "e", ownBytes(new Uint8Array()));
    expect(dataProperty(value(await evidence.sealEvidence(other, kind)), "kind")).toBe(kind);
  }
  const badKind = await failed(
    evidence.sealEvidence(value(await evidence.openEvidence(handle)), "bogus") as Promise<
      Completion<never>
    >,
  );
  expect(badKind.declaration.name).toBe("test::invalid_kind");
});

test("evidence names are validated and the seal consumes the bundle", async () => {
  const { owner, evidence } = rig();
  const handle = value(await owner.admitGrant(GRANT));
  const ev = value(await evidence.openEvidence(handle));
  for (const [name, reason] of [
    ["", "empty"],
    ["a\0b", "nul_byte"],
    ["a/b", "separator"],
    ["x".repeat(129), "too_long"],
  ] as const) {
    const details = await failed(
      evidence.appendEvidence(ev, name, ownBytes(new Uint8Array())) as Promise<Completion<never>>,
    );
    expect(details.declaration.name).toBe("test::invalid_name");
    expect(details.payload).toMatchObject({ reason });
  }
  await evidence.appendEvidence(ev, "dup", ownBytes(new Uint8Array()));
  const dupe = await failed(
    evidence.appendEvidence(ev, "dup", ownBytes(new Uint8Array())) as Promise<Completion<never>>,
  );
  expect(dupe.payload).toMatchObject({ reason: "duplicate" });
  await evidence.sealEvidence(ev, "n-receipt");
  for (const call of [
    evidence.appendEvidence(ev, "late", ownBytes(new Uint8Array())),
    evidence.sealEvidence(ev, "n-receipt"),
  ]) {
    const details = await failed(call as Promise<Completion<never>>);
    expect(details.declaration.name).toBe("test::closed_handle");
  }
  const ev2 = value(await evidence.openEvidence(handle));
  await owner.releaseGrant(handle);
  for (const call of [
    evidence.appendEvidence(ev2, "x", ownBytes(new Uint8Array())),
    evidence.sealEvidence(ev2, "n-receipt"),
    evidence.openEvidence(handle),
  ]) {
    const details = await failed(call as Promise<Completion<never>>);
    expect(details.declaration.name).toBe("test::stale_handle");
  }
});
