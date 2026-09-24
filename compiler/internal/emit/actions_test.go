package emit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

const actionEmitWeb = "package web\n" +
	"    provides [save_invoice, load_line, invoice_wire, saved, rejected, stale, denied, busy, save_outcome, found, missing, unavailable, load_outcome]\n" +
	"    uses []\n" +
	"record invoice_wire\n" +
	"    str label\n" +
	"    int seats\n" +
	"record saved\n" +
	"    str label\n" +
	"record rejected\n" +
	"    str reason\n" +
	"record stale\n" +
	"    int revision\n" +
	"record denied\n" +
	"    str reason\n" +
	"record busy\n" +
	"    str reason\n" +
	"variant save_outcome\n" +
	"    saved\n" +
	"    rejected\n" +
	"    stale\n" +
	"    denied\n" +
	"    busy\n" +
	"fn save_outcome save_validated\n" +
	"    emits []\n" +
	"    given\n" +
	"        invoice_wire body\n" +
	"    asserts\n" +
	"        sample: invoice_wire(\"inv-1\", 2) => ok saved(\"inv-1\")\n" +
	"    ok saved(body.label)\n" +
	"action save_invoice\n" +
	"    post \"/invoices/save\"\n" +
	"    body json invoice_wire\n" +
	"    handles save_validated\n" +
	"    result save_outcome\n" +
	"    cases\n" +
	"        saved => 200\n" +
	"        rejected => 422\n" +
	"        stale => 409\n" +
	"        denied => 403\n" +
	"        busy => 503\n" +
	"record found\n" +
	"    str label\n" +
	"record missing\n" +
	"    str reason\n" +
	"record unavailable\n" +
	"    str reason\n" +
	"variant load_outcome\n" +
	"    found\n" +
	"    missing\n" +
	"    unavailable\n" +
	"fn load_outcome load_validated\n" +
	"    emits []\n" +
	"    given\n" +
	"        str invoice_id\n" +
	"    asserts\n" +
	"        sample: \"inv-1\" => ok found(\"inv-1\")\n" +
	"    ok found(invoice_id)\n" +
	"action load_line\n" +
	"    get \"/invoices/{invoice_id}\"\n" +
	"    captures\n" +
	"        str invoice_id\n" +
	"    handles load_validated\n" +
	"    result load_outcome\n" +
	"    cases\n" +
	"        found => 200\n" +
	"        missing => 403\n" +
	"        unavailable => 503\n" +
	"fn void main\n" +
	"    emits []\n" +
	"    given\n" +
	"        str[] arguments\n" +
	"    asserts\n" +
	"        empty: [] => ok\n" +
	"    ok\n"

func actionEmitProgram(t *testing.T, files map[string]string) *check.Program {
	t.Helper()
	root := t.TempDir()
	files["can.project.json"] = `{"source_root":"src","error_registry":"can.errors.json"}`
	files["can.errors.json"] = `{"active":[],"retired":[]}`
	for name, text := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	program, err := check.CheckProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func emittedBody(t *testing.T, program *check.Program) string {
	t.Helper()
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, artifact := range artifacts {
		bodies = append(bodies, string(artifact.Bytes))
	}
	return strings.Join(bodies, "\n")
}

func TestActionEmissionFreezesContractTable(t *testing.T) {
	program := actionEmitProgram(t, map[string]string{"src/web/web.can": actionEmitWeb})
	if len(program.Actions) != 2 {
		t.Fatalf("checked %d actions, want 2", len(program.Actions))
	}
	webID := program.Actions[0].Symbol.Package.ID
	joined := emittedBody(t, program)
	for _, want := range []string{
		`export const $canActions = Object.freeze([`,
		`"identity":"` + webID + `::save_invoice"`,
		`"method":"POST"`,
		`"path":"/invoices/save"`,
		`"captures":[]`,
		`"body":{"mode":"json","type":"` + webID + `::invoice_wire","schema":{`,
		`"handler":"` + webID + `::save_validated"`,
		`"result":"` + webID + `::save_outcome"`,
		`"cases":[{"leaf":"` + webID + `::saved","status":200},{"leaf":"` + webID + `::rejected","status":422},{"leaf":"` + webID + `::stale","status":409},{"leaf":"` + webID + `::denied","status":403},{"leaf":"` + webID + `::busy","status":503}]`,
		`"identity":"` + webID + `::load_line"`,
		`"method":"GET"`,
		`"path":"/invoices/{invoice_id}"`,
		`"captures":[{"name":"invoice_id","type":"str"}]`,
		`"handler":"` + webID + `::load_validated"`,
		`"result":"` + webID + `::load_outcome"`,
		`"cases":[{"leaf":"` + webID + `::found","status":200},{"leaf":"` + webID + `::missing","status":403},{"leaf":"` + webID + `::unavailable","status":503}]`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("emitted actions omit %s", want)
		}
	}
	if count := strings.Count(joined, `"body":`); count != 1 {
		t.Fatalf("GET action gained a body contract: %d body entries", count)
	}
	if !strings.Contains(joined, `Object.freeze([{`) || !strings.Contains(joined, `]);`) {
		t.Fatal("action table is not a frozen array")
	}
}

func TestActionEmissionOmitsEmptyTable(t *testing.T) {
	program := actionEmitProgram(t, map[string]string{
		"src/main.can": "package app\n    provides []\n    uses []\nfn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n",
	})
	joined := emittedBody(t, program)
	if strings.Contains(joined, "$canActions") {
		t.Fatal("action-free program emitted an action table")
	}
}
