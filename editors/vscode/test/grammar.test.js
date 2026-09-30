const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const tm = require("vscode-textmate");
const onig = require("vscode-oniguruma");

const extensionRoot = path.resolve(__dirname, "..");
const sourceRoot = path.resolve(extensionRoot, "..", "..");
let grammar;

async function setup() {
  if (grammar) return grammar;
  const wasm = fs.readFileSync(require.resolve("vscode-oniguruma/release/onig.wasm"));
  await onig.loadWASM(wasm.buffer.slice(wasm.byteOffset, wasm.byteOffset + wasm.byteLength));
  const registry = new tm.Registry({
    onigLib: Promise.resolve({
      createOnigScanner: (patterns) => new onig.OnigScanner(patterns),
      createOnigString: (s) => new onig.OnigString(s),
    }),
    loadGrammar: async () => tm.parseRawGrammar(
      fs.readFileSync(path.join(extensionRoot, "syntaxes/can.tmGrammar.json"), "utf8"),
      "can.tmGrammar.json",
    ),
  });
  grammar = await registry.loadGrammar("source.can");
  return grammar;
}

async function tokenize(lines) {
  const g = await setup();
  let stack = tm.INITIAL;
  return lines.map((line) => {
    const result = g.tokenizeLine(line, stack);
    stack = result.ruleStack;
    return result.tokens.map(({ startIndex, endIndex, scopes }) => ({
      text: line.slice(startIndex, endIndex), startIndex, endIndex, scopes,
    }));
  });
}

function scopesAt(lines, tokens, line, needle, occurrence = 0) {
  let offset = -1;
  for (let i = 0; i <= occurrence; i++) offset = lines[line].indexOf(needle, offset + 1);
  assert.notEqual(offset, -1, `missing ${needle} in ${lines[line]}`);
  const token = tokens[line].find((t) => t.startIndex <= offset && t.endIndex > offset);
  assert.ok(token, `no token for ${needle}`);
  return token.scopes;
}

function has(scopes, scope) { assert.ok(scopes.includes(scope), `${scope} missing from ${scopes}`); }
function lacks(scopes, scope) { assert.ok(!scopes.includes(scope), `${scope} unexpectedly in ${scopes}`); }

test("nested comments stay comments, including apparent code", async () => {
  const lines = ["/* outer", "   /* inner fn record */", "   tail */", "fn int next"];
  const tokens = await tokenize(lines);
  for (const [line, word] of [[0,"outer"],[1,"fn"],[1,"record"],[2,"tail"]]) {
    has(scopesAt(lines, tokens, line, word), "comment.block.can");
  }
  has(scopesAt(lines, tokens, 3, "fn"), "keyword.declaration.function.can");
});

test("strings honor escapes, raw content, triple lines and unfinished ordinary lines", async () => {
  const lines = [
    'str normal = "line\\nquote\\" // text"',
    'str raw = r"\\n /* literal */"',
    'str poem = r"""',
    '    /* still literal */',
    '"""',
    'str ordinaryTriple = """',
    '    escaped \\" quote',
    '"""',
    'str broken = "unfinished',
    'fn int next',
  ];
  const tokens = await tokenize(lines);
  has(scopesAt(lines, tokens, 0, "\\n"), "constant.character.escape.can");
  has(scopesAt(lines, tokens, 0, "//"), "string.quoted.double.can");
  has(scopesAt(lines, tokens, 1, "\\n"), "string.quoted.double.raw.can");
  lacks(scopesAt(lines, tokens, 1, "\\n"), "constant.character.escape.can");
  has(scopesAt(lines, tokens, 3, "/*"), "string.quoted.triple.raw.can");
  lacks(scopesAt(lines, tokens, 3, "/*"), "comment.block.can");
  has(scopesAt(lines, tokens, 6, "\\\""), "constant.character.escape.can");
  has(scopesAt(lines, tokens, 8, "unfinished"), "string.quoted.double.can");
  has(scopesAt(lines, tokens, 9, "fn"), "keyword.declaration.function.can");
});

test("current operators and numeric forms use ordered TextMate matches", async () => {
  const line = "0xAB 0b101 0o77 42 1.25e-2 => ** << >> <= >= ... .. & | ^ ~ < > + - * / %";
  const lines = [line];
  const tokens = await tokenize(lines);
  for (const [word, scope] of [["0xAB","constant.numeric.hex.can"],["0b101","constant.numeric.binary.can"],["0o77","constant.numeric.octal.can"],["42","constant.numeric.decimal.can"],["1.25e-2","constant.numeric.decimal.can"]]) {
    has(scopesAt(lines, tokens, 0, word), scope);
  }
  for (const op of ["=>","**","<<",">>","<=",">=","...","..","&","|","^","~","<",">","+","-","*","/","%"])
    assert.ok(tokens[0].some((t) => t.text === op && t.scopes.includes("keyword.operator.can")), `operator ${op} not scoped`);
});

test("contextual declaration names, tags and generic angles stay contextual", async () => {
  const lines = [
    "package app", "    provides [owner]", "    uses []",
    "owner record owner<item>", "fn int owner", "    given", "        int owner",
    "    asserts", "        sample: 3 => ok 4", "    ok owner",
    "int comparison = a < b > c", "str text = owner",
  ];
  const tokens = await tokenize(lines);
  has(scopesAt(lines,tokens,0,"app"),"entity.name.namespace.can");
  has(scopesAt(lines,tokens,3,"owner"),"storage.modifier.can");
  has(scopesAt(lines,tokens,3,"record"),"keyword.declaration.can");
  has(scopesAt(lines,tokens,3,"item"),"entity.name.type.parameter.can");
  has(scopesAt(lines,tokens,4,"owner"),"entity.name.function.can");
  has(scopesAt(lines,tokens,8,"sample"),"entity.name.tag.can");
  lacks(scopesAt(lines,tokens,6,"owner"),"entity.name.tag.can");
  lacks(scopesAt(lines,tokens,11,"owner"),"keyword.declaration.can");
  has(scopesAt(lines,tokens,10,"<"),"keyword.operator.can");
});

test("maintained compiler fixture tokenizes without comment or string leaks", async () => {
  const sample = fs.readFileSync(path.join(sourceRoot,"compiler/testdata/current/lexer/literals.can"),"utf8").trimEnd().split(/\r?\n/);
  const tokens = await tokenize(sample);
  has(scopesAt(sample,tokens,0,"package"),"keyword.control.module.can");
  has(scopesAt(sample,tokens,4,"r\""),"string.quoted.double.raw.can");
  has(scopesAt(sample,tokens,6,"1.25e-2"),"constant.numeric.decimal.can");
  has(scopesAt(sample,tokens,8,"\\backslashes"),"string.quoted.triple.raw.can");
});

test("language configuration declares layout and safe pairs", () => {
  const config = JSON.parse(fs.readFileSync(path.join(extensionRoot,"language-configuration.json"),"utf8"));
  assert.deepEqual(config.comments.blockComment,["/*","*/"]);
  assert.match("fn int square", new RegExp(config.indentationRules.increaseIndentPattern));
  assert.match("    asserts", new RegExp(config.indentationRules.increaseIndentPattern));
  assert.match("fetch receipt retrieve", new RegExp(config.indentationRules.increaseIndentPattern));
  assert.match("    match value", new RegExp(config.indentationRules.increaseIndentPattern));
  assert.match("    }", new RegExp(config.indentationRules.decreaseIndentPattern));
  assert.deepEqual(config.autoClosingPairs.find((p) => p.open === '"').notIn,["string","comment"]);
});

test("error declarations and braces keep distinct scopes", async () => {
  const lines = ["error unavailable{str reason}", "fn int next", "    emits {unavailable}"];
  const tokens = await tokenize(lines);
  has(scopesAt(lines,tokens,0,"error"),"keyword.declaration.error.can");
  has(scopesAt(lines,tokens,0,"{"),"punctuation.section.can");
  has(scopesAt(lines,tokens,2,"{"),"punctuation.section.can");
});

test("function result generics are scoped without turning comparisons into generics", async () => {
  const lines = ["fn sql::decision<int> decide", "fn box<array<int>> nested", "int value = a<b>c"];
  const tokens = await tokenize(lines);
  has(scopesAt(lines,tokens,0,"<"),"punctuation.definition.generic.can");
  has(scopesAt(lines,tokens,0,"int"),"entity.name.type.parameter.can");
  has(scopesAt(lines,tokens,0,"decide"),"entity.name.function.can");
  has(scopesAt(lines,tokens,1,"<"),"punctuation.definition.generic.can");
  has(scopesAt(lines,tokens,1,"<",1),"punctuation.definition.generic.can");
  has(scopesAt(lines,tokens,1,"int"),"entity.name.type.parameter.can");
  lacks(scopesAt(lines,tokens,2,"<"),"punctuation.definition.generic.can");
});

test("real generic function headers close before subsequent declaration bodies", async () => {
  const fixture = fs.readFileSync(path.join(sourceRoot, "compiler/testdata/current/generics/chain-helpers.can"), "utf8").trimEnd().split(/\r?\n/);
  const tokens = await tokenize(fixture);
  const wrap = fixture.findIndex((line) => line === "fn box<item> wrap<item>");
  const pass = fixture.findIndex((line) => line === "fn item pass<item>");
  assert.ok(wrap > 0 && pass > 0);
  has(scopesAt(fixture,tokens,wrap,"wrap"),"entity.name.function.can");
  has(scopesAt(fixture,tokens,wrap,"<",1),"punctuation.definition.generic.can");
  has(scopesAt(fixture,tokens,pass,"<"),"punctuation.definition.generic.can");
  has(scopesAt(fixture,tokens,wrap+1,"emits"),"keyword.control.section.can");
  lacks(scopesAt(fixture,tokens,wrap+1,"emits"),"meta.generic.can");
  const synthetic = ["fn box<item> wrap<item>","    emits {}","    given","        item value","record event"];
  const scoped = await tokenize(synthetic);
  has(scopesAt(synthetic,scoped,4,"record"),"keyword.declaration.can");
});

test("qualified calls, scenario tags and contextual names keep their roles", async () => {
  const lines = [
    "int value = call text::from_int(7)",
    "    scenario checkout: 1 => ok 1",
    "str when = \"later\"",
    "int concurrent = 3",
    "int race = 4",
    "    result[] values = match call concurrent",
    "    when",
  ];
  const tokens = await tokenize(lines);
  has(scopesAt(lines,tokens,0,"from_int"),"entity.name.function.can");
  lacks(scopesAt(lines,tokens,0,"from_int"),"entity.name.type.can");
  has(scopesAt(lines,tokens,1,"checkout"),"entity.name.tag.can");
  for (const [line,word] of [[2,"when"],[3,"concurrent"],[4,"race"]]) {
    has(scopesAt(lines,tokens,line,word),"variable.other.readwrite.can");
    lacks(scopesAt(lines,tokens,line,word),"keyword.control.can");
  }
  has(scopesAt(lines,tokens,5,"concurrent"),"keyword.control.coordination.can");
  has(scopesAt(lines,tokens,6,"when"),"keyword.control.section.can");
});
