package emit

import (
	"strings"
	"testing"
)

func TestActionJSONEmissionCarriesResponseSchemas(t *testing.T) {
	program := actionEmitProgram(t, map[string]string{"src/web/web.can": actionEmitWeb})
	if len(program.Actions) != 2 {
		t.Fatalf("checked %d actions, want 2", len(program.Actions))
	}
	for _, action := range program.Actions {
		if action.ResponseSchema == nil {
			t.Fatalf("JSON action %s has no checked response schema", action.Symbol.ID)
		}
	}
	joined := emittedBody(t, program)
	if count := strings.Count(joined, `"responseSchema":`); count != 2 {
		t.Fatalf("emitted %d response schemas, want 2 (POST save and GET load)", count)
	}
	for _, action := range program.Actions {
		want := `"responseSchema":{"root":"` + action.ResponseSchema.Root + `"`
		if !strings.Contains(joined, want) {
			t.Fatalf("emitted actions omit %s response root %s", action.Symbol.ID, want)
		}
		if action.ResponseSchema.Root != action.Returns.Identity() {
			t.Fatalf("response root = %s, want result %s", action.ResponseSchema.Root, action.Returns.Identity())
		}
	}
}

const actionEmitFormWeb = "package web\n" +
	"    provides [append_line, line_wire, stored, store_failed, store_outcome]\n" +
	"    uses []\n" +
	"record line_wire\n" +
	"    str name\n" +
	"    str amount\n" +
	"record stored\n" +
	"    str label\n" +
	"record store_failed\n" +
	"    str reason\n" +
	"variant store_outcome\n" +
	"    stored\n" +
	"    store_failed\n" +
	"action append_line\n" +
	"    post \"/lines\"\n" +
	"    form line_wire limit 2048\n" +
	"    returns store_outcome\n" +
	"    body html\n" +
	"    cases\n" +
	"        stored status 200 swap inner\n" +
	"        store_failed status 422 swap inner\n" +
	"fn void main\n" +
	"    emits []\n" +
	"    given\n" +
	"        str[] arguments\n" +
	"    asserts\n" +
	"        empty: [] => ok\n" +
	"    ok\n"

func TestActionJSONEmissionOmitsFormResponseSchema(t *testing.T) {
	program := actionEmitProgram(t, map[string]string{"src/web/web.can": actionEmitFormWeb})
	if len(program.Actions) != 1 {
		t.Fatalf("checked %d actions, want 1", len(program.Actions))
	}
	if program.Actions[0].ResponseSchema != nil {
		t.Fatal("form action gained a JSON response schema")
	}
	joined := emittedBody(t, program)
	if strings.Contains(joined, `"responseSchema":`) {
		t.Fatal("form action emitted a JSON response schema")
	}
	if !strings.Contains(joined, `"input":{"mode":"form","type":"`) {
		t.Fatal("form action lost its form input contract")
	}
	if !strings.Contains(joined, `"body":"html"`) {
		t.Fatal("form action lost its html response mode")
	}
}
