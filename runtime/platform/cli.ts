// I11's first executable CLI slice. The remaining bounded input/environment
// operations and assertion boundary queues are implemented by I13/I29/I18.
import { denyLiveBoundary, type AssertionContext } from "../assert/context.ts";
import { types as nativeTypes } from "node:util";
import { success, failure, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { copyBytes } from "../bytes.ts";
import { createDomainRuntime } from "../domain.ts";
import { type FailureOrigin } from "../failure.ts";

type Domain = ReturnType<typeof createDomainRuntime>;
type Contracts = Readonly<{writeFailed: string}>;
const origin: FailureOrigin = Object.freeze({source: "can:cli", start: 0, end: 0, invocation: Object.freeze([])});
const writeCodes = new Set(["EPIPE", "EBADF", "EIO", "ENOSPC", "EACCES", "EINVAL", "EFBIG", "EROFS", "EINTR", "EAGAIN"]);
function expectedWriteFailure(cause: unknown): boolean {
  if (cause === null || typeof cause !== "object" || nativeTypes.isProxy(cause) || !nativeTypes.isNativeError(cause)) return false;
  const code = Object.getOwnPropertyDescriptor(cause, "code");
  return code !== undefined && "value" in code && typeof code.value === "string" && writeCodes.has(code.value);
}
export function createCLI(domain: Domain, contracts: Contracts) {
  return Object.freeze({
    async stdoutWrite(bytes: unknown, context?: AssertionContext): Promise<Completion<bigint>> { denyLiveBoundary(context, origin); return write(bytes, "stdout_write", Bun.stdout); },
    async stderrWrite(bytes: unknown, context?: AssertionContext): Promise<Completion<bigint>> { denyLiveBoundary(context, origin); return write(bytes, "stderr_write", Bun.stderr); },
  });
  async function write(bytes: unknown, operation: string, destination: typeof Bun.stdout): Promise<Completion<bigint>> {
    const data = copyBytes(bytes, origin);
    try { return success(BigInt(await Bun.write(destination, data))); }
    catch (cause) {
      if (!expectedWriteFailure(cause)) throw cause;
      return failure(domain.create(contracts.writeFailed, record(contracts.writeFailed, [["operation", operation]]), origin, cause));
    }
  }
}
