import { invoke, type Completion } from "../completion.ts";
import { type FailureOrigin } from "../failure.ts";
import { assertionEqual, completionEqual, completionValueEqual } from "./runner.ts";
import { provideRawHTTP, type RawFixtureInput } from "./provider.ts";
import {
  contextReport,
  scheduledFixture,
  fixtureMismatch,
  suppliedEvidence,
  type AssertionContext,
} from "./context.ts";

type FixtureRow = Readonly<{
  selector: string;
  arguments: () => Promise<Completion<readonly unknown[]>>;
  expected: () => Promise<Completion>;
  raw?: Readonly<{ operation: string; spec: RawFixtureInput }>;
}>;
// Allocation occurs only at a proven frame barrier; rows are never searched by arguments.
export async function withFixture(
  context: AssertionContext | undefined,
  identity: string,
  rows: readonly FixtureRow[],
  actualArguments: readonly unknown[],
  run: () => Completion | Promise<Completion>,
  origin: FailureOrigin,
): Promise<Completion> {
  if (context === undefined) return invoke(run, origin);
  const selected = rows.filter((row) => row.selector === contextReport(context).root.name);
  if (selected.length === 0) return invoke(run, origin);
  return scheduledFixture(context, identity, selected.length, origin, async (allocation) => {
    const row = selected[allocation.row];
    const expectedArguments = await invoke(row.arguments, origin);
    if (expectedArguments.kind !== "ok")
      throw fixtureMismatch(context, allocation, "malformed fixture", origin);
    // Raw rows check arguments/fingerprints: genuine byte tokens match by
    // content while every other opaque value keeps identity comparison.
    const sameArguments =
      row.raw !== undefined
        ? completionValueEqual(actualArguments, expectedArguments.value)
        : assertionEqual(actualArguments, expectedArguments.value);
    if (!sameArguments) throw fixtureMismatch(context, allocation, "argument mismatch", origin);
    if (row.raw !== undefined) {
      provideRawHTTP(context, row.raw.operation, row.raw.spec);
      const actual = await invoke(run, origin);
      const expected = await invoke(row.expected, origin);
      if (expected.kind === "standard")
        throw fixtureMismatch(context, allocation, "malformed fixture", origin);
      if (!completionEqual(actual, expected))
        throw fixtureMismatch(context, allocation, "outcome mismatch", origin);
      return actual;
    }
    const expected = await invoke(row.expected, origin);
    if (expected.kind === "standard")
      throw fixtureMismatch(context, allocation, "malformed fixture", origin);
    suppliedEvidence(context);
    return expected;
  });
}
