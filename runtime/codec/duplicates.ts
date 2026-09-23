import { Budget, maxDepth, reject, childPath } from "./budget.ts";

// A resource/key scanner, not a JSON parser. It does not construct values or
// admit syntax. The caller must run native JSON.parse before reporting the
// pending duplicate flag, so bounded malformed JSON retains syntax precedence.
export function scanJSON(text: string): string | undefined {
  const stack: { kind: string; path: string; index: number; key?: string; keys?: Set<string> }[] =
    [];
  const budget = new Budget();
  let duplicate: string | undefined;
  function valuePath(): string {
    const parent = stack.at(-1);
    if (!parent) return "";
    if (parent.kind === "[") return childPath(parent.path, parent.index++);
    const key = parent.key;
    parent.key = undefined;
    return key === undefined ? parent.path : childPath(parent.path, key);
  }
  const whitespace = (c: string | undefined) => c === " " || c === "\t" || c === "\r" || c === "\n";
  for (let i = 0; i < text.length;) {
    const ch = text[i];
    if (whitespace(ch) || ch === ":" || ch === ",") {
      i++;
      continue;
    }
    if (ch === "{" || ch === "[") {
      if (stack.length + 1 > maxDepth) reject("", "depth_limit");
      budget.visit(stack.length + 1, "");
      const path = valuePath();
      stack.push(
        ch === "{" ? { kind: ch, path, index: 0, keys: new Set() } : { kind: ch, path, index: 0 },
      );
      i++;
      continue;
    }
    if (ch === "}" || ch === "]") {
      stack.pop();
      i++;
      continue;
    }
    if (ch === '"') {
      const start = i++;
      while (i < text.length) {
        if (text[i] === "\\") {
          i += 2;
          continue;
        }
        if (text[i++] === '"') break;
      }
      let next = i;
      while (whitespace(text[next])) next++;
      const frame = stack.at(-1);
      if (frame?.keys && text[next] === ":") {
        try {
          const key: unknown = JSON.parse(text.slice(start, i));
          if (typeof key === "string") {
            if (frame.keys.has(key) && duplicate === undefined)
              duplicate = childPath(frame.path, key);
            frame.key = key;
            frame.keys.add(key);
          }
        } catch (cause) {
          // Only syntax failures are deferred; unrelated native faults escape.
          if (!(cause instanceof SyntaxError)) throw cause;
        }
      } else {
        valuePath();
        budget.visit(stack.length, "");
      }
      continue;
    }
    // Skip one primitive-like run. Malformed runs are still rejected by the
    // native parser. On valid input this counts every primitive exactly once.
    valuePath();
    budget.visit(stack.length, "");
    while (i < text.length && !whitespace(text[i]) && !'{}[],:"'.includes(text[i])) i++;
  }
  return duplicate;
}
