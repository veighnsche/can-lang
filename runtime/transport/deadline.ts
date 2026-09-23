// Private transport outcome markers; catalogue adapters construct domain values.
// Native causes never become authored error fields.
export type RequestReason =
  | "url"
  | "origin"
  | "path_query"
  | "fragment"
  | "userinfo"
  | "header_value"
  | "header_name"
  | "query_value"
  | "credential_value"
  | "content_type"
  | "instructions";
export type TransportProblem =
  | Readonly<{ kind: "timeout" }>
  | Readonly<{ kind: "transport"; phase: "connect" | "body" | "protocol" | "cancelled" }>
  | Readonly<{ kind: "limit"; limit: number }>
  | Readonly<{ kind: "invalid"; reason: RequestReason }>
  | Readonly<{ kind: "credential" }>
  | Readonly<{
      kind: "status";
      status: number;
      headers: readonly Readonly<{ name: string; value: string }>[];
    }>;
const problems = new WeakMap<object, TransportProblem>();
export function transportFault(problem: TransportProblem): Error {
  const error = new Error("Can transport failure");
  problems.set(error, Object.freeze({ ...problem }));
  return error;
}
export function transportProblem(cause: unknown): TransportProblem | undefined {
  return cause !== null && (typeof cause === "object" || typeof cause === "function")
    ? problems.get(cause)
    : undefined;
}

// Begins only after request preparation. The monotonic check also catches
// synchronous decoding that completes after expiry before a timer can run.
export class Deadline {
  readonly signal: AbortSignal;
  private readonly controller = new AbortController();
  private readonly start = performance.now();
  private readonly timer: ReturnType<typeof setTimeout>;
  private readonly expired: Promise<never>;
  private cancelled = false;
  private readonly cancel = () => {
    this.cancelled = true;
    this.controller.abort();
    this.reject(transportFault({ kind: "transport", phase: "cancelled" }));
  };
  private reject!: (reason: unknown) => void;
  constructor(
    private readonly milliseconds: number,
    private readonly ownerSignal?: AbortSignal,
  ) {
    if (!Number.isSafeInteger(milliseconds) || milliseconds < 1 || milliseconds > 2147483647)
      throw new TypeError("invalid transport deadline");
    this.signal = this.controller.signal;
    this.expired = new Promise((_, reject) => {
      this.reject = reject;
    });
    // Observation is installed even when synchronous preparation of a native
    // operation throws before the first wait is reached.
    void this.expired.catch(() => {});
    this.timer = setTimeout(() => this.expire(), milliseconds);
    if (ownerSignal?.aborted) this.cancel();
    else ownerSignal?.addEventListener("abort", this.cancel, { once: true });
  }
  private expire(): never | void {
    const cause = transportFault({ kind: "timeout" });
    this.controller.abort();
    this.reject(cause);
  }
  check(): void {
    if (
      performance.now() - this.start >= this.milliseconds ||
      (this.signal.aborted && !this.cancelled)
    ) {
      this.expire();
      throw transportFault({ kind: "timeout" });
    }
    if (this.cancelled) throw transportFault({ kind: "transport", phase: "cancelled" });
  }
  async wait<T>(operation: Promise<T>): Promise<T> {
    // Promise.race observes both branches, including native work settling late.
    try {
      return await Promise.race([operation, this.expired]);
    } finally {
      this.check();
    }
  }
  dispose(): void {
    clearTimeout(this.timer);
    this.ownerSignal?.removeEventListener("abort", this.cancel);
  }
}
