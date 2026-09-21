import { invoke, type Completion } from "../completion.ts";
import { type FailureOrigin } from "../failure.ts";
import { assertionEqual } from "./runner.ts";
import { contextReport, fixtureIndex, suppliedEvidence, violation, type AssertionContext } from "./context.ts";

type FixtureRow = Readonly<{
  selector: string;
  arguments: () => Promise<Completion<readonly unknown[]>>;
  expected: () => Promise<Completion>;
}>;
// This sequential table is deliberately separate from I18's later participant
// reservation/barrier scheduler. No row is found by searching argument values.
export async function withFixture(
  context: AssertionContext | undefined, identity: string, rows: readonly FixtureRow[],
  actualArguments: readonly unknown[], run: () => Promise<Completion>, origin: FailureOrigin,
): Promise<Completion> {
  if (context === undefined) return invoke(run, origin);
  const selected = rows.filter(row => row.selector === contextReport(context).root.name);
  if (selected.length === 0) return invoke(run, origin);
  const row = selected[fixtureIndex(context, identity, selected.length, origin)];
  const expectedArguments = await invoke(row.arguments, origin);
  if (expectedArguments.kind !== "ok") throw violation(context, "malformed fixture", origin);
  if (!assertionEqual(actualArguments, expectedArguments.value)) throw violation(context, "argument mismatch", origin);
  const expected = await invoke(row.expected, origin);
  if (expected.kind === "standard") throw violation(context, "malformed fixture", origin);
  suppliedEvidence(context);
  return expected;
}
