import { checkedCompletion, type Completion } from "../completion.ts";
import { Deadline, transportFault } from "./deadline.ts";
import { readBody } from "./body.ts";
import { nativeOperation, nativeCompletion, cleanupOperation } from "./owned.ts";
import { prepareRequest, headerSnapshot, type Connection, type Entries } from "./request.ts";
import { scopeRaceInput } from "./request-scope.ts";
import type { RequestBudget } from "./request-budget.ts";
import type { HTTPExchange } from "../assert/provider.ts";

export type NativeRequest = Readonly<{
  path: string;
  method: "GET" | "HEAD" | "POST" | "PUT" | "PATCH" | "DELETE" | "OPTIONS";
  query: Entries;
  headers: Entries;
  body?: Uint8Array;
  bodyEncoding?: "json" | "text" | "bytes";
  envelope?: boolean;
  ownerSignal?: AbortSignal;
  exchange?: HTTPExchange;
  // Shared request budget: the effective bound is min(connection
  // timeout, remaining budget). Absent, the ambient request scope
  // budget applies when a scope is active.
  budget?: RequestBudget;
}>;
export type ResponseMetadata = Readonly<{
  status: number;
  headers: readonly Readonly<{ name: string; value: string }>[];
}>;

// cappedTimeout validates the connection timeout exactly like the
// Deadline, then caps it by the remaining shared budget (explicit wins
// over ambient). The budget clock is fractional, so the cap floors to
// a whole millisecond the Deadline and the timeout_ms report accept.
// Zero means "report timeout without starting", never a zero-length
// native wait.
function cappedTimeout(connectionMs: number, budget: RequestBudget | undefined): number {
  if (!Number.isSafeInteger(connectionMs) || connectionMs < 1 || connectionMs > 2147483647)
    throw new TypeError("invalid transport deadline");
  if (budget === undefined) return connectionMs;
  return Math.max(0, Math.floor(Math.min(connectionMs, budget.remainingMilliseconds())));
}

// Private boundary: decoding returns only a checked immutable Can value. The
// native Response, headers and delivered byte buffer never escape to user code.
//
// Fetch takes the cancel-PRESENT branch: the native abort genuinely
// stops the wire request, so scope expiry (disconnect/shutdown) aborts
// in flight and reports timeout — no owned remainder, no escalation.
// A timeout never claims server rollback; reread to reconcile.
export async function performRequest<T>(
  connection: Connection,
  request: NativeRequest,
  readEnvironment: (name: string) => string | undefined,
  decode: (bytes: Uint8Array, metadata: ResponseMetadata) => Completion<T> | Promise<Completion<T>>,
): Promise<Completion<T>> {
  if (request.body !== undefined && request.body.byteLength > connection.maxBodyBytes)
    throw transportFault({ kind: "limit", limit: connection.maxBodyBytes });
  const body = request.body === undefined ? undefined : new Uint8Array(request.body);
  const prepared = prepareRequest(
    connection,
    request.path,
    request.query,
    request.headers,
    readEnvironment,
    request.bodyEncoding,
  );
  // Server-only ambient read: this module never ships to browsers (the
  // browser action client uses fetchJsonAction, not performRequest).
  const resolved = scopeRaceInput(request.budget);
  if (resolved.scopeSignal?.aborted) throw transportFault({ kind: "timeout", milliseconds: 0 });
  const effective = cappedTimeout(connection.timeoutMilliseconds, resolved.budget);
  if (effective <= 0) throw transportFault({ kind: "timeout", milliseconds: 0 });
  const deadline = new Deadline(effective, request.ownerSignal);
  if (resolved.scopeSignal !== undefined) deadline.expireOn(resolved.scopeSignal);
  let response: Response | undefined,
    consuming = false;
  try {
    response = await nativeOperation(
      async () => {
        let received: Response;
        try {
          received = await (request.exchange ?? fetch)(prepared.url, {
            method: request.method,
            headers: prepared.headers,
            body,
            redirect: "manual",
            credentials: "omit",
            signal: deadline.signal,
          });
        } catch (cause) {
          deadline.check();
          // Classification is confined to the actual fetch call, not ownership,
          // argument preparation, or response decoding.
          if (
            cause instanceof TypeError ||
            cause instanceof DOMException ||
            (cause instanceof Error && "code" in cause)
          )
            throw transportFault({ kind: "transport", phase: "connect" });
          throw cause;
        }
        if (deadline.signal.aborted && received.body)
          cleanupOperation(() => received.body!.cancel());
        return received;
      },
      [],
      deadline,
    );
    deadline.check();
    const metadata = Object.freeze({
      status: response.status,
      headers: headerSnapshot(response.headers),
    });
    if (response.status < 200 || response.status > 599)
      throw transportFault({ kind: "transport", phase: "protocol" });
    if (!request.envelope && (response.status < 200 || response.status >= 300))
      throw transportFault({ kind: "status", ...metadata });
    consuming = true;
    const bytes = await readBody(
      response,
      connection.maxBodyBytes,
      deadline,
      () => {},
      (operation) => nativeOperation(operation, [], deadline),
      cleanupOperation,
    );
    const result = await nativeCompletion(() => decode(bytes, metadata), [], deadline);
    deadline.check();
    return checkedCompletion(result);
  } finally {
    if (response?.body && !consuming) cleanupOperation(() => response!.body!.cancel());
    deadline.dispose();
  }
}
