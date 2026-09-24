package check

import (
	"sort"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// fetchWebFile builds a web package carrying the shared JSON save/load
// actions plus browser fetch clients. Capture clients pass path captures
// in order; POST clients append the wire body last.
func fetchWebFile(extra ...string) string {
	clients := "fn load_outcome reload_line\n" +
		"    emits [http::transport_failed, http::invalid_request, http::status_error, codec::invalid_data]\n" +
		"    given\n" +
		"        str invoice_id\n" +
		"        int line\n" +
		"    asserts\n" +
		"        sample: \"inv-1\", 1 => ok found(\"inv-1\")\n" +
		"    match call http::fetch_json_get<load_outcome>(\"load_line\", invoice_id, line)\n" +
		"        http::transport_failed\n" +
		"        http::invalid_request\n" +
		"        http::status_error\n" +
		"        codec::invalid_data\n" +
		"        ok load_outcome got => ok got\n" +
		"fn save_outcome store_invoice\n" +
		"    emits [http::transport_failed, http::invalid_request, http::body_limit, http::status_error, codec::invalid_data]\n" +
		"    given\n" +
		"        invoice_wire body\n" +
		"    asserts\n" +
		"        sample: invoice_wire(\"inv-1\", 2) => ok saved(\"inv-1\")\n" +
		"    match call http::fetch_json_post<save_outcome, invoice_wire>(\"save_invoice\", body)\n" +
		"        http::transport_failed\n" +
		"        http::invalid_request\n" +
		"        http::body_limit\n" +
		"        http::status_error\n" +
		"        codec::invalid_data\n" +
		"        ok save_outcome done => ok done\n"
	decls := actionSaveDomain + actionSaveHandler + actionSaveAction + actionLoadDomain + actionLoadHandler + actionLoadAction + clients
	for _, text := range extra {
		decls += text
	}
	return "package web\n    provides [save_invoice, load_line, invoice_wire, saved, rejected, stale, denied, busy, save_outcome, found, missing, unavailable, load_outcome]\n    uses [http, codec]\n" + decls + actionMain
}

// fetchSites walks one checked region and returns every JSON fetch site.
func fetchSites(t *testing.T, region *ir.Region) []*ir.JSONFetchSite {
	t.Helper()
	var sites []*ir.JSONFetchSite
	var expression func(node *ir.Expression)
	var completion func(done *ir.Completion)
	var block func(node *ir.Block)
	var match func(selected *ir.Match)
	expression = func(node *ir.Expression) {
		if node == nil {
			return
		}
		if node.Invocation != nil {
			for i := range node.Invocation.Steps {
				if site := node.Invocation.Steps[i].JSONFetch; site != nil {
					sites = append(sites, site)
				}
			}
		}
		if node.Match != nil {
			match(node.Match)
		}
		for _, input := range node.Inputs {
			expression(input)
		}
	}
	match = func(selected *ir.Match) {
		for _, value := range selected.Values {
			expression(value)
		}
		if selected.Call != nil {
			for i := range selected.Call.Steps {
				if site := selected.Call.Steps[i].JSONFetch; site != nil {
					sites = append(sites, site)
				}
			}
		}
		for i := range selected.Arms {
			expression(selected.Arms[i].Value)
			completion(selected.Arms[i].Body)
		}
	}
	completion = func(done *ir.Completion) {
		if done == nil {
			return
		}
		expression(done.Value)
		if done.Call != nil {
			for i := range done.Call.Steps {
				if site := done.Call.Steps[i].JSONFetch; site != nil {
					sites = append(sites, site)
				}
			}
		}
		if done.Block != nil {
			block(done.Block)
		}
		if done.Match != nil {
			match(done.Match)
		}
	}
	block = func(node *ir.Block) {
		if node == nil {
			return
		}
		for i := range node.Steps {
			expression(node.Steps[i].Value)
			if node.Steps[i].Call != nil {
				for j := range node.Steps[i].Call.Steps {
					if site := node.Steps[i].Call.Steps[j].JSONFetch; site != nil {
						sites = append(sites, site)
					}
				}
			}
		}
		completion(node.Terminal)
	}
	block(region.Body)
	return sites
}

func fetchClientSites(t *testing.T, program *Program, name string) []*ir.JSONFetchSite {
	t.Helper()
	for _, fn := range program.Functions {
		if fn.Symbol.Name == name {
			return fetchSites(t, fn.Region)
		}
	}
	t.Fatalf("client %s missing from checked program", name)
	return nil
}

func TestFetchJSONClientsResolve(t *testing.T) {
	program, err := programFixture(t, map[string]string{"src/web/web.can": fetchWebFile()})
	if err != nil {
		t.Fatal(err)
	}
	webID := actionByName(t, program, "save_invoice").Symbol.Package.ID
	load := fetchClientSites(t, program, "reload_line")
	if len(load) != 1 {
		t.Fatalf("load client holds %d fetch sites", len(load))
	}
	site := load[0]
	if site.Action != webID+"::load_line" || site.Method != "GET" || site.Path != "/invoices/{invoice_id}/lines/{line}" {
		t.Fatalf("load site = %+v", site)
	}
	if len(site.Captures) != 2 || site.Captures[0].Name != "invoice_id" || site.Captures[0].Type != "str" || site.Captures[1].Name != "line" || site.Captures[1].Type != "int" {
		t.Fatalf("load captures = %+v", site.Captures)
	}
	if site.Request != nil {
		t.Fatal("bodyless GET site gained a request contract")
	}
	if site.Response.Root == "" {
		t.Fatal("load site carries no response schema")
	}
	if len(site.Cases) != 3 {
		t.Fatalf("load cases = %+v", site.Cases)
	}
	save := fetchClientSites(t, program, "store_invoice")
	if len(save) != 1 {
		t.Fatalf("save client holds %d fetch sites", len(save))
	}
	post := save[0]
	if post.Action != webID+"::save_invoice" || post.Method != "POST" || post.Path != "/invoices/save" {
		t.Fatalf("save site = %+v", post)
	}
	if len(post.Captures) != 0 {
		t.Fatalf("exact-path save gained captures: %+v", post.Captures)
	}
	if post.Request == nil || post.Request.Root == "" {
		t.Fatal("POST site carries no request contract")
	}
	if post.Response.Root == "" {
		t.Fatal("save site carries no response schema")
	}
	if len(post.Cases) != 5 {
		t.Fatalf("save cases = %+v", post.Cases)
	}
}

func TestFetchJSONFailureBound(t *testing.T) {
	program, err := programFixture(t, map[string]string{"src/web/web.can": fetchWebFile()})
	if err != nil {
		t.Fatal(err)
	}
	bound := func(operation string) []string {
		for key, special := range program.Fetches {
			if strings.HasPrefix(key, operation+"<") {
				var names []string
				for _, failure := range special.Contract.Errors() {
					names = append(names, failure.Declaration())
				}
				return names
			}
		}
		t.Fatalf("no specialization for %s", operation)
		return nil
	}
	get := bound(fetchJSONGet)
	wantGet := []string{"can.std.http@1::transport_failed", "can.std.http@1::invalid_request", "can.std.http@1::status_error", "can.std.codec@1::invalid_data"}
	sort.Strings(get)
	sort.Strings(wantGet)
	if strings.Join(get, ",") != strings.Join(wantGet, ",") {
		t.Fatalf("GET bound = %v, want %v", get, wantGet)
	}
	post := bound(fetchJSONPost)
	wantPost := []string{"can.std.http@1::transport_failed", "can.std.http@1::invalid_request", "can.std.http@1::body_limit", "can.std.http@1::status_error", "can.std.codec@1::invalid_data"}
	sort.Strings(post)
	sort.Strings(wantPost)
	if strings.Join(post, ",") != strings.Join(wantPost, ",") {
		t.Fatalf("POST bound = %v, want %v", post, wantPost)
	}
}

func TestFetchJSONCrossPackageName(t *testing.T) {
	web := "package web\n" +
		"    provides [save_invoice, load_line, invoice_wire, saved, rejected, stale, denied, busy, save_outcome, found, missing, unavailable, load_outcome]\n" +
		"    uses []\n" + actionSaveDomain + actionSaveHandler + actionSaveAction + actionLoadDomain + actionLoadHandler + actionLoadAction
	app := "package app\n" +
		"    provides []\n" +
		"    uses [web, http, codec]\n" +
		"fn web::load_outcome reload_line\n" +
		"    emits [http::transport_failed, http::invalid_request, http::status_error, codec::invalid_data]\n" +
		"    given\n" +
		"        str invoice_id\n" +
		"        int line\n" +
		"    asserts\n" +
		"        sample: \"inv-1\", 1 => ok web::found(\"inv-1\")\n" +
		"    match call http::fetch_json_get<web::load_outcome>(\"web::load_line\", invoice_id, line)\n" +
		"        http::transport_failed\n" +
		"        http::invalid_request\n" +
		"        http::status_error\n" +
		"        codec::invalid_data\n" +
		"        ok web::load_outcome got => ok got\n" + actionMain
	program, err := programFixture(t, map[string]string{"src/web/web.can": web, "src/app/main.can": app})
	if err != nil {
		t.Fatal(err)
	}
	sites := fetchClientSites(t, program, "reload_line")
	if len(sites) != 1 {
		t.Fatalf("cross-package client holds %d fetch sites", len(sites))
	}
	webID := actionByName(t, program, "load_line").Symbol.Package.ID
	if sites[0].Action != webID+"::load_line" {
		t.Fatalf("cross-package site action = %s", sites[0].Action)
	}
}

func TestFetchJSONRejects(t *testing.T) {
	base := fetchWebFile()
	cases := []struct {
		name string
		file string
		want string
	}{
		{
			"unknown action",
			strings.Replace(base, `"load_line", invoice_id, line`, `"nope", invoice_id, line`, 1),
			`unknown JSON action "nope"`,
		},
		{
			"renamed action diagnoses the fetch",
			strings.Replace(strings.Replace(base, "action load_line\n", "action load_row\n", 1), "provides [save_invoice, load_line,", "provides [save_invoice, load_row,", 1),
			`unknown JSON action "load_line"`,
		},
		{
			"get rejects a post action",
			strings.Replace(base, `http::fetch_json_get<load_outcome>("load_line", invoice_id, line)`, `http::fetch_json_get<save_outcome>("save_invoice")`, 1),
			`is not a bodyless GET action`,
		},
		{
			"post rejects a get action",
			strings.Replace(base, `http::fetch_json_post<save_outcome, invoice_wire>("save_invoice", body)`, `http::fetch_json_post<load_outcome, invoice_wire>("load_line", "inv-1", 1, body)`, 1),
			`is not a JSON POST action`,
		},
		{
			"result agreement",
			strings.Replace(base, `http::fetch_json_get<load_outcome>("load_line", invoice_id, line)`, `http::fetch_json_get<save_outcome>("load_line", invoice_id, line)`, 1),
			`result is`,
		},
		{
			"wire agreement",
			strings.Replace(base, `http::fetch_json_post<save_outcome, invoice_wire>("save_invoice", body)`, `http::fetch_json_post<save_outcome, found>("save_invoice", found("x"))`, 1),
			`wire is`,
		},
		{
			"non-literal name",
			strings.Replace(base, "    match call http::fetch_json_get<load_outcome>(\"load_line\", invoice_id, line)\n", "    str target = \"load_line\"\n    match call http::fetch_json_get<load_outcome>(target, invoice_id, line)\n", 1),
			`must be a static string literal`,
		},
		{
			"capture arity",
			strings.Replace(base, `"load_line", invoice_id, line`, `"load_line", invoice_id`, 1),
			`call arity mismatch`,
		},
		{
			"get carries no body slot",
			strings.Replace(base, `"load_line", invoice_id, line`, `"load_line", invoice_id, line, invoice_id`, 1),
			`call arity mismatch`,
		},
		{
			"post needs its body",
			strings.Replace(base, `"save_invoice", body`, `"save_invoice"`, 1),
			`call arity mismatch`,
		},
		{
			"capture type",
			strings.Replace(base, `"load_line", invoice_id, line`, `"load_line", line, line`, 1),
			`does not fit expected type`,
		},
		{
			"unknown package",
			strings.Replace(base, `"load_line", invoice_id, line`, `"shop::load_line", invoice_id, line`, 1),
			`names unknown package`,
		},
		{
			"missing type arguments",
			strings.Replace(base, `http::fetch_json_get<load_outcome>`, `http::fetch_json_get`, 1),
			``,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := programFixture(t, map[string]string{"src/web/web.can": tc.file})
			if err == nil {
				t.Fatal("bad fetch admitted")
			}
			if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("fetch diagnostic omits %q: %v", tc.want, err)
			}
		})
	}
}

func TestFetchJSONPostRejectsFormAction(t *testing.T) {
	form := "record line_wire\n" +
		"    str sku\n" +
		"record batch_wire\n" +
		"    str customer\n" +
		"    form::rows<line_wire> lines\n" +
		"record stored\n" +
		"    str label\n" +
		"variant store_outcome\n" +
		"    stored\n" +
		"fn store_outcome store_validated\n" +
		"    emits []\n" +
		"    given\n" +
		"        batch_wire body\n" +
		"    asserts\n" +
		"        sample: batch_wire(\"c\", form::rows<line_wire>([], [])) => ok stored(\"c\")\n" +
		"    ok stored(body.customer)\n" +
		"action append_line\n" +
		"    post \"/invoices/append\"\n" +
		"    body form batch_wire\n" +
		"    handles store_validated\n" +
		"    result store_outcome\n" +
		"    cases\n" +
		"        stored => 200\n"
	decls := actionSaveDomain + actionSaveHandler + actionSaveAction + actionLoadDomain + actionLoadHandler + actionLoadAction + form +
		"fn store_outcome push_batch\n" +
		"    emits [http::transport_failed, http::invalid_request, http::body_limit, http::status_error, codec::invalid_data]\n" +
		"    given\n" +
		"        batch_wire body\n" +
		"    asserts\n" +
		"        sample: batch_wire(\"c\", form::rows<line_wire>([], [])) => ok stored(\"c\")\n" +
		"    match call http::fetch_json_post<store_outcome, batch_wire>(\"append_line\", body)\n" +
		"        http::transport_failed\n" +
		"        http::invalid_request\n" +
		"        http::body_limit\n" +
		"        http::status_error\n" +
		"        codec::invalid_data\n" +
		"        ok store_outcome done => ok done\n"
	file := "package web\n    provides [save_invoice, load_line, append_line, invoice_wire, line_wire, batch_wire, saved, rejected, stale, denied, busy, save_outcome, found, missing, unavailable, load_outcome, stored, store_outcome]\n    uses [http, codec, form, option]\n" + decls + actionMain
	_, err := programFixture(t, map[string]string{"src/web/web.can": file})
	if err == nil {
		t.Fatal("form action admitted as a JSON fetch")
	}
	if !strings.Contains(err.Error(), "is not a JSON POST action") {
		t.Fatalf("form fetch diagnostic differs: %v", err)
	}
}
