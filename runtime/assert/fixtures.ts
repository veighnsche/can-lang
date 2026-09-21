import { invoke, type Completion } from "../completion.ts";
import { type FailureOrigin } from "../failure.ts";
import { assertionEqual } from "./runner.ts";
import { contextReport, scheduledFixture, fixtureMismatch, suppliedEvidence, type AssertionContext } from "./context.ts";

type FixtureRow = Readonly<{
  selector: string;
  arguments: () => Promise<Completion<readonly unknown[]>>;
  expected: () => Promise<Completion>;
}>;
// Allocation occurs only at a proven frame barrier; rows are never searched by arguments.
export async function withFixture(
  context: AssertionContext | undefined, identity: string, rows: readonly FixtureRow[],
  actualArguments: readonly unknown[], run: () => Completion | Promise<Completion>, origin: FailureOrigin,
): Promise<Completion> {
  if (context === undefined) return invoke(run, origin);
  const selected = rows.filter(row => row.selector === contextReport(context).root.name);
  if (selected.length === 0) return invoke(run, origin);
  return scheduledFixture(context,identity,selected.length,origin,async allocation=>{
  const row = selected[allocation.row];
  const expectedArguments = await invoke(row.arguments, origin);
  if (expectedArguments.kind !== "ok") throw fixtureMismatch(context, allocation, "malformed fixture", origin);
  if (!assertionEqual(actualArguments, expectedArguments.value)) throw fixtureMismatch(context, allocation, "argument mismatch", origin);
  const expected = await invoke(row.expected, origin);
  if (expected.kind === "standard") throw fixtureMismatch(context, allocation, "malformed fixture", origin);
  suppliedEvidence(context);
  return expected;
  });
}
