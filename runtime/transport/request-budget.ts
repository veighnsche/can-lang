// Shared request budget and the C-C unknown-write vocabulary (E01).
//
// One budget per request: each operation's effective deadline is min(its
// own bound, remaining budget). Serial per-operation budgets (5s+5s+5s =
// 15s user-visible) are rejected as the default; per-operation bounds
// remain mandatory inputs. The budget is pure time accounting — it holds
// no lease, revokes nothing, and never settles the operations it bounds.
// Ownership after a visible timeout, lease preservation, and supervisor
// escalation follow the written request-policy spec; this module publishes
// the accounting plus the one honest outcome vocabulary shared by
// SQL/S3/actions/workers. Catalogue adapters construct domain values from
// these markers; native causes never become authored error fields.
export type BudgetClock = () => number;

export type RequestBudget = Readonly<{
  totalMilliseconds: number;
  remainingMilliseconds(): number;
  effectiveMilliseconds(operationBound: number): number;
  expired(): boolean;
}>;

const MAX_MILLISECONDS = 2147483647;

function checkBound(name: string, value: number): void {
  if (!Number.isSafeInteger(value) || value < 1 || value > MAX_MILLISECONDS)
    throw new TypeError(`invalid request ${name}`);
}

// createRequestBudget starts one shared budget at creation. The clock
// defaults to the monotonic performance clock; tests inject a fake.
// remainingMilliseconds never reports below zero; effectiveMilliseconds
// reports zero once exhausted, meaning "return unknown-write without
// starting" — never a zero-length native wait.
export function createRequestBudget(
  totalMilliseconds: number,
  clock: BudgetClock = performance.now.bind(performance),
): RequestBudget {
  checkBound("budget", totalMilliseconds);
  const start = clock();
  const remaining = (): number => Math.max(0, totalMilliseconds - (clock() - start));
  return Object.freeze({
    totalMilliseconds,
    remainingMilliseconds: remaining,
    effectiveMilliseconds: (operationBound: number): number => {
      checkBound("operation bound", operationBound);
      return Math.max(0, Math.min(operationBound, remaining()));
    },
    expired: (): boolean => remaining() <= 0,
  });
}

export type UnknownWriteSource = "sql" | "s3" | "action" | "worker" | "fetch";

// UnknownWrite is the one honest outcome when the visible boundary
// returns before the native operation settles: the commit state is
// unknown (never silently resolved), the remaining work stays owned
// until settlement, and the supervisor is escalated to bound
// nonsettling work. The marker is frozen at creation; a late native
// settlement is observed separately and never rewrites it.
export type UnknownWrite = Readonly<{
  kind: "unknown-write";
  source: UnknownWriteSource;
  commit: "unknown";
  owned: true;
  escalation: "supervisor";
}>;

const sources: ReadonlySet<string> = new Set(["sql", "s3", "action", "worker", "fetch"]);

export function unknownWrite(source: UnknownWriteSource): UnknownWrite {
  if (typeof source !== "string" || !sources.has(source))
    throw new TypeError("invalid unknown-write source");
  return Object.freeze({
    kind: "unknown-write",
    source,
    commit: "unknown",
    owned: true,
    escalation: "supervisor",
  });
}

export function isUnknownWrite(value: unknown): value is UnknownWrite {
  if (value === null || (typeof value !== "object" && typeof value !== "function")) return false;
  const shaped = value as Partial<UnknownWrite>;
  return (
    shaped.kind === "unknown-write" &&
    typeof shaped.source === "string" &&
    sources.has(shaped.source) &&
    shaped.commit === "unknown" &&
    shaped.owned === true &&
    shaped.escalation === "supervisor"
  );
}
