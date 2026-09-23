import { resolve, join, basename, extname } from "node:path";
import { success, type Completion, type AssertionContext } from "../../completion.ts";
// Pure path computation: no domain, no failures, no assertion boundary.
export function createPath() {
  return Object.freeze({
    async resolvePath(
      base: string,
      parts: readonly string[],
      _context?: AssertionContext,
    ): Promise<Completion<string>> {
      return success(resolve(base, ...parts));
    },
    async joinPath(
      parts: readonly string[],
      _context?: AssertionContext,
    ): Promise<Completion<string>> {
      return success(join(...parts));
    },
    async basename(path: string, _context?: AssertionContext): Promise<Completion<string>> {
      return success(basename(path));
    },
    async extension(path: string, _context?: AssertionContext): Promise<Completion<string>> {
      return success(extname(path));
    },
  });
}
