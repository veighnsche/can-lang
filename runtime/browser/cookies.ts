// Browser-profile cookie factory. Cookie jars need native validation and
// serialization, so the capability gate rejects cookie operations at check
// time with span evidence. The stubs below only satisfy linking and fail
// closed when reached. No handle is ever minted, so the kind predicate
// never matches.
import type { AssertionContext, Completion } from "../completion.ts";
import type { DomainRuntime } from "../domain-core.ts";

export const COOKIE_KIND = "cookie";

export function isCookieValue(_kind: string | undefined, _value: unknown): boolean {
  return false;
}

type Ids = Readonly<{
  invalid: string;
  collection: string;
  pair: string;
  some: string;
  none: string;
  samesiteStrict: string;
  samesiteLax: string;
  samesiteNone: string;
}>;

const unavailable = (): never => {
  throw new TypeError("cookie handling is unavailable in the browser profile");
};

export function createCookies(_domain: DomainRuntime, _ids: Ids) {
  return Object.freeze({
    async parse(_header: unknown, _context?: AssertionContext): Promise<Completion<unknown>> {
      return unavailable();
    },
    async get(
      _collection: unknown,
      _name: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return unavailable();
    },
    async make(
      _name: unknown,
      _value: unknown,
      _attributes: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return unavailable();
    },
    async serialize(_cookie: unknown, _context?: AssertionContext): Promise<Completion<string>> {
      return unavailable();
    },
    async remove(
      _name: unknown,
      _path: unknown,
      _domainName: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return unavailable();
    },
  });
}
