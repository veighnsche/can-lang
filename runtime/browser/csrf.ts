// Browser-profile CSRF factory. Session-bound tokens need a server secret,
// so the capability gate rejects CSRF operations at check time with span
// evidence. The stubs below only satisfy linking and fail closed when
// reached.
import type { AssertionContext, Completion } from "../completion.ts";
import type { DomainRuntime } from "../domain-core.ts";

const unavailable = (): never => {
  throw new TypeError("CSRF tokens are unavailable in the browser profile");
};

export function createCSRF(_domain: DomainRuntime, _ids: Readonly<{ invalid: string }>) {
  return Object.freeze({
    async generate(
      _secret: unknown,
      _session: unknown,
      _expiresInMs: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<string>> {
      unavailable();
    },
    async verify(
      _secret: unknown,
      _session: unknown,
      _token: unknown,
      _maxAgeMs: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<boolean>> {
      unavailable();
    },
  });
}
