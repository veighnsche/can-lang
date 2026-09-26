package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// actionDocumentDecls is one captured HTML read: typed int captures, a
// finite outcome variant with a truthful not-found leaf, and a document
// case per leaf. Callers append server, client or rejection functions.
const actionDocumentDecls = "package doc\n" +
	"    provides []\n" +
	"    uses [action, codec, html, http, sql]\n" +
	"record page_key\n" +
	"    int tenant_id\n" +
	"    int invoice_id\n" +
	"record page_found\n" +
	"    str title\n" +
	"record page_missing\n" +
	"    str reason\n" +
	"record page_unavailable\n" +
	"    str reason\n" +
	"variant page_outcome\n" +
	"    page_found\n" +
	"    page_missing\n" +
	"    page_unavailable\n" +
	"action read_invoice_page\n" +
	"    get \"/tenants/:tenant_id/invoices/:invoice_id/page\"\n" +
	"    captures page_key\n" +
	"    input none\n" +
	"    returns page_outcome\n" +
	"    body html\n" +
	"    cases\n" +
	"        page_found status 200 document\n" +
	"        page_missing status 404 document\n" +
	"        page_unavailable status 503 document\n"

// actionDocumentServer binds the read: a request-first handler capturing
// the startup pool, the single document renderer, and the route table.
const actionDocumentServer = "fn page_outcome load_page\n" +
	"    emits []\n" +
	"    given\n" +
	"        near sql::pool pool\n" +
	"        http::request req\n" +
	"        page_key key\n" +
	"    asserts\n" +
	"        sample: page_key(1, 7) => ok page_missing(\"gone\")\n" +
	"    match call http::request_headers(req)\n" +
	"        ok http::header[] headers => ok page_missing(\"gone\")\n" +
	"fn html::safe render_page\n" +
	"    emits []\n" +
	"    given\n" +
	"        page_outcome outcome\n" +
	"    asserts\n" +
	"        sample: page_missing(\"gone\") => ok\n" +
	"    match outcome\n" +
	"        page_found => ok call html::text_fragment(\"found\")\n" +
	"        page_missing => ok call html::text_fragment(\"missing\")\n" +
	"        page_unavailable => ok call html::text_fragment(\"unavailable\")\n" +
	"fn http::router routes\n" +
	"    emits [http::invalid_route, http::duplicate_route, http::ambiguous_route]\n" +
	"    given\n" +
	"        near sql::pool pool\n" +
	"    asserts\n" +
	"        sample: => ok\n" +
	"    match chain\n" +
	"        call action::mount(read_invoice_page, callable load_page, callable render_page) as http::route page\n" +
	"        call http::make_router([page]) as http::router built\n" +
	"        http::invalid_route\n" +
	"        http::duplicate_route\n" +
	"        http::ambiguous_route\n" +
	"        ok => ok built\n"

// actionDocumentURL builds canonical read paths from captures records.
const actionDocumentURL = "fn str page_url\n" +
	"    emits [action::invalid_path]\n" +
	"    given\n" +
	"        page_key key\n" +
	"    asserts\n" +
	"        sample: page_key(1, 7) => ok \"/tenants/1/invoices/7/page\"\n" +
	"    match call action::url(read_invoice_page, key)\n" +
	"        action::invalid_path\n" +
	"        ok str built => ok built\n"

const actionDocumentMount = "call action::mount(read_invoice_page, callable load_page, callable render_page)"

func actionDocumentProgram(t *testing.T, text string) *Program {
	t.Helper()
	program, err := programFixture(t, map[string]string{
		"src/doc/doc.can":  text,
		"src/app/main.can": "package app\n    provides []\n    uses []\n" + actionMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func TestActionDocumentReadChecks(t *testing.T) {
	read := "action read_line\n" +
		"    get \"/invoices/:invoice_id/lines/:line/page\"\n" +
		"    captures line_key\n" +
		"    input none\n" +
		"    returns load_outcome\n" +
		"    body html\n" +
		"    cases\n" +
		"        found status 200 document\n" +
		"        missing status 404 document\n" +
		"        unavailable status 503 document\n"
	program, err := programFixture(t, map[string]string{"src/web/web.can": actionWebFile(read)})
	if err != nil {
		t.Fatal(err)
	}
	action := actionByName(t, program, "read_line")
	if action.Method != "GET" || action.Path != "/invoices/:invoice_id/lines/:line/page" {
		t.Fatalf("read route = %s %s", action.Method, action.Path)
	}
	if action.Input.Mode != "none" || action.Input.Type != nil {
		t.Fatalf("document read gained a wire input: %+v", action.Input)
	}
	if action.Body != "html" {
		t.Fatalf("document read response body = %s", action.Body)
	}
	if len(action.Captures) != 2 || action.Captures[0].Name != "invoice_id" || action.Captures[1].Name != "line" {
		t.Fatalf("read captures = %+v", action.Captures)
	}
	if action.Captures[0].Type.Declaration() != "str" || action.Captures[1].Type.Declaration() != "int" {
		t.Fatal("read captures lost their str/int types")
	}
	want := []ActionCase{
		{Leaf: action.Symbol.Package.ID + "::found", Status: 200, Swap: "document"},
		{Leaf: action.Symbol.Package.ID + "::missing", Status: 404, Swap: "document"},
		{Leaf: action.Symbol.Package.ID + "::unavailable", Status: 503, Swap: "document"},
	}
	if len(action.Cases) != len(want) {
		t.Fatalf("cases = %+v", action.Cases)
	}
	for i, kase := range want {
		if action.Cases[i] != kase {
			t.Fatalf("cases = %+v, want %+v", action.Cases, want)
		}
	}
	if action.ResponseSchema != nil {
		t.Fatal("document read gained a JSON response schema")
	}
}

func TestActionDocumentMountSite(t *testing.T) {
	program := actionDocumentProgram(t, actionDocumentDecls+actionDocumentServer+actionDocumentURL)
	if len(program.Actions) != 1 {
		t.Fatalf("checked %d actions, want 1", len(program.Actions))
	}
	if !program.ActionRoutes || program.ActionClient {
		t.Fatalf("document state use = routes %v client %v, want routes only", program.ActionRoutes, program.ActionClient)
	}
	docID := program.Actions[0].Symbol.Package.ID
	steps := actionBindingSteps(t, program)
	sites := map[string][]ir.ActionSite{}
	contracts := map[string]*types.Type{}
	for _, step := range steps {
		if step.Action == nil {
			continue
		}
		sites[step.Identity] = append(sites[step.Identity], *step.Action)
		contracts[step.Identity] = step.Contract
		if len(step.Arguments) != len(step.Contract.Inputs()) {
			t.Fatalf("%s step lowers %d values for %d contract inputs", step.Identity, len(step.Arguments), len(step.Contract.Inputs()))
		}
	}
	if len(sites[actionMount]) != 1 || len(sites[actionURL]) != 1 {
		t.Fatalf("document sites = mount %d url %d", len(sites[actionMount]), len(sites[actionURL]))
	}
	mount := sites[actionMount][0]
	if mount.Method != "GET" || mount.InputMode != "none" || mount.Body != "html" {
		t.Fatalf("document mount site = %+v", mount)
	}
	if mount.Form != nil || mount.Response != nil || mount.Request != nil {
		t.Fatalf("document mount site gained a codec: %+v", mount)
	}
	if mount.Rejected != "" || mount.RawEntry != "" || mount.Issue != "" {
		t.Fatalf("document mount site gained structural identities: %+v", mount)
	}
	if mount.CapturesType != docID+"::page_key" || len(mount.Captures) != 2 {
		t.Fatalf("document mount captures = %+v", mount)
	}
	if len(mount.Cases) != 3 {
		t.Fatalf("document mount cases = %+v", mount.Cases)
	}
	leaves := map[string]bool{}
	for _, kase := range mount.Cases {
		if kase.Swap != "document" || kase.Leaf == "" || leaves[kase.Leaf] {
			t.Fatalf("document guard case = %+v", kase)
		}
		leaves[kase.Leaf] = true
	}
	contract := contracts[actionMount]
	if contract.Result().Declaration() != "can.std.http@1::route" || len(contract.Errors()) != 1 || contract.Errors()[0].Declaration() != "can.std.http@1::invalid_route" {
		t.Fatalf("document mount contract = %s emits %v", types.CanonicalName(contract.Result()), contract.Errors())
	}
	if len(contract.Inputs()) != 2 {
		t.Fatalf("document mount contract takes %d inputs, want handler plus document renderer", len(contract.Inputs()))
	}
	handler := contract.Inputs()[0]
	if handler.Kind() != types.Callable || len(handler.Errors()) != 0 {
		t.Fatalf("document handler input = %v", handler)
	}
	if len(handler.Inputs()) != 2 || handler.Inputs()[0].Declaration() != "can.std.http@1::request" || handler.Inputs()[1].Declaration() != docID+"::page_key" {
		t.Fatalf("document handler inputs = %v", handler.Inputs())
	}
	if handler.Result().Declaration() != docID+"::page_outcome" {
		t.Fatalf("document handler result = %s", handler.Result().Declaration())
	}
	render := contract.Inputs()[1]
	if render.Kind() != types.Callable || len(render.Errors()) != 0 {
		t.Fatalf("document renderer input = %v", render)
	}
	if len(render.Inputs()) != 1 || render.Inputs()[0].Declaration() != docID+"::page_outcome" {
		t.Fatalf("document renderer inputs = %v", render.Inputs())
	}
	if render.Result().Declaration() != "can.std.html@1::safe" {
		t.Fatalf("document renderer result = %s", render.Result().Declaration())
	}
	url := sites[actionURL][0]
	if url.Action != docID+"::read_invoice_page" || url.Method != "GET" || url.Path != "/tenants/:tenant_id/invoices/:invoice_id/page" {
		t.Fatalf("document url site = %+v", url)
	}
	if len(url.Captures) != 2 || url.Captures[0].Name != "tenant_id" || url.Captures[1].Name != "invoice_id" {
		t.Fatalf("document url captures = %+v", url.Captures)
	}
	urlContract := contracts[actionURL]
	if urlContract.Result().Declaration() != "str" || len(urlContract.Inputs()) != 1 || urlContract.Inputs()[0].Declaration() != docID+"::page_key" {
		t.Fatalf("document url contract inputs = %v", urlContract.Inputs())
	}
}

func TestActionDocumentBindingRejects(t *testing.T) {
	full := actionDocumentDecls + actionDocumentServer + actionDocumentURL
	fetch := func(call string) string {
		return "fn page_outcome fetch_page\n" +
			"    emits [http::transport_failed, http::invalid_request, http::body_limit, http::status_error, codec::invalid_data]\n" +
			"    given\n" +
			"        page_key key\n" +
			"    asserts\n" +
			"        sample: page_key(1, 7) => ok page_missing(\"gone\")\n" +
			"    match call " + call + "\n" +
			"        when\n" +
			"            sample: page_key(1, 7) => ok page_missing(\"gone\")\n" +
			"        http::transport_failed\n" +
			"        http::invalid_request\n" +
			"        http::body_limit\n" +
			"        http::status_error\n" +
			"        codec::invalid_data\n" +
			"        ok page_outcome got => ok got\n"
	}
	for name, tc := range map[string]struct {
		text string
		want string
	}{
		"document mount without renderer": {
			strings.Replace(full, actionDocumentMount, "call action::mount(read_invoice_page, callable load_page)", 1),
			"takes its handler plus the document renderer",
		},
		"document mount with structural renderer": {
			strings.Replace(full, actionDocumentMount, "call action::mount(read_invoice_page, callable load_page, callable render_page, callable render_page)", 1),
			"takes its handler plus the document renderer",
		},
		"document renderer mistyped": {
			strings.Replace(full, actionDocumentMount, "call action::mount(read_invoice_page, callable load_page, callable load_page)", 1),
			`document renderer "load_page" returns `,
		},
		"request on HTML GET": {
			actionDocumentDecls + fetch("action::request(read_invoice_page, key)"),
			`requires a bodyless JSON GET action, but "read_invoice_page" is HTML GET`,
		},
		"post on HTML GET": {
			actionDocumentDecls + fetch("action::post(read_invoice_page, key)"),
			`requires a JSON POST action, but "read_invoice_page" is HTML GET`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.text == full || tc.text == actionDocumentDecls {
				t.Fatal("invalid refusal fixture")
			}
			_, err := programFixture(t, map[string]string{
				"src/doc/doc.can":  tc.text,
				"src/app/main.can": "package app\n    provides []\n    uses []\n" + actionMain,
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("document diagnostic omits %q: %v", tc.want, err)
			}
		})
	}
}
