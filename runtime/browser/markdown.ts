// Browser-profile Markdown factory. Rendering needs the native Markdown
// engine, so the capability gate rejects Markdown operations at check time
// with span evidence. The stubs below only satisfy linking and fail closed
// when reached.
import type { AssertionContext, Completion } from "../completion.ts";
import type { DomainRuntime } from "../domain-core.ts";

type Ids = Readonly<{ overLimit: string; htmlStructure: string; htmlURL: string }>;

const unavailable = (): never => {
  throw new TypeError("Markdown rendering is unavailable in the browser profile");
};

export function createMarkdown(_domain: DomainRuntime, _ids: Ids) {
  return Object.freeze({
    async renderTextHTML(
      _source: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<string>> {
      return unavailable();
    },
    async renderSafe(_source: unknown, _context?: AssertionContext): Promise<Completion<unknown>> {
      return unavailable();
    },
  });
}
