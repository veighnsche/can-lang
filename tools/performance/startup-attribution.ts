// Opt-in startup attribution probe: exact ordered AST inventory and fail-closed
// instrumentation of the owned generated $canInitialize body. Runs under node
// (the installed TypeScript 7 sync API requires node internals); uses only the
// installed `typescript` package plus node builtins. No network, no writes
// outside the given paths, nothing written on failure.
import { API } from "typescript/unstable/sync";
import {
  NodeFlags,
  SyntaxKind,
  isExpressionStatement,
  isFunctionDeclaration,
  isVariableStatement,
  visitEachChild,
} from "typescript/unstable/ast";
import { createHash } from "node:crypto";
import { Buffer } from "node:buffer";
import { lstatSync, readFileSync, renameSync, unlinkSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

const MARK = "__canStartupMark";
const SCHEMA = "can.startup-probe-manifest/1";

function fail(code: string, detail: string): never {
  process.stderr.write(`startup-probe-error ${code} ${detail}\n`);
  process.exit(2);
}

function sha256Hex(bytes: string): string {
  return createHash("sha256").update(bytes, "utf8").digest("hex");
}

function writeAtomic(path: string, content: string): void {
  const tmp = `${path}.tmp-${process.pid}`;
  try {
    writeFileSync(tmp, content);
    renameSync(tmp, path);
  } catch (error) {
    try { unlinkSync(tmp); } catch { /* best effort */ }
    throw error;
  }
}

function skipTrivia(text: string, pos: number): number {
  let i = pos;
  for (;;) {
    const c = text[i];
    if (c === " " || c === "\t" || c === "\n" || c === "\r") { i++; continue; }
    if (c === "/" && text[i + 1] === "/") {
      const end = text.indexOf("\n", i + 2);
      i = end < 0 ? text.length : end + 1;
      continue;
    }
    if (c === "/" && text[i + 1] === "*") {
      const end = text.indexOf("*/", i + 2);
      if (end < 0) return text.length;
      i = end + 2;
      continue;
    }
    return i;
  }
}

function calleeName(expression: any): string {
  if (expression.kind === SyntaxKind.Identifier) return expression.text;
  if (expression.kind === SyntaxKind.PropertyAccessExpression) {
    const base = expression.expression;
    const name = expression.name ? expression.name.text : "complex";
    return base && base.kind === SyntaxKind.Identifier ? `${base.text}.${name}` : name;
  }
  return "complex";
}

function collectFactories(statement: any): string[] {
  const found: string[] = [];
  const walk = (node: any): void => {
    if (node.kind === SyntaxKind.CallExpression) found.push(calleeName(node.expression));
    else if (node.kind === SyntaxKind.NewExpression) found.push(`new:${calleeName(node.expression)}`);
    visitEachChild(node, (child: any) => { walk(child); return child; });
  };
  walk(statement);
  return [...new Set(found)];
}

function containsAsync(statement: any): boolean {
  let hit = false;
  const walk = (node: any): void => {
    if (node.kind === SyntaxKind.AwaitExpression || node.kind === SyntaxKind.YieldExpression) hit = true;
    visitEachChild(node, (child: any) => { walk(child); return child; });
  };
  walk(statement);
  return hit;
}

function statementBinding(statement: any): { kind: string; binding: string } {
  if (isVariableStatement(statement)) {
    const list = statement.declarationList;
    if ((list.flags & NodeFlags.Const) === 0) fail("unsupported_statement", "non-const variable statement");
    const names: string[] = [];
    for (const declarator of list.declarations) {
      if (declarator.name.kind !== SyntaxKind.Identifier) fail("unsupported_statement", "non-identifier declarator");
      if (!declarator.initializer) fail("unsupported_statement", "declarator without initializer");
      names.push(declarator.name.text);
    }
    return { kind: "VariableStatement", binding: names.join(",") };
  }
  if (isExpressionStatement(statement)) {
    const expression = statement.expression;
    if (expression.kind === SyntaxKind.BinaryExpression && expression.operatorToken.kind === SyntaxKind.EqualsToken) {
      const left = expression.left;
      const binding = left.kind === SyntaxKind.Identifier ? left.text : `complex:${left.kind}`;
      return { kind: "ExpressionStatement", binding };
    }
    if (expression.kind === SyntaxKind.CallExpression) {
      return { kind: "ExpressionStatement", binding: `call:${calleeName(expression.expression)}` };
    }
    return { kind: "ExpressionStatement", binding: "call:complex" };
  }
  fail("unsupported_statement", `statement kind ${statement.kind}`);
}

function usage(): never {
  process.stderr.write("usage: startup-attribution.ts inventory --state S --manifest M\n      startup-attribution.ts instrument --state S --out O --manifest M --probe-specifier P\n");
  process.exit(2);
}

function argValue(args: string[], name: string): string {
  const i = args.indexOf(name);
  if (i < 0 || i + 1 >= args.length) usage();
  return args[i + 1];
}

const [command, ...rest] = process.argv.slice(2);
if (command !== "inventory" && command !== "instrument") usage();
const statePath = resolve(argValue(rest, "--state"));
const manifestPath = resolve(argValue(rest, "--manifest"));
const outPath = command === "instrument" ? resolve(argValue(rest, "--out")) : null;
const probeSpecifier = command === "instrument" ? argValue(rest, "--probe-specifier") : null;

let text: string;
try {
  text = readFileSync(statePath, "utf8");
} catch (error) {
  fail("io_error", `cannot read state: ${(error as Error).message}`);
}
if (text.includes(MARK)) fail("already_instrumented", "state already contains probe marks");
if (!/\.(ts|tsx|mts|cts)$/.test(statePath)) fail("parse_error", "unsupported state file type");

const api = new API({});
try {
  await api.ensureInitialized();
} catch (error) {
  fail("parse_error", `typescript service unavailable: ${(error as Error).message}`);
}
try {
  let sourceFile: any;
  let diagnostics: unknown;
  try {
    const snapshot = api.updateSnapshot({ openFiles: [statePath] });
    const project = snapshot.getDefaultProjectForFile(statePath);
    const program = project ? (project as any).program : undefined;
    sourceFile = program ? program.getSourceFile(statePath) : undefined;
    diagnostics = program ? program.getSyntacticDiagnostics(statePath) : undefined;
  } catch (error) {
    fail("parse_error", `typescript service failed: ${(error as Error).message}`);
  }
  if (!sourceFile) fail("parse_error", "typescript returned no source file");
  if (!Array.isArray(diagnostics) || diagnostics.length !== 0) {
    fail("has_diagnostics", `${Array.isArray(diagnostics) ? diagnostics.length : "?"} syntactic diagnostics`);
  }
  const initializers = sourceFile.statements.filter(
    (node: any) => isFunctionDeclaration(node) && node.name && node.name.text === "$canInitialize",
  );
  if (initializers.length === 0 || !initializers[0].body) fail("initializer_missing", "no $canInitialize implementation");
  if (initializers.length > 1) fail("initializer_ambiguous", "multiple $canInitialize declarations");
  const body = initializers[0].body.statements;
  if (!Array.isArray(body) || body.length === 0) fail("initializer_missing", "empty $canInitialize body");
  const statements = [];
  let previousEnd = -1;
  for (let index = 0; index < body.length; index++) {
    const node = body[index];
    const { kind, binding } = statementBinding(node);
    if (containsAsync(node)) fail("async_expression", `statement ${index} contains await/yield`);
    const start = skipTrivia(text, node.pos);
    const end = node.end;
    if (!(start < end) || start < previousEnd || end > text.length) fail("parse_error", `statement ${index} span invalid`);
    if (text[end - 1] !== ";") fail("missing_semicolon", `statement ${index} does not end with ;`);
    previousEnd = end;
    statements.push({
      index,
      start,
      end,
      bytes: Buffer.byteLength(text.slice(start, end), "utf8"),
      sha256: sha256Hex(text.slice(start, end)),
      kind,
      binding,
      factories: collectFactories(node),
    });
  }
  const manifest = {
    schema: SCHEMA,
    state_path: statePath,
    state_sha256: sha256Hex(text),
    state_bytes: Buffer.byteLength(text, "utf8"),
    initializer: { name: "$canInitialize", start: statements[0].start, end: statements[statements.length - 1].end, statement_count: statements.length },
    probe_specifier: probeSpecifier,
    instrumented_sha256: null as string | null,
    statements,
  };
  let instrumented: string | null = null;
  if (command === "instrument" && outPath && probeSpecifier) {
    const importLine = `import {${MARK} as ${MARK}} from "${probeSpecifier}";\n`;
    let output = importLine + text;
    for (let index = statements.length - 1; index >= 0; index--) {
      const { start, end } = statements[index];
      const offset = importLine.length;
      output =
        output.slice(0, offset + start) +
        `${MARK}(${index},0);\n` +
        output.slice(offset + start, offset + end) +
        `\n${MARK}(${index},1);` +
        output.slice(offset + end);
    }
    instrumented = output;
    manifest.instrumented_sha256 = sha256Hex(output);
  }
  const outputs: Array<[string, string]> =
    instrumented !== null && outPath
      ? [["out", outPath], ["manifest", manifestPath]]
      : [["manifest", manifestPath]];
  const seen = new Set<string>([statePath]);
  for (const [label, candidate] of [["state", statePath], ...outputs]) {
    if (label !== "state" && seen.has(candidate)) {
      fail("path_alias", `${label} aliases another probe path: ${candidate}`);
    }
    seen.add(candidate);
  }
  for (const [label, candidate] of outputs) {
    try {
      lstatSync(candidate);
      fail("output_exists", `${label} already exists: ${candidate}`);
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code !== "ENOENT") {
        fail("io_error", `cannot inspect ${label}: ${(error as Error).message}`);
      }
    }
  }
  const written: Array<{ path: string; dev: number; ino: number }> = [];
  const recordCreated = (path: string): void => {
    const info = lstatSync(path);
    written.push({ path, dev: info.dev, ino: info.ino });
  };
  try {
    if (instrumented !== null && outPath) {
      writeAtomic(outPath, instrumented);
      recordCreated(outPath);
    }
    writeAtomic(manifestPath, JSON.stringify(manifest, null, 2) + "\n");
    recordCreated(manifestPath);
  } catch (error) {
    for (const created of written) {
      try {
        const info = lstatSync(created.path);
        if (info.dev === created.dev && info.ino === created.ino) unlinkSync(created.path);
      } catch { /* best effort; never touch replaced files */ }
    }
    fail("io_error", `cannot write output: ${(error as Error).message}`);
  }
} finally {
  api.close();
}
