// NT-P28 S1c test workspace adapter. Owner-scoped scratch directories:
// open mints a fresh mkdtemp root behind an opaque handle; mkdir,
// write_text and read_text address handle-plus-relative-path and
// delegate to the shared files adapters; close seals idempotently and
// removes the tree. Absolute paths, NUL bytes and parent escapes
// reject before any native call, and every resolved path is verified
// to stay under its root. Delegated failures name the resolved
// absolute path, exactly as the shared adapters report it. Roots are
// registered scope-managed resources so scope drain removes abandoned
// workspaces; close completes the resource.
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve, sep } from "node:path";
import { failure, success, type AssertionContext, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
import type { FailureOrigin } from "../failure.ts";
import { closeResource, registerResource } from "../owner.ts";
import { createFileDirectory } from "../platform/files/directory.ts";
import { createFileReads } from "../platform/files/read.ts";
import { createFileWrites } from "../platform/files/write.ts";
import { ownerTag, type TestOwner } from "./owner.ts";

const workspaceBrand = Symbol("can.test.workspace");
export type WorkspaceHandle = Readonly<{ readonly [workspaceBrand]: number }>;

const MAX_WORKSPACES = 64;

const origin: FailureOrigin = Object.freeze({
  source: "can:test-workspace",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type TestWorkspaceErrors = Readonly<{
  staleHandle: string;
  closedHandle: string;
  invalidPath: string;
  notFound: string;
  alreadyExists: string;
  denied: string;
  unexpectedKind: string;
  limitExceeded: string;
  invalidData: string;
  ioError: string;
}>;

type WorkspaceCell = {
  readonly owner: unknown;
  readonly root: string;
  readonly resource: unknown;
  closed: boolean;
};

function isWorkspaceHandle(value: unknown): value is WorkspaceHandle {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as Record<symbol, unknown>)[workspaceBrand] === "number"
  );
}

function workspaceName(id: number): string {
  return `workspace#${id}`;
}

export function createTestWorkspace(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: TestWorkspaceErrors,
  owner: TestOwner,
) {
  const reads = createFileReads(domain, {
    notFound: errors.notFound,
    denied: errors.denied,
    invalidPath: errors.invalidPath,
    unexpectedKind: errors.unexpectedKind,
    limitExceeded: errors.limitExceeded,
    invalidData: errors.invalidData,
    ioError: errors.ioError,
  });
  // mkdir/writeText exercise only their documented failure subsets; the
  // remaining factory identities are real but unreachable through these
  // three methods (copy/move/list/stat are never delegated).
  const writes = createFileWrites(domain, {
    notFound: errors.notFound,
    alreadyExists: errors.alreadyExists,
    denied: errors.denied,
    invalidPath: errors.invalidPath,
    unexpectedKind: errors.unexpectedKind,
    notEmpty: errors.ioError,
    crossDevice: errors.ioError,
    ioError: errors.ioError,
  });
  const directories = createFileDirectory(domain, {
    notFound: errors.notFound,
    alreadyExists: errors.alreadyExists,
    denied: errors.denied,
    invalidPath: errors.invalidPath,
    unexpectedKind: errors.unexpectedKind,
    limitExceeded: errors.limitExceeded,
    notEmpty: errors.ioError,
    ioError: errors.ioError,
    fileInfo: errors.ioError,
    entry: errors.ioError,
  });
  const workspaces = new Map<number, WorkspaceCell>();
  let nextId = 0;

  function fail(
    identity: string,
    fields: readonly (readonly [string, unknown])[],
  ): Completion<never> {
    return failure(domain.create(identity, record(identity, fields), origin));
  }

  function resolvePath(handle: unknown, path: unknown): string | Completion<never> {
    if (!isWorkspaceHandle(handle))
      throw new TypeError("test workspace operation needs a workspace handle");
    const id = handle[workspaceBrand];
    const cell = workspaces.get(id);
    if (cell === undefined || !owner.ownerLive(cell.owner))
      return fail(errors.staleHandle, [["handle", workspaceName(id)]]);
    if (cell.closed) return fail(errors.closedHandle, [["handle", workspaceName(id)]]);
    if (typeof path !== "string" || path === "")
      return fail(errors.invalidPath, [
        ["path", typeof path === "string" ? path : "?"],
        ["reason", "empty"],
      ]);
    if (path.includes("\0"))
      return fail(errors.invalidPath, [
        ["path", path],
        ["reason", "nul_byte"],
      ]);
    if (path.startsWith("/") || /^[A-Za-z]:[\\/]/.test(path))
      return fail(errors.invalidPath, [
        ["path", path],
        ["reason", "absolute"],
      ]);
    const kept: string[] = [];
    for (const segment of path.split("/")) {
      if (segment === "" || segment === ".") continue;
      if (segment === "..")
        return fail(errors.invalidPath, [
          ["path", path],
          ["reason", "parent_escape"],
        ]);
      kept.push(segment);
    }
    const resolved = resolve(cell.root, ...kept);
    if (resolved !== cell.root && !resolved.startsWith(cell.root + sep))
      return fail(errors.invalidPath, [
        ["path", path],
        ["reason", "escape"],
      ]);
    return resolved;
  }

  return Object.freeze({
    async openWorkspace(
      ownerHandle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<WorkspaceHandle>> {
      const tag = ownerTag(ownerHandle);
      if (tag === null) throw new TypeError("test::workspace_open needs an owner handle");
      if (!owner.ownerLive(ownerHandle)) return fail(errors.staleHandle, [["handle", tag]]);
      if (workspaces.size >= MAX_WORKSPACES)
        throw new TypeError("test workspace table cap reached");
      const root = mkdtempSync(join(tmpdir(), "can-test-ws-"));
      const resource = registerResource(
        "test-workspace",
        { root },
        async () => {
          rmSync(root, { recursive: true, force: true });
          return success(undefined);
        },
        { scopeManaged: true, idempotent: true },
      );
      nextId += 1;
      workspaces.set(nextId, { owner: ownerHandle, root, resource, closed: false });
      const handle: WorkspaceHandle = Object.freeze({ [workspaceBrand]: nextId });
      return success(handle);
    },

    async workspaceMkdir(
      handle: unknown,
      path: unknown,
      recursive: unknown,
      context?: AssertionContext,
    ): Promise<Completion<undefined>> {
      const resolved = resolvePath(handle, path);
      if (typeof resolved !== "string") return resolved;
      return directories.mkdir(resolved, recursive as boolean, context);
    },

    async workspaceWriteText(
      handle: unknown,
      path: unknown,
      value: unknown,
      overwrite: unknown,
      context?: AssertionContext,
    ): Promise<Completion<undefined>> {
      const resolved = resolvePath(handle, path);
      if (typeof resolved !== "string") return resolved;
      return writes.writeText(resolved, value as string, overwrite as boolean, context);
    },

    async workspaceReadText(
      handle: unknown,
      path: unknown,
      maxBytes: unknown,
      context?: AssertionContext,
    ): Promise<Completion<string>> {
      const resolved = resolvePath(handle, path);
      if (typeof resolved !== "string") return resolved;
      return reads.readText(resolved, maxBytes as bigint, context);
    },

    async closeWorkspace(handle: unknown): Promise<Completion<void>> {
      if (!isWorkspaceHandle(handle))
        throw new TypeError("test::workspace_close needs a workspace handle");
      const id = handle[workspaceBrand];
      const cell = workspaces.get(id);
      if (cell === undefined) throw new TypeError(`foreign workspace handle ${workspaceName(id)}`);
      cell.closed = true;
      try {
        await closeResource(cell.resource, "test-workspace");
      } catch {}
      return success(undefined);
    },
  });
}

export type TestWorkspace = ReturnType<typeof createTestWorkspace>;
