package browser

import (
	"strings"
	"testing"
)

func scanSource(t *testing.T, src string) ModuleScan {
	t.Helper()
	scan, err := ScanModule([]byte(src))
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	return scan
}

func operations(scan ModuleScan) []string {
	var out []string
	for _, finding := range scan.Findings {
		out = append(out, finding.Operation)
	}
	return out
}

func specifiers(scan ModuleScan) []string {
	var out []string
	for _, edge := range scan.Edges {
		out = append(out, edge.Specifier)
	}
	return out
}

func TestScanQuotedDataPasses(t *testing.T) {
	scan := scanSource(t, "export const a = \"Bun.\";\nexport const b = 'require(';\n"+
		"export const c = `node: introduction`;\n// Bun.write(x)\n/* process.env.HOME require(\"y\") */\n"+
		"export const re = /Bun\\.write/;\n")
	if len(scan.Findings) != 0 {
		t.Fatalf("quoted data findings: %+v", scan.Findings)
	}
	if len(scan.Edges) != 0 {
		t.Fatalf("quoted data edges: %+v", scan.Edges)
	}
}

func TestScanTemplateInterpolationIsCode(t *testing.T) {
	scan := scanSource(t, "export const a = `prefix ${Bun.write(\"x\")} suffix`;\n")
	if got := operations(scan); len(got) != 1 || got[0] != "Bun.write" {
		t.Fatalf("interpolation operations = %v", got)
	}
	nested := scanSource(t, "export const a = `outer ${\"inner\"} ${`deep ${process.env.HOME}`} end`;\n")
	if got := operations(nested); len(got) != 1 || got[0] != "process.env" {
		t.Fatalf("nested interpolation operations = %v", got)
	}
}

func TestScanStaticEdges(t *testing.T) {
	scan := scanSource(t, "import \"./side.ts\";\n"+
		"import { a } from \"./a.ts\";\n"+
		"import type { T } from \"../types.ts\";\n"+
		"export { b } from \"./b.ts\";\n"+
		"export type { U } from \"./u.ts\";\n"+
		"export * from \"./all.ts\";\n"+
		"export const local = 1;\n")
	want := []string{"./side.ts", "./a.ts", "../types.ts", "./b.ts", "./u.ts", "./all.ts"}
	if got := specifiers(scan); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("edges = %v, want %v", got, want)
	}
	typeOnly := map[string]bool{}
	for _, edge := range scan.Edges {
		typeOnly[edge.Specifier] = edge.TypeOnly
	}
	if !typeOnly["../types.ts"] || !typeOnly["./u.ts"] {
		t.Fatalf("type-only flags = %+v", typeOnly)
	}
	if typeOnly["./a.ts"] || typeOnly["./b.ts"] || typeOnly["./side.ts"] || typeOnly["./all.ts"] {
		t.Fatalf("value edges marked type-only: %+v", typeOnly)
	}
	if scan.Edges[1].Pos.Line != 2 || scan.Edges[1].Pos.Column != 18 {
		t.Fatalf("edge position = %+v", scan.Edges[1].Pos)
	}
}

func TestScanImportMemberAndKeyAreNotEdges(t *testing.T) {
	scan := scanSource(t, "export const api = { import: 1, export: 2 };\nfoo.import(\"./x.ts\");\nbar.export = 1;\n")
	if len(scan.Edges) != 0 {
		t.Fatalf("member/key edges: %+v", scan.Edges)
	}
	if len(scan.Findings) != 0 {
		t.Fatalf("member/key findings: %+v", scan.Findings)
	}
}

func TestScanDynamicImportIsFinding(t *testing.T) {
	scan := scanSource(t, "export async function load(): Promise<unknown> { return import(\"./lazy.ts\"); }\n")
	if got := operations(scan); len(got) != 1 || got[0] != "import()" {
		t.Fatalf("operations = %v", got)
	}
	if len(scan.Edges) != 0 {
		t.Fatalf("dynamic import recorded as edge: %+v", scan.Edges)
	}
}

func TestScanImportMetaAllowed(t *testing.T) {
	scan := scanSource(t, "export const url = import.meta.url;\n$canConfigureDiagnostics(import.meta.url);\n")
	if len(scan.Findings) != 0 {
		t.Fatalf("import.meta findings: %+v", scan.Findings)
	}
}

func TestScanHostMembers(t *testing.T) {
	scan := scanSource(t, "await Bun.write(Bun.stderr, line);\nprocess.exit(1);\n")
	want := []string{"Bun.write", "Bun.stderr", "process.exit"}
	if got := operations(scan); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("operations = %v, want %v", got, want)
	}
	first := scan.Findings[0].Pos
	if first.Line != 1 || first.Column != 6 {
		t.Fatalf("first finding position = %+v", first)
	}
}

func TestScanBareHostReferences(t *testing.T) {
	scan := scanSource(t, "export const check = typeof Bun;\n")
	if len(scan.Findings) != 0 {
		t.Fatalf("typeof sniff findings: %+v", scan.Findings)
	}
	scan = scanSource(t, "export const host = Bun;\nif (process === undefined) { throw new Error(\"x\"); }\n")
	if got := operations(scan); strings.Join(got, ",") != "Bun,process" {
		t.Fatalf("bare operations = %v", got)
	}
	scan = scanSource(t, "export const t = typeof Bun.write;\n")
	if got := operations(scan); len(got) != 1 || got[0] != "Bun.write" {
		t.Fatalf("typeof member operations = %v", got)
	}
}

func TestScanHostKeysAndTernary(t *testing.T) {
	scan := scanSource(t, "export const m = { Bun: 1, process: 2 };\nlabel: Bun;\n")
	if got := operations(scan); len(got) != 1 || got[0] != "Bun" {
		t.Fatalf("key/label operations = %v", got)
	}
	scan = scanSource(t, "export const pick = flag ? Bun : null;\n")
	if got := operations(scan); len(got) != 1 || got[0] != "Bun" {
		t.Fatalf("ternary operations = %v", got)
	}
}

func TestScanGlobalHostMembers(t *testing.T) {
	scan := scanSource(t, "export const w = globalThis.Bun;\nconst v = self.process;\n")
	if got := operations(scan); strings.Join(got, ",") != "globalThis.Bun,self.process" {
		t.Fatalf("operations = %v", got)
	}
	scan = scanSource(t, "export const w = globalThis.fetch;\n")
	if len(scan.Findings) != 0 {
		t.Fatalf("benign global member findings: %+v", scan.Findings)
	}
}

func TestScanRequireShapes(t *testing.T) {
	scan := scanSource(t, "export const fs = require(\"node:fs\");\n")
	if got := operations(scan); len(got) != 1 || got[0] != "require" {
		t.Fatalf("require call operations = %v", got)
	}
	// A member named require (the checks adapter) is not CJS require.
	scan = scanSource(t, "await $canChecks.require(ok, \"reason\", origin);\n")
	if len(scan.Findings) != 0 {
		t.Fatalf("member require findings: %+v", scan.Findings)
	}
	// A method definition named require is not CJS require.
	scan = scanSource(t, "export function createChecks() {\n  return Object.freeze({\n    async require(condition: boolean): Promise<void> {},\n  });\n}\n")
	if len(scan.Findings) != 0 {
		t.Fatalf("method require findings: %+v", scan.Findings)
	}
	scan = scanSource(t, "export const m = { require: 1 };\n")
	if len(scan.Findings) != 0 {
		t.Fatalf("key require findings: %+v", scan.Findings)
	}
}

func TestScanEvalAndFunction(t *testing.T) {
	scan := scanSource(t, "eval(\"1+1\");\nconst f = new Function(\"return 1\");\n")
	if got := operations(scan); strings.Join(got, ",") != "eval,Function" {
		t.Fatalf("operations = %v", got)
	}
	scan = scanSource(t, "if (handler instanceof Function) {}\ntype F = Function;\nadapter.eval(x);\n")
	if len(scan.Findings) != 0 {
		t.Fatalf("benign eval/Function findings: %+v", scan.Findings)
	}
}

func TestScanRegexAndDivision(t *testing.T) {
	scan := scanSource(t, "const languageClass = /^[A-Za-z0-9_-]+$/;\n"+
		"const cells = children.split(/\\0(\\d+)\\0/);\n"+
		"const half = total / count / scale;\n"+
		"if (/^x/.test(name)) { return 1; }\n")
	if len(scan.Findings) != 0 {
		t.Fatalf("regex/division findings: %+v", scan.Findings)
	}
	if len(scan.Edges) != 0 {
		t.Fatalf("regex/division edges: %+v", scan.Edges)
	}
}

func TestScanFailsClosed(t *testing.T) {
	for _, src := range []string{
		"export const a = \"unterminated;\n",
		"export const a = 'unterminated;\n",
		"export const a = `unterminated ${x};\n",
		"/* unterminated comment\n",
		"const re = /unterminated;\n",
		"const re = /x\n/;\n",
		"export const weird = #;\n",
		string([]byte{0xff, 0xfe}),
		"export const nul = \x00;\n",
	} {
		if _, err := ScanModule([]byte(src)); err == nil {
			t.Fatalf("malformed source admitted: %q", src)
		}
	}
	// A NUL byte inside a string literal is data, not a lex failure.
	data := scanSource(t, "export const s = \"a\x00b\";\n")
	if len(data.Findings) != 0 || len(data.Edges) != 0 {
		t.Fatalf("NUL data scan = %+v", data)
	}
}
