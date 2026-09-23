import {
  checkedCompletion,
  invoke,
  success,
  failure,
  errorPayload,
  type Completion,
} from "./completion.ts";
import { launchOwned } from "./owner.ts";
import { record } from "./data.ts";
import type { createDomainRuntime } from "./domain.ts";
import type { FailureOrigin } from "./failure.ts";

import { coordinationContexts, type AssertionContext } from "./assert/context.ts";

export type Mode = "all" | "settled" | "any" | "race";
type Indexed = Readonly<{ index: number; completion: Completion }>;
export type Selection = Readonly<
  | { kind: "all"; outcomes: readonly Completion[] }
  | { kind: "one"; index: number; completion: Completion }
  | { kind: "all-failed"; outcomes: readonly Completion[] }
>;
// Null-prototype private carriers never expose a Can payload to Promise
// resolution. Rejecting an adapter preserves its exact original completion.
function carrier<T extends object>(fields: T): Readonly<T> {
  return Object.freeze(Object.assign(Object.create(null), fields));
}
type ContextParticipant = Readonly<{
  captures: readonly unknown[];
  run: (context?: AssertionContext) => Completion | Promise<Completion>;
}>;
export async function settle(
  mode: Mode,
  participants: readonly ContextParticipant[],
  context?: AssertionContext,
  site?: string,
  positions?: readonly (readonly number[])[],
): Promise<Selection> {
  const frames =
    context === undefined ? undefined : coordinationContexts(context, site!, positions!, mode);
  let owner;
  try {
    owner = launchOwned(
      frames
        ? participants.map((participant, index) => ({
            captures: participant.captures,
            run: () => {
              frames.start(index);
              return participant.run(frames.contexts[index]);
            },
          }))
        : participants,
    );
  } catch (cause) {
    frames?.abort();
    throw cause;
  }
  try {
    const outcomes: Completion[] = [];
    outcomes.length = participants.length;
    const rejected = new Set<number>();
    let firstSuccess: readonly number[] | undefined;
    const tags = new WeakSet<object>();
    const promises = owner.promises.map((pending, index) =>
      pending.then((completion) => {
        checkedCompletion(completion);
        frames?.observed(index, completion);
        outcomes[index] = completion;
        const tagged = carrier({ index, completion });
        tags.add(tagged);
        if (completion.kind !== "ok") {
          rejected.add(index);
          throw tagged;
        }
        if (firstSuccess === undefined) firstSuccess = Object.freeze([...rejected, index]);
        return tagged;
      }),
    );
    const all = () => participants.map((_, index) => index);
    const selected = (tag: Indexed): Selection => {
      owner.publish(mode === "any" ? firstSuccess! : [tag.index]);
      return carrier({ kind: "one" as const, index: tag.index, completion: tag.completion });
    };
    if (mode === "settled") {
      await Promise.allSettled(promises);
      owner.publish(all());
      return carrier({ kind: "all" as const, outcomes: Object.freeze(outcomes) });
    }
    try {
      if (mode === "all") {
        await Promise.all(promises);
        owner.publish(all());
        return carrier({ kind: "all" as const, outcomes: Object.freeze(outcomes) });
      }
      if (mode === "any") return selected(await Promise.any(promises));
      return selected(await Promise.race(promises));
    } catch (cause) {
      if (mode === "any") {
        // Native any rejects only after every adapter rejected. Its AggregateError
        // supplies no Can identity: use the retained original completion sequence.
        if (
          outcomes.length !== participants.length ||
          outcomes.some((value) => value === undefined || value.kind === "ok")
        )
          throw new TypeError("invalid all-failed settlement");
        owner.publish(all());
        return carrier({ kind: "all-failed" as const, outcomes: Object.freeze(outcomes) });
      }
      if (typeof cause !== "object" || cause === null || !tags.has(cause)) throw cause;
      return selected(cause as Indexed);
    }
  } finally {
    frames?.selected();
  }
}

export type Handlers = Readonly<{
  each: (index: number, completion: Completion) => Completion | Promise<Completion>;
  shared: (index: number, completion: Completion) => Completion | Promise<Completion>;
  allFailed: (outcomes: readonly Completion[]) => Completion | Promise<Completion>;
}>;
// Dispatch starts after native selection. A failed handler exits directly,
// without another participant arm and without publishing a partial collection.
export async function handle(
  selection: Selection,
  handlers: Handlers,
  collect: boolean,
  origin: FailureOrigin,
): Promise<Completion> {
  if (selection.kind === "one")
    return invoke(() => handlers.shared(selection.index, selection.completion), origin);
  if (selection.kind === "all-failed")
    return invoke(() => handlers.allFailed(selection.outcomes), origin);
  const values: unknown[] = [];
  for (let index = 0; index < selection.outcomes.length; index++) {
    const result = await invoke(() => handlers.each(index, selection.outcomes[index]), origin);
    if (result.kind !== "ok") return result;
    if (collect) values.push(result.value);
  }
  return success(collect ? Object.freeze(values) : undefined);
}

// Failure payload injection happens once at aggregate construction. Domain
// values retain their original nominal records; standard values are the exact
// opaque snapshots, never diagnostic strings or native AggregateError objects.
export function aggregate(
  outcomes: readonly Completion[],
  domain: ReturnType<typeof createDomainRuntime>,
  identity: string,
  origin: FailureOrigin,
): Completion {
  const failures = Object.freeze(
    outcomes.map((outcome) => {
      checkedCompletion(outcome);
      if (outcome.kind === "ok") throw new TypeError("successful aggregate member");
      return outcome.kind === "domain" ? errorPayload(outcome) : outcome.value;
    }),
  );
  return failure(domain.create(identity, record(identity, [["failures", failures]]), origin));
}
