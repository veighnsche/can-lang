package emit

import (
	"strings"
	"testing"
)

func TestExactESMEdges(t *testing.T) {
	modules := []Module{
		{Path: "entry.ts", Imports: []ModuleImport{{Target: "packages/p-a/a.ts"}}, Body: ""},
		{Path: "packages/p-a/a.ts", Imports: []ModuleImport{{Target: "runtime/r-1/data.ts", Names: []ImportName{{"record", "$record"}}}, {Target: "packages/p-b/b.ts", TypeOnly: true, Names: []ImportName{{"Value", "$Value"}}}}, Body: "const value: $Value = 1n;"},
		{Path: "packages/p-b/b.ts", Body: "export type Value = bigint;"}, {Path: "runtime/r-1/data.ts", Body: "export const record = Object.freeze;"},
	}
	result, err := Modules(modules)
	if err != nil {
		t.Fatal(err)
	}
	got := string(result[1].Bytes)
	if !strings.Contains(got, `from "../../runtime/r-1/data.ts"`) || !strings.Contains(got, `import type { Value as $Value } from "../p-b/b.ts"`) {
		t.Fatal(got)
	}
	if len(result[1].Imports) != 2 {
		t.Fatal("type-only edge lost")
	}
	modules[1].Imports[1].Target = "packages/missing.ts"
	if _, err := Modules(modules); err == nil {
		t.Fatal("missing type-only import accepted")
	}
}
