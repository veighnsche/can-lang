import {
  compareInvocations,
  invocationPath,
  type InvocationIdentity,
  type InvocationPath,
} from "./lineage.ts";

declare const queuesBrand: unique symbol;
export type FixtureQueues = Readonly<{ [queuesBrand]: true }>;
export type Allocation = Readonly<{ table: string; row: number; path: InvocationPath }>;
export type QueueProblem =
  | "missing fixture"
  | "ambiguous fixture"
  | "unused fixture"
  | "unexpected live boundary";
export class FixtureQueueError extends Error {
  constructor(
    readonly reason: QueueProblem,
    readonly expected: Allocation,
    readonly actual: InvocationPath,
  ) {
    super(reason);
  }
}
type Table = { total: number; used: number; registered: InvocationIdentity; last?: Allocation };
type State = {
  root: InvocationIdentity;
  tables: Map<string, Table>;
  claimed: WeakMap<object, string>;
  closed: boolean;
};
const queues = new WeakMap<object, State>();
function state(value: FixtureQueues): State {
  const found = value !== null && typeof value === "object" ? queues.get(value) : undefined;
  if (!found) throw new TypeError("invalid fixture queues");
  return found;
}
export function fixtureQueues(root: InvocationIdentity): FixtureQueues {
  invocationPath(root);
  const value = Object.freeze(Object.create(null)) as FixtureQueues;
  queues.set(value, { root, tables: new Map(), claimed: new WeakMap(), closed: false });
  return value;
}
export function registerTable(
  value: FixtureQueues,
  table: string,
  total: number,
  identity: InvocationIdentity,
): void {
  const owner = state(value);
  compareInvocations(owner.root, identity);
  if (typeof table !== "string" || !table || !Number.isSafeInteger(total) || total < 0)
    throw new TypeError("invalid fixture table");
  const path = invocationPath(identity),
    expected = Object.freeze({ table, row: 0, path });
  if (owner.closed) throw new FixtureQueueError("unexpected live boundary", expected, path);
  const prior = owner.tables.get(table);
  if (prior && prior.total !== total)
    throw new FixtureQueueError(
      "ambiguous fixture",
      Object.freeze({ table, row: prior.used, path: invocationPath(prior.registered) }),
      path,
    );
  if (!prior) owner.tables.set(table, { total, used: 0, registered: identity });
}
// The barrier chooses which reserved invocation reaches this function next.
// Queue allocation consumes exactly one row; it has no argument-search API.
export function allocateFixture(
  value: FixtureQueues,
  table: string,
  total: number,
  identity: InvocationIdentity,
): Allocation {
  registerTable(value, table, total, identity);
  const owner = state(value),
    queue = owner.tables.get(table)!,
    path = invocationPath(identity);
  const allocation = Object.freeze({ table, row: queue.used, path });
  if (owner.claimed.has(identity))
    throw new FixtureQueueError("ambiguous fixture", allocation, path);
  if (queue.used >= queue.total) throw new FixtureQueueError("missing fixture", allocation, path);
  owner.claimed.set(identity, table);
  queue.used++;
  queue.last = allocation;
  return allocation;
}
export function closeQueues(value: FixtureQueues): readonly FixtureQueueError[] {
  const owner = state(value);
  if (owner.closed) throw new TypeError("fixture queues already closed");
  owner.closed = true;
  return Object.freeze(
    [...owner.tables]
      .filter(([, table]) => table.used !== table.total)
      .map(([name, table]) => {
        const actual = table.last?.path ?? invocationPath(table.registered);
        return new FixtureQueueError(
          "unused fixture",
          Object.freeze({ table: name, row: table.used, path: invocationPath(table.registered) }),
          actual,
        );
      }),
  );
}
