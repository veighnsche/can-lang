package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// actionJSONHeader builds a web package with the given provides, catalogue
// uses and extra declarations. Actions below are self-contained so JSON
// wire tests do not depend on the T10 save/load domain. Exported actions
// must expose only exported wire, result and leaf types.
func actionJSONHeader(provides, uses string, decls ...string) string {
	return "package web\n    provides [" + provides + "]\n    uses [" + uses + "]\n" + strings.Join(decls, "") + actionMain
}

const actionJSONProvides = "seal_invoice, load_seal, seal_wire, sealed, seal_failed, seal_outcome"

const actionJSONSealedDomain = "owner record invoice_sealed\n" +
	"    str label\n" +
	"record seal_wire\n" +
	"    str label\n" +
	"    int seats\n" +
	"record sealed\n" +
	"    str label\n" +
	"record seal_failed\n" +
	"    str reason\n" +
	"variant seal_outcome\n" +
	"    sealed\n" +
	"    seal_failed\n"

const actionJSONSealedAction = "action seal_invoice\n" +
	"    post \"/invoices/sealed\"\n" +
	"    json seal_wire limit 8192\n" +
	"    returns seal_outcome\n" +
	"    body json\n" +
	"    cases\n" +
	"        sealed status 200\n" +
	"        seal_failed status 422\n"

const actionJSONLoadAction = "action load_seal\n" +
	"    get \"/invoices/sealed\"\n" +
	"    input none\n" +
	"    returns seal_outcome\n" +
	"    body json\n" +
	"    cases\n" +
	"        sealed status 200\n" +
	"        seal_failed status 422\n"

func TestActionJSONModesCheckResponseSchemas(t *testing.T) {
	text := actionJSONHeader(actionJSONProvides, "", actionJSONSealedDomain, actionJSONSealedAction, actionJSONLoadAction)
	program, err := programFixture(t, map[string]string{"src/web/web.can": text})
	if err != nil {
		t.Fatal(err)
	}
	save := actionByName(t, program, "seal_invoice")
	load := actionByName(t, program, "load_seal")
	if save.Input.Mode != "json" {
		t.Fatalf("POST save lost its json input: %+v", save.Input)
	}
	if load.Input.Mode != "none" || load.Input.Type != nil {
		t.Fatalf("GET load gained a wire input: %+v", load.Input)
	}
	for name, action := range map[string]*ActionDeclaration{"save": save, "load": load} {
		if action.ResponseSchema == nil {
			t.Fatalf("%s action has no checked JSON response schema", name)
		}
		if action.ResponseSchema.Root != action.Returns.Identity() {
			t.Fatalf("%s response root = %s, want result %s", name, action.ResponseSchema.Root, action.Returns.Identity())
		}
		nodes := map[string]bool{}
		for _, node := range action.ResponseSchema.Nodes {
			nodes[node.Identity] = true
		}
		for _, leaf := range action.Returns.Leaves() {
			if !nodes[leaf.Identity()] {
				t.Fatalf("%s response schema omits leaf %s", name, leaf.Identity())
			}
		}
	}
}

func TestActionJSONRejects(t *testing.T) {
	sealed := actionJSONSealedDomain + actionJSONSealedAction
	loaded := actionJSONLoadAction
	for name, tc := range map[string]struct {
		text string
		want string
	}{
		"nested owner in json body": {
			actionJSONHeader("seal_invoice, seal_wire, sealed, seal_failed, seal_outcome, invoice_sealed", "", strings.Replace(actionJSONSealedDomain, "record seal_wire\n    str label\n    int seats\n", "record seal_wire\n    str label\n    invoice_sealed inner\n", 1)+actionJSONSealedAction),
			"is not codec-admissible",
		},
		"owner leaf in post result": {
			actionJSONHeader(actionJSONProvides+", invoice_sealed", "", strings.Replace(sealed, "record seal_failed\n    str reason\n", "record seal_failed\n    str reason\n    invoice_sealed inner\n", 1)+loaded),
			"is not a JSON response type",
		},
		"owner leaf in get result": {
			actionJSONHeader(actionJSONProvides+", invoice_sealed", "", strings.Replace(sealed+loaded, "record seal_failed\n    str reason\n", "record seal_failed\n    str reason\n    invoice_sealed inner\n", 1)),
			"is not a JSON response type",
		},
		"opaque leaf in post result": {
			actionJSONHeader(actionJSONProvides, "bytes", strings.Replace(sealed, "record seal_failed\n    str reason\n", "record seal_failed\n    str reason\n    bytes::buffer payload\n", 1)+loaded),
			"is not a JSON response type",
		},
		"html leaf never renders as json": {
			actionJSONHeader(actionJSONProvides, "html", strings.Replace(sealed, "record seal_failed\n    str reason\n", "record seal_failed\n    str reason\n    html::safe page\n", 1)+loaded),
			"is not a JSON response type",
		},
		"opaque leaf in get result": {
			actionJSONHeader(actionJSONProvides, "bytes", strings.Replace(sealed+loaded, "record seal_failed\n    str reason\n", "record seal_failed\n    str reason\n    bytes::buffer payload\n", 1)),
			"is not a JSON response type",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := programFixture(t, map[string]string{"src/web/web.can": tc.text})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("JSON wire diagnostic omits %q: %v", tc.want, err)
			}
		})
	}
}

func TestActionFormResultSkipsJSONResponseGate(t *testing.T) {
	form := "record line_wire\n" +
		"    str name\n" +
		"    str amount\n" +
		"record stored\n" +
		"    bytes::buffer payload\n" +
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
		"        store_failed status 422 swap inner\n"
	text := "package web\n    provides [append_line, line_wire, stored, store_failed, store_outcome]\n    uses [bytes]\n" + form + actionMain
	program, err := programFixture(t, map[string]string{"src/web/web.can": text})
	if err != nil {
		t.Fatal(err)
	}
	action := actionByName(t, program, "append_line")
	if action.Input.Mode != "form" {
		t.Fatalf("form action lost its input: %+v", action.Input)
	}
	if action.ResponseSchema != nil {
		t.Fatal("form action gained a JSON response schema")
	}
}

func TestActionJSONRouteBodyLeafChanges(t *testing.T) {
	base := actionJSONHeader(actionJSONProvides, "", actionJSONSealedDomain, actionJSONSealedAction, actionJSONLoadAction)
	program, err := programFixture(t, map[string]string{"src/web/web.can": base})
	if err != nil {
		t.Fatal(err)
	}
	before := actionByName(t, program, "seal_invoice")
	beforeRoot := before.ResponseSchema.Root
	beforeSeats := actionSchemaNodeKind(t, before.Input.Schema, "seats")

	changed := strings.Replace(base, "post \"/invoices/sealed\"", "post \"/invoices/sealed-v2\"", 1)
	changed = strings.Replace(changed, "record seal_wire\n    str label\n    int seats\n", "record seal_wire\n    str label\n    str seats\n", 1)
	changed = strings.Replace(changed, "variant seal_outcome\n    sealed\n    seal_failed\n", "variant seal_outcome_v2\n    sealed\n    seal_failed\n", 1)
	changed = strings.Replace(changed, "    returns seal_outcome\n", "    returns seal_outcome_v2\n", 2)
	changed = strings.Replace(changed, "seal_failed, seal_outcome]", "seal_failed, seal_outcome_v2]", 1)
	rebuilt, err := programFixture(t, map[string]string{"src/web/web.can": changed})
	if err != nil {
		t.Fatal(err)
	}
	after := actionByName(t, rebuilt, "seal_invoice")
	if after.Path != "/invoices/sealed-v2" {
		t.Fatalf("route edit did not rebuild the path: %s", after.Path)
	}
	if got := actionSchemaNodeKind(t, after.Input.Schema, "seats"); got == beforeSeats {
		t.Fatalf("wire field type edit kept schema kind %s", got)
	}
	if after.ResponseSchema.Root == beforeRoot {
		t.Fatal("result rename kept the response schema root")
	}
	if after.ResponseSchema.Root != after.Returns.Identity() {
		t.Fatalf("response root = %s, want result %s", after.ResponseSchema.Root, after.Returns.Identity())
	}
}

func actionSchemaNodeKind(t *testing.T, schema types.CodecSchema, field string) string {
	t.Helper()
	nodes := map[string]types.CodecNode{}
	for _, node := range schema.Nodes {
		nodes[node.Identity] = node
	}
	root, ok := nodes[schema.Root]
	if !ok {
		t.Fatalf("schema root %s missing", schema.Root)
	}
	for _, entry := range root.Fields {
		if entry.Name == field {
			node, ok := nodes[entry.Type]
			if !ok {
				t.Fatalf("schema field %s points at missing node %s", field, entry.Type)
			}
			return string(node.Kind) + ":" + node.Name
		}
	}
	t.Fatalf("schema root has no field %s", field)
	return ""
}

func TestActionJSONResponseDiagnosticSpan(t *testing.T) {
	text := actionJSONHeader("seal_invoice, seal_wire, sealed, seal_failed, seal_outcome", "bytes", strings.Replace(actionJSONSealedDomain+actionJSONSealedAction, "record seal_failed\n    str reason\n", "record seal_failed\n    str reason\n    bytes::buffer payload\n", 1))
	_, err := programFixture(t, map[string]string{"src/web/web.can": text})
	if err == nil {
		t.Fatal("opaque result leaf admitted")
	}
	located, ok := source.AsLocated(err)
	if !ok {
		t.Fatalf("response failure lost its span: %v", err)
	}
	if !strings.HasSuffix(located.File, "web.can") {
		t.Fatalf("response span file = %s", located.File)
	}
	if got := text[located.Span.Start:located.Span.End]; got != "seal_outcome" {
		t.Fatalf("response span covers %q", got)
	}
}
