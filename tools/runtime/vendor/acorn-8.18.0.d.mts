// Minimal declaration for the pinned Acorn parser used by output validation.
export function parse(source: string, options: {
  ecmaVersion: "latest";
  sourceType: "module";
}): { type: string; [key: string]: unknown };
