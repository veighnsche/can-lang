import { denyLiveBoundary, type AssertionContext } from "../assert/context.ts";
import { success, failure, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
import type { FailureOrigin } from "../failure.ts";
const origin: FailureOrigin = Object.freeze({
  source: "can:env",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
type Contracts = Readonly<{ invalidName: string; missing: string; some: string; none: string }>;
export function createEnvironment<Option>(
  domain: ReturnType<typeof createDomainRuntime>,
  types: Contracts,
  lookup: (name: string) => string | undefined,
) {
  const invalid = (name: string) =>
    failure(domain.create(types.invalidName, record(types.invalidName, [["name", name]]), origin));
  return Object.freeze({
    async required(name: string, context?: AssertionContext): Promise<Completion<string>> {
      denyLiveBoundary(context, origin);
      if (!/^[A-Z_][A-Z0-9_]*$/.test(name)) return invalid(name);
      const value = lookup(name);
      if (value === undefined)
        return failure(
          domain.create(types.missing, record(types.missing, [["variable", name]]), origin),
        );
      return success(value);
    },
    async optional(name: string, context?: AssertionContext): Promise<Completion<Option>> {
      denyLiveBoundary(context, origin);
      if (!/^[A-Z_][A-Z0-9_]*$/.test(name)) return invalid(name);
      const value = lookup(name);
      return success(
        (value === undefined
          ? record(types.none, [])
          : record(types.some, [["value", value]])) as Option,
      );
    },
  });
}
