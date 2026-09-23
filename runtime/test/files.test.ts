import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { mkdtemp, rm, mkdir, writeFile, symlink, chmod } from "node:fs/promises";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createFileReads } from "../platform/files/read.ts";
import { createFileWrites } from "../platform/files/write.ts";
import { createFileDirectory } from "../platform/files/directory.ts";
import { createFileGlobs } from "../platform/files/glob.ts";
import { createPath } from "../platform/files/path.ts";
import { classifyFileError, validatePath } from "../platform/files/errors.ts";
import { copyBytes, ownBytes } from "../bytes.ts";
import { recordIdentity, dataProperty } from "../data.ts";
import { success, invoke, type Completion } from "../completion.ts";
import { runAssertion } from "../assert/runner.ts";
const hash = (kind: string, name: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, name]))
    .digest("hex");
const shape = (
  kind: string,
  declaration: string,
  fields: { name: string; type: string }[] = [],
): FailureShape => ({
  identity: hash(kind, declaration),
  kind,
  declaration,
  fields,
  arguments: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const str = shape("primitive", "str"),
  int = shape("primitive", "int");
const declarations = catalogue.errors.filter((e) =>
  [1110, 1300, 1301, 1302, 1303, 1304, 1305, 1306, 1307, 1308].includes(e.id),
);
const errors = declarations.map((e) =>
  shape(
    "error",
    e.identity,
    e.fields.map((f) => ({ name: f.name, type: f.type === "int" ? int.identity : str.identity })),
  ),
);
const domain = createDomainRuntime({
  declarations: declarations.map((e) => ({ ...e, parameters: 0 })),
  shapes: [str, int, ...errors],
});
const identity = (name: string) => errors.find((e) => e.declaration === name)!.identity;
const fileInfo = hash("record", "can.std.files@1::file_info"),
  entry = hash("record", "can.std.files@1::entry");
const reads = createFileReads(domain, {
  notFound: identity("can.std.files@1::not_found"),
  denied: identity("can.std.files@1::denied"),
  invalidPath: identity("can.std.files@1::invalid_path"),
  unexpectedKind: identity("can.std.files@1::unexpected_kind"),
  limitExceeded: identity("can.std.files@1::limit_exceeded"),
  invalidData: identity("can.std.codec@1::invalid_data"),
  ioError: identity("can.std.files@1::io_error"),
});
const writes = createFileWrites(domain, {
  notFound: identity("can.std.files@1::not_found"),
  alreadyExists: identity("can.std.files@1::already_exists"),
  denied: identity("can.std.files@1::denied"),
  invalidPath: identity("can.std.files@1::invalid_path"),
  unexpectedKind: identity("can.std.files@1::unexpected_kind"),
  notEmpty: identity("can.std.files@1::not_empty"),
  crossDevice: identity("can.std.files@1::cross_device"),
  ioError: identity("can.std.files@1::io_error"),
});
const dirs = createFileDirectory(domain, {
  notFound: identity("can.std.files@1::not_found"),
  alreadyExists: identity("can.std.files@1::already_exists"),
  denied: identity("can.std.files@1::denied"),
  invalidPath: identity("can.std.files@1::invalid_path"),
  unexpectedKind: identity("can.std.files@1::unexpected_kind"),
  limitExceeded: identity("can.std.files@1::limit_exceeded"),
  notEmpty: identity("can.std.files@1::not_empty"),
  ioError: identity("can.std.files@1::io_error"),
  fileInfo,
  entry,
});
const globs = createFileGlobs(domain, {
  notFound: identity("can.std.files@1::not_found"),
  denied: identity("can.std.files@1::denied"),
  invalidPath: identity("can.std.files@1::invalid_path"),
  limitExceeded: identity("can.std.files@1::limit_exceeded"),
  ioError: identity("can.std.files@1::io_error"),
});
const paths = createPath();
const origin = { source: "test", start: 0, end: 0, invocation: [] };
function check(result: Completion, id: number, payload: object) {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain");
  const d = domainFailureDiagnostics(result.value);
  expect(d.declaration.id).toBe(id);
  expect(d.payload).toMatchObject(payload);
}
async function withTemp(run: (root: string) => Promise<void>): Promise<void> {
  const root = await mkdtemp(join(tmpdir(), "can-files-test-"));
  try {
    await run(root);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
}
test("binary round-trip preserves NUL and invalid UTF-8; text round-trips strictly", async () => {
  await withTemp(async (root) => {
    const file = join(root, "a.bin");
    expect(await writes.writeBytes(file, ownBytes(new Uint8Array([0, 255, 1])), true)).toEqual(
      success(undefined),
    );
    const back = await reads.readBytes(file, 100n);
    expect(back.kind).toBe("ok");
    if (back.kind !== "ok") throw Error();
    expect(Array.from(copyBytes(back.value, origin))).toEqual([0, 255, 1]);
    check(await reads.readText(file, 100n), 1110, { path: file, reason: "utf8" });
    const text = join(root, "b.txt");
    expect(await writes.writeText(text, "﻿hé😀", true)).toEqual(success(undefined));
    expect(await reads.readText(text, 100n)).toEqual(success("﻿hé😀"));
  });
});
test("missing paths report not_found; denied paths never report absence", async () => {
  await withTemp(async (root) => {
    const missing = join(root, "nope.txt");
    check(await reads.readBytes(missing, 10n), 1300, { path: missing });
    check(await reads.readText(missing, 10n), 1300, { path: missing });
    check(await dirs.stat(missing, true), 1300, { path: missing });
    expect(await dirs.exists(missing)).toEqual(success(false));
    check(await dirs.list(missing, 10n), 1300, { path: missing });
    check(await globs.glob(missing, "*", false, 10n), 1300, { path: missing });
    check(await dirs.remove(missing, false), 1300, { path: missing });
    const locked = join(root, "locked");
    await mkdir(locked);
    await writeFile(join(locked, "secret.txt"), "x");
    await chmod(locked, 0o000);
    try {
      check(await reads.readBytes(join(locked, "secret.txt"), 10n), 1301, {
        operation: "read_bytes",
      });
      check(await dirs.list(locked, 10n), 1301, { operation: "list" });
      check(await dirs.exists(join(locked, "secret.txt")), 1301, { operation: "exists" });
    } finally {
      await chmod(locked, 0o700);
    }
  });
});
test("negative limits and empty/NUL paths fail before touching disk", async () => {
  await withTemp(async (root) => {
    const ghost = join(root, "ghost.txt");
    check(await reads.readBytes(ghost, -1n), 1305, { limit: -1n });
    check(await dirs.list(root, -1n), 1305, { limit: -1n });
    check(await globs.glob(root, "*", false, -1n), 1305, { limit: -1n });
    expect(await Bun.file(ghost).exists()).toBe(false);
    for (const bad of ["", root + "\0/x"]) {
      const reason = bad === "" ? "empty" : "nul_byte";
      check(await reads.readBytes(bad, 1n), 1303, { path: bad, reason });
      check(await writes.writeText(bad, "x", true), 1303, { path: bad, reason });
      check(await dirs.stat(bad, true), 1303, { path: bad, reason });
      check(await dirs.exists(bad), 1303, { path: bad, reason });
      check(await dirs.list(bad, 1n), 1303, { path: bad, reason });
      check(await dirs.mkdir(bad, false), 1303, { path: bad, reason });
      check(await dirs.remove(bad, false), 1303, { path: bad, reason });
      check(await globs.glob(bad, "*", false, 1n), 1303, { path: bad, reason });
      check(await globs.glob(root, bad, false, 1n), 1303, { path: bad, reason });
    }
  });
});
test("over-cap input fails; exact caps and huge limits succeed", async () => {
  await withTemp(async (root) => {
    const file = join(root, "big.bin");
    await writeFile(file, new Uint8Array(1000));
    check(await reads.readBytes(file, 999n), 1305, { limit: 999n });
    const exact = await reads.readBytes(file, 1000n);
    expect(exact.kind).toBe("ok");
    expect(await reads.readBytes(file, 10n ** 100n)).toEqual(exact);
    check(await reads.readBytes(file, 0n), 1305, { limit: 0n });
    const empty = join(root, "empty.txt");
    await writeFile(empty, "");
    expect(await reads.readText(empty, 0n)).toEqual(success(""));
  });
});
test("exclusive creation is atomic; concurrent creators race to one winner", async () => {
  await withTemp(async (root) => {
    const file = join(root, "x.txt");
    expect(await writes.writeText(file, "first", false)).toEqual(success(undefined));
    check(await writes.writeText(file, "second", false), 1302, { path: file });
    expect(await writes.writeText(file, "second", true)).toEqual(success(undefined));
    expect(await reads.readText(file, 100n)).toEqual(success("second"));
    const racy = join(root, "race.txt");
    const outcomes = await Promise.all(
      Array.from({ length: 10 }, (_, i) => writes.writeText(racy, "writer" + i, false)),
    );
    expect(outcomes.filter((o) => o.kind === "ok")).toHaveLength(1);
    for (const o of outcomes.filter((o) => o.kind !== "ok")) check(o, 1302, { path: racy });
    const dup = join(root, "dup.txt");
    expect(await writes.copy(file, dup, false)).toEqual(success(undefined));
    check(await writes.copy(file, dup, false), 1302, { path: dup });
  });
});
test("missing parents fail; writes never create directories", async () => {
  await withTemp(async (root) => {
    check(await writes.writeText(join(root, "nodir", "f.txt"), "x", true), 1300, {
      path: join(root, "nodir", "f.txt"),
    });
    expect(await Bun.file(join(root, "nodir")).exists()).toBe(false);
    check(await dirs.mkdir(join(root, "nodir", "sub"), false), 1300, {});
    check(await dirs.mkdir(root, false), 1302, { path: root });
    expect(await dirs.mkdir(root, true)).toEqual(success(undefined));
    expect(await dirs.mkdir(join(root, "a", "b"), true)).toEqual(success(undefined));
    const src = join(root, "src.txt");
    await writeFile(src, "data");
    check(await writes.copy(join(root, "missing.txt"), join(root, "dst.txt"), true), 1300, {
      path: join(root, "missing.txt"),
    });
    check(await writes.copy(src, join(root, "nodir2", "dst.txt"), true), 1300, {
      path: join(root, "nodir2", "dst.txt"),
    });
    check(await writes.move(join(root, "missing.txt"), join(root, "dst.txt"), true), 1300, {
      path: join(root, "missing.txt"),
    });
    check(await writes.move(src, join(root, "nodir2", "dst.txt"), true), 1300, {
      path: join(root, "nodir2", "dst.txt"),
    });
  });
});
test("directory-as-file and file-as-directory are kind errors", async () => {
  await withTemp(async (root) => {
    await writeFile(join(root, "f.txt"), "x");
    await mkdir(join(root, "sub"));
    check(await reads.readBytes(join(root, "sub"), 10n), 1308, {
      path: join(root, "sub"),
      operation: "read_bytes",
    });
    check(await reads.readText(join(root, "sub"), 10n), 1308, { operation: "read_text" });
    check(await dirs.list(join(root, "f.txt"), 10n), 1308, {
      path: join(root, "f.txt"),
      operation: "list",
    });
    check(await writes.copy(join(root, "sub"), join(root, "c.txt"), true), 1308, {
      operation: "copy",
    });
  });
});
test("stat follows or reports links; exists sees dangling links", async () => {
  await withTemp(async (root) => {
    await writeFile(join(root, "a.txt"), "hello");
    await symlink("a.txt", join(root, "link.txt"));
    await symlink("gone", join(root, "dangling"));
    await mkdir(join(root, "sub"));
    const file = await dirs.stat(join(root, "a.txt"), true);
    expect(file.kind).toBe("ok");
    if (file.kind !== "ok") throw Error();
    expect(recordIdentity(file.value)).toBe(fileInfo);
    expect(dataProperty(file.value, "kind")).toBe("file");
    expect(dataProperty(file.value, "size")).toBe(5n);
    const link = await dirs.stat(join(root, "link.txt"), false);
    expect(link.kind).toBe("ok");
    if (link.kind !== "ok") throw Error();
    expect(dataProperty(link.value, "kind")).toBe("symlink");
    const followed = await dirs.stat(join(root, "link.txt"), true);
    expect(followed.kind).toBe("ok");
    if (followed.kind !== "ok") throw Error();
    expect(dataProperty(followed.value, "kind")).toBe("file");
    const dangling = await dirs.stat(join(root, "dangling"), false);
    expect(dangling.kind).toBe("ok");
    if (dangling.kind !== "ok") throw Error();
    expect(dataProperty(dangling.value, "kind")).toBe("symlink");
    check(await dirs.stat(join(root, "dangling"), true), 1300, { path: join(root, "dangling") });
    expect(await dirs.exists(join(root, "dangling"))).toEqual(success(true));
    expect(await dirs.exists(join(root, "sub"))).toEqual(success(true));
  });
});
test("list returns absolute sorted entries with kinds", async () => {
  await withTemp(async (root) => {
    await writeFile(join(root, "b.txt"), "b");
    await writeFile(join(root, "a.txt"), "a");
    await mkdir(join(root, "sub"));
    await symlink("a.txt", join(root, "link.txt"));
    const result = await dirs.list(root, 100n);
    expect(result.kind).toBe("ok");
    if (result.kind !== "ok") throw Error();
    const rows = result.value as readonly unknown[];
    expect(rows.map((v) => dataProperty(v, "path"))).toEqual([
      join(root, "a.txt"),
      join(root, "b.txt"),
      join(root, "link.txt"),
      join(root, "sub"),
    ]);
    expect(rows.map((v) => dataProperty(v, "kind"))).toEqual([
      "file",
      "file",
      "symlink",
      "directory",
    ]);
    for (const v of rows) expect(recordIdentity(v)).toBe(entry);
    check(await dirs.list(root, 2n), 1305, { limit: 2n });
  });
});
test("copy and move round-trip; overwrite gates replacement", async () => {
  await withTemp(async (root) => {
    const src = join(root, "src.txt"),
      dst = join(root, "dst.txt");
    await writeFile(src, "payload");
    expect(await writes.copy(src, dst, false)).toEqual(success(undefined));
    expect(await reads.readText(dst, 100n)).toEqual(success("payload"));
    expect(await reads.readText(src, 100n)).toEqual(success("payload"));
    check(await writes.copy(src, dst, false), 1302, { path: dst });
    const moved = join(root, "moved.txt");
    expect(await writes.move(src, moved, true)).toEqual(success(undefined));
    expect(await dirs.exists(src)).toEqual(success(false));
    expect(await reads.readText(moved, 100n)).toEqual(success("payload"));
    check(await writes.move(dst, moved, false), 1302, { path: moved });
    expect(await writes.move(dst, moved, true)).toEqual(success(undefined));
    expect(await reads.readText(moved, 100n)).toEqual(success("payload"));
  });
});
test("remove deletes entries; trees need explicit recursion", async () => {
  await withTemp(async (root) => {
    const tree = join(root, "tree");
    await mkdir(join(tree, "sub"), { recursive: true });
    await writeFile(join(tree, "sub", "f.txt"), "x");
    await writeFile(join(tree, "top.txt"), "t");
    await symlink("top.txt", join(tree, "link.txt"));
    check(await dirs.remove(tree, false), 1306, { path: tree });
    expect(await dirs.exists(tree)).toEqual(success(true));
    expect(await dirs.remove(join(tree, "link.txt"), false)).toEqual(success(undefined));
    expect(await dirs.exists(join(tree, "top.txt"))).toEqual(success(true));
    expect(await dirs.remove(join(tree, "sub", "f.txt"), false)).toEqual(success(undefined));
    expect(await dirs.remove(join(tree, "sub"), false)).toEqual(success(undefined));
    expect(await dirs.remove(tree, true)).toEqual(success(undefined));
    expect(await dirs.exists(tree)).toEqual(success(false));
    expect(await dirs.exists(root)).toEqual(success(true));
  });
});
test("glob matches dotfiles and symlinks with pinned traversal", async () => {
  await withTemp(async (root) => {
    await mkdir(join(root, "sub"));
    await writeFile(join(root, "sub", "a.txt"), "x");
    await writeFile(join(root, "top.txt"), "y");
    await writeFile(join(root, ".dot"), "d");
    await symlink("top.txt", join(root, "link.txt"));
    await symlink("sub", join(root, "linkdir"));
    await symlink("gone", join(root, "dangling.txt"));
    expect(await globs.glob(root, "*.txt", false, 100n)).toEqual(
      success([join(root, "dangling.txt"), join(root, "link.txt"), join(root, "top.txt")]),
    );
    expect(await globs.glob(root, "*.txt", true, 100n)).toEqual(
      success([join(root, "dangling.txt"), join(root, "link.txt"), join(root, "top.txt")]),
    );
    expect(await globs.glob(root, "**/*.txt", false, 100n)).toEqual(
      success([
        join(root, "dangling.txt"),
        join(root, "link.txt"),
        join(root, "sub", "a.txt"),
        join(root, "top.txt"),
      ]),
    );
    expect(await globs.glob(root, "**/*.txt", true, 100n)).toEqual(
      success([
        join(root, "dangling.txt"),
        join(root, "link.txt"),
        join(root, "linkdir", "a.txt"),
        join(root, "sub", "a.txt"),
        join(root, "top.txt"),
      ]),
    );
    expect(await globs.glob(root, "*", false, 100n)).toEqual(
      success([
        join(root, "dangling.txt"),
        join(root, "link.txt"),
        join(root, "linkdir"),
        join(root, "sub"),
        join(root, "top.txt"),
      ]),
    );
    expect(await globs.glob(root, ".*", false, 100n)).toEqual(success([join(root, ".dot")]));
    check(await globs.glob(root, "**/*", false, 2n), 1305, { limit: 2n });
  });
});
test("path operations compute natively", async () => {
  expect(await paths.resolvePath("/a/b", ["c", "d"])).toEqual(success("/a/b/c/d"));
  expect(await paths.resolvePath("/a/b", ["..", "c"])).toEqual(success("/a/c"));
  expect(await paths.joinPath(["a", "b", "c"])).toEqual(success("a/b/c"));
  expect(await paths.basename("/a/b.txt")).toEqual(success("b.txt"));
  expect(await paths.basename("/a/")).toEqual(success("a"));
  expect(await paths.extension("a.tar.gz")).toEqual(success(".gz"));
  expect(await paths.extension("noext")).toEqual(success(""));
});
test("unsupplied assertions cannot touch the filesystem", async () => {
  await withTemp(async (root) => {
    const ghost = join(root, "ghost.txt");
    const calls: [string, (context: any) => Promise<Completion>][] = [
      ["read", (context) => reads.readBytes(join(root, "a.txt"), 1n, context)],
      ["write", (context) => writes.writeText(ghost, "x", true, context)],
      ["stat", (context) => dirs.stat(root, true, context)],
      ["exists", (context) => dirs.exists(root, context)],
      ["list", (context) => dirs.list(root, 1n, context)],
      ["mkdir", (context) => dirs.mkdir(join(root, "d"), false, context)],
      ["copy", (context) => writes.copy(join(root, "a"), join(root, "b"), true, context)],
      ["move", (context) => writes.move(join(root, "a"), join(root, "b"), true, context)],
      ["remove", (context) => dirs.remove(root, false, context)],
      ["glob", (context) => globs.glob(root, "*", false, 1n, context)],
    ];
    for (const [name, actual] of calls) {
      const result = await runAssertion({
        root: { package: "test", declaration: "boundary", name },
        actual,
        expected: async () => success(undefined),
      });
      expect(result.passed).toBe(false);
    }
    expect(await Bun.file(ghost).exists()).toBe(false);
    expect(await Bun.file(join(root, "d")).exists()).toBe(false);
    expect(await paths.resolvePath("/a", ["b"], undefined)).toEqual(success("/a/b"));
  });
});
test("unknown native failures stay standard; classification is exact", async () => {
  expect(classifyFileError(new TypeError("defect"))).toBeUndefined();
  expect(classifyFileError(null)).toBeUndefined();
  expect(classifyFileError("ENOENT")).toBeUndefined();
  expect(classifyFileError(new Proxy(new Error("x"), {}))).toBeUndefined();
  expect(classifyFileError(Object.assign(new Error("denied"), { code: "EACCES" }))).toEqual({
    kind: "denied",
    code: "EACCES",
  });
  expect(classifyFileError(Object.assign(new Error("busy"), { code: "EBUSY" }))).toEqual({
    kind: "io_error",
    code: "EBUSY",
  });
  expect(classifyFileError(Object.assign(new Error("device"), { code: "EXDEV" }))).toEqual({
    kind: "cross_device",
    code: "EXDEV",
  });
  expect(classifyFileError(Object.assign(new Error("bogus"), { code: "BOGUS" }))).toBeUndefined();
  expect(validatePath("")).toBe("empty");
  expect(validatePath("a\0b")).toBe("nul_byte");
  expect(validatePath("ok")).toBeUndefined();
  const result = await invoke(() => reads.readBytes(123 as unknown as string, 1n), origin);
  expect(result.kind).toBe("standard");
});
