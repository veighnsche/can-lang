// Browser-profile log writer. Emitted lines keep their exact JSON bytes;
// the sink is the console instead of stderr, routed by level, with the
// same live-boundary guard and failure taxonomy as the canonical module.
import { denyLiveBoundary, type AssertionContext } from "../assert/context.ts";
import { success, failure, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import type { DomainRuntime } from "../domain-core.ts";

const origin = Object.freeze({
  source: "can:log",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

type Sink = Readonly<{
  encode: (value: Readonly<{ level: string; message: string }>) => string;
  info: (line: string) => void;
  error: (line: string) => void;
}>;

const sink: Sink = Object.freeze({
  encode: JSON.stringify,
  info: (line) => console.log(line),
  error: (line) => console.error(line),
});

export function createLog(domain: DomainRuntime, writeFailed: string, host: Sink = sink) {
  async function write(
    level: "info" | "error",
    message: string,
    context?: AssertionContext,
  ): Promise<Completion<undefined>> {
    denyLiveBoundary(context, origin);
    if (typeof message !== "string") throw new TypeError("invalid log text");
    const failed = (cause?: unknown) =>
      failure(domain.create(writeFailed, record(writeFailed, [["level", level]]), origin, cause));
    let line: string;
    try {
      line = host.encode({ level, message });
    } catch (cause) {
      if (!(cause instanceof TypeError) && !(cause instanceof RangeError)) throw cause;
      return failed(cause);
    }
    try {
      if (level === "error") host.error(line);
      else host.info(line);
      return success(undefined);
    } catch (cause) {
      return failed(cause);
    }
  }
  return Object.freeze({
    async writeInfo(message: string, context?: AssertionContext) {
      return write("info", message, context);
    },
    async writeError(message: string, context?: AssertionContext) {
      return write("error", message, context);
    },
  });
}
