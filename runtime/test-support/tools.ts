// NT-P28 S1c test tools adapter. Keyed process dispatch: test::run_tool
// names a tool through an owner-registered key, never a path; unknown
// keys reject before any spawn. Resolved tools run through the shared
// process support, so the spawn discipline (no shell, detached group,
// stream caps, deadline, grace, reaping, scope-drain kill) and the
// cwd/arg/env failure vocabulary are exactly process::run's. The tool
// table is instance-level for this slice: the grant gates liveness
// while the N-link slice scopes registration per grant.
import { failure, type AssertionContext, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
import type { FailureOrigin } from "../failure.ts";
import { createProcesses } from "../platform/process/spawn.ts";
import { ownerTag, type TestOwner } from "./owner.ts";

const origin: FailureOrigin = Object.freeze({
  source: "can:test-tools",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type ToolRecord = Readonly<{ readonly executable: string }>;
export type ToolTable = ReadonlyMap<string, ToolRecord>;

export type TestToolsErrors = Readonly<{
  staleHandle: string;
  unknownTool: string;
  notFound: string;
  denied: string;
  spawnFailed: string;
  timeout: string;
  outputLimit: string;
  invalidConfig: string;
  processIoError: string;
  toolResult: string;
}>;

export function createTestTools(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: TestToolsErrors,
  owner: TestOwner,
  tools: ToolTable,
) {
  // requireSuccess/which are never called through this surface; nonzero
  // still needs a real identity for the shared factory.
  const processes = createProcesses(domain, {
    notFound: errors.notFound,
    denied: errors.denied,
    spawnFailed: errors.spawnFailed,
    timeout: errors.timeout,
    outputLimit: errors.outputLimit,
    nonzero: errors.spawnFailed,
    invalidConfig: errors.invalidConfig,
    ioError: errors.processIoError,
    result: errors.toolResult,
  });

  return Object.freeze({
    async runTool(
      ownerHandle: unknown,
      key: unknown,
      args: unknown,
      options: unknown,
      context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const tag = ownerTag(ownerHandle);
      if (tag === null) throw new TypeError("test::run_tool needs an owner handle");
      if (!owner.ownerLive(ownerHandle))
        return failure(
          domain.create(errors.staleHandle, record(errors.staleHandle, [["handle", tag]]), origin),
        );
      const entry = typeof key === "string" ? tools.get(key) : undefined;
      if (entry === undefined)
        return failure(
          domain.create(
            errors.unknownTool,
            record(errors.unknownTool, [["key", typeof key === "string" ? key : "?"]]),
            origin,
          ),
        );
      return processes.run(entry.executable, args as readonly unknown[], options, context);
    },
  });
}

export type TestTools = ReturnType<typeof createTestTools>;
