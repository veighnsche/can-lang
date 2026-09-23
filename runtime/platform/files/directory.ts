import { stat, lstat, readdir, mkdir, rm, rmdir, unlink } from "node:fs/promises";
import { resolve, join } from "node:path";
import { denyLiveBoundary, type AssertionContext } from "../../assert/context.ts";
import { success, failure, type Completion } from "../../completion.ts";
import { array, record } from "../../data.ts";
import { createDomainRuntime } from "../../domain.ts";
import type { FailureOrigin } from "../../failure.ts";
import { classifyFileError, validatePath, kindOfEntry } from "./errors.ts";
const origin: FailureOrigin = Object.freeze({
  source: "can:files-directory",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
type Contracts = Readonly<{
  notFound: string;
  alreadyExists: string;
  denied: string;
  invalidPath: string;
  unexpectedKind: string;
  limitExceeded: string;
  notEmpty: string;
  ioError: string;
  fileInfo: string;
  entry: string;
}>;
export function createFileDirectory(
  domain: ReturnType<typeof createDomainRuntime>,
  types: Contracts,
) {
  const fail = (
    identity: string,
    fields: readonly (readonly [string, unknown])[],
    cause?: unknown,
  ) => failure(domain.create(identity, record(identity, fields), origin, cause));
  function failureFor(cause: unknown, path: string, operation: string): Completion<never> {
    const classified = classifyFileError(cause);
    if (classified === undefined) throw cause;
    switch (classified.kind) {
      case "not_found":
        return fail(types.notFound, [["path", path]], cause);
      case "already_exists":
        return fail(types.alreadyExists, [["path", path]], cause);
      case "denied":
        return fail(
          types.denied,
          [
            ["path", path],
            ["operation", operation],
          ],
          cause,
        );
      case "unexpected_kind":
        return fail(
          types.unexpectedKind,
          [
            ["path", path],
            ["operation", operation],
          ],
          cause,
        );
      case "not_empty":
        return fail(types.notEmpty, [["path", path]], cause);
      case "not_directory":
        return fail(
          types.invalidPath,
          [
            ["path", path],
            ["reason", "not_directory"],
          ],
          cause,
        );
      case "name_too_long":
        return fail(
          types.invalidPath,
          [
            ["path", path],
            ["reason", "name_too_long"],
          ],
          cause,
        );
      default:
        return fail(
          types.ioError,
          [
            ["path", path],
            ["operation", operation],
          ],
          cause,
        );
    }
  }
  function invalid(path: string): Completion<never> | undefined {
    const reason = validatePath(path);
    if (reason === undefined) return undefined;
    return fail(types.invalidPath, [
      ["path", path],
      ["reason", reason],
    ]);
  }
  return Object.freeze({
    async stat(
      path: string,
      followSymlinks: boolean,
      context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      const bad = invalid(path);
      if (bad !== undefined) return bad;
      try {
        const info = await (followSymlinks ? stat(path) : lstat(path));
        return success(
          record(types.fileInfo, [
            ["kind", kindOfEntry(info)],
            ["size", BigInt(info.size)],
          ]),
        );
      } catch (cause) {
        return failureFor(cause, path, "stat");
      }
    },
    async exists(path: string, context?: AssertionContext): Promise<Completion<boolean>> {
      denyLiveBoundary(context, origin);
      const bad = invalid(path);
      if (bad !== undefined) return bad;
      try {
        await lstat(path);
        return success(true);
      } catch (cause) {
        if (classifyFileError(cause)?.kind === "not_found") return success(false);
        return failureFor(cause, path, "exists");
      }
    },
    async list(
      path: string,
      maxEntries: bigint,
      context?: AssertionContext,
    ): Promise<Completion<readonly unknown[]>> {
      denyLiveBoundary(context, origin);
      const bad = invalid(path);
      if (bad !== undefined) return bad;
      if (maxEntries < 0n) return fail(types.limitExceeded, [["limit", maxEntries]]);
      try {
        const entries = await readdir(path, { withFileTypes: true });
        if (BigInt(entries.length) > maxEntries)
          return fail(types.limitExceeded, [["limit", maxEntries]]);
        const base = resolve(path);
        const out = entries.map((entry) =>
          record(types.entry, [
            ["path", join(base, entry.name)],
            ["kind", kindOfEntry(entry)],
          ]),
        );
        out.sort((a, b) =>
          String((a as Record<string, unknown>).path) < String((b as Record<string, unknown>).path)
            ? -1
            : 1,
        );
        return success(array(out));
      } catch (cause) {
        // A file listed as a directory is a kind error, not a generic failure.
        if (classifyFileError(cause)?.kind === "not_directory")
          return fail(
            types.unexpectedKind,
            [
              ["path", path],
              ["operation", "list"],
            ],
            cause,
          );
        return failureFor(cause, path, "list");
      }
    },
    async mkdir(
      path: string,
      recursive: boolean,
      context?: AssertionContext,
    ): Promise<Completion<undefined>> {
      denyLiveBoundary(context, origin);
      const bad = invalid(path);
      if (bad !== undefined) return bad;
      try {
        await mkdir(path, { recursive });
        return success(undefined);
      } catch (cause) {
        return failureFor(cause, path, "mkdir");
      }
    },
    async remove(
      path: string,
      recursive: boolean,
      context?: AssertionContext,
    ): Promise<Completion<undefined>> {
      denyLiveBoundary(context, origin);
      const bad = invalid(path);
      if (bad !== undefined) return bad;
      try {
        if (recursive) {
          await rm(path, { recursive: true, force: false });
          return success(undefined);
        }
        // One entry only: directories remove when empty, anything else unlinks.
        try {
          await rmdir(path);
          return success(undefined);
        } catch (cause) {
          if (classifyFileError(cause)?.kind !== "not_directory") throw cause;
          await unlink(path);
          return success(undefined);
        }
      } catch (cause) {
        return failureFor(cause, path, "remove");
      }
    },
  });
}
