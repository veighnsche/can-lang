package syntax

import (
	"strings"
	"testing"
)

// A03: captured GET document actions declare `body html` with per-case
// `document` modes next to `swap inner` fragments.
const actionReadDocumentSource = "action read_invoice\n" +
	"    get \"/tenants/:tenant_id/invoices/:invoice_id\"\n" +
	"    captures invoice_key\n" +
	"    input none\n" +
	"    returns read_outcome\n" +
	"    body html\n" +
	"    cases\n" +
	"        rendered status 200 document\n" +
	"        denied status 403 document\n"

func TestActionDocumentCasesParse(t *testing.T) {
	result := nativeParse(t, actionReadDocumentSource)
	if !result.OK() {
		t.Fatalf("document action rejected: %v", result.Diagnostics)
	}
	decl := result.File.Declarations[0].(*ActionDecl)
	if decl.Method.Text != "get" || decl.Response.Text != "html" || decl.Captures == nil {
		t.Fatalf("document action lost its GET/html/captures shape: %+v", decl)
	}
	if len(decl.Cases) != 2 {
		t.Fatalf("document action case table has %d rows", len(decl.Cases))
	}
	for _, kase := range decl.Cases {
		if kase.Document == nil || kase.Document.Text != "document" || kase.Swap != nil {
			t.Fatalf("case lost its document mode: %+v", kase)
		}
	}
}

func TestActionDocumentFormatRoundTrip(t *testing.T) {
	result := nativeParse(t, actionReadDocumentSource)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	formatted := Format(result.File)
	if !strings.Contains(formatted, actionReadDocumentSource) {
		t.Fatalf("formatted output rewrote the document action:\n%s", formatted)
	}
	reparsed := nativeParse(t, formatted[len(nativePackage):])
	if !reparsed.OK() {
		t.Fatalf("formatted document action does not reparse: %v", reparsed.Diagnostics)
	}
	if Format(reparsed.File) != formatted {
		t.Fatal("document action formatting is not stable")
	}
}

func TestActionDocumentMixedModes(t *testing.T) {
	text := "action read_mixed\n" +
		"    get \"/invoices/:invoice_id\"\n" +
		"    captures invoice_key\n" +
		"    input none\n" +
		"    returns read_outcome\n" +
		"    body html\n" +
		"    cases\n" +
		"        rendered status 200 document\n" +
		"        refreshed status 200 swap inner\n"
	result := nativeParse(t, text)
	if !result.OK() {
		t.Fatalf("mixed modes rejected: %v", result.Diagnostics)
	}
	decl := result.File.Declarations[0].(*ActionDecl)
	if decl.Cases[0].Document == nil || decl.Cases[1].Swap == nil {
		t.Fatalf("mixed modes lost: %+v", decl.Cases)
	}
	formatted := Format(result.File)
	if !strings.Contains(formatted, "rendered status 200 document") || !strings.Contains(formatted, "refreshed status 200 swap inner") {
		t.Fatalf("mixed modes moved:\n%s", formatted)
	}
}

func TestActionDocumentParsingRejects(t *testing.T) {
	head := "action read_invoice\n" +
		"    get \"/invoices/:invoice_id\"\n" +
		"    captures invoice_key\n" +
		"    input none\n" +
		"    returns read_outcome\n" +
		"    body html\n" +
		"    cases\n"
	for name, row := range map[string]string{
		"both modes":   "        rendered status 200 swap inner document\n",
		"reversed":     "        rendered status 200 document swap inner\n",
		"doubled":      "        rendered status 200 document document\n",
		"swap unknown": "        rendered status 200 swap outer\n",
		"bare swap":    "        rendered status 200 swap\n",
	} {
		t.Run(name, func(t *testing.T) {
			if result := nativeParse(t, head+row); result.OK() {
				t.Fatalf("invalid case accepted: %q", row)
			}
		})
	}
}
