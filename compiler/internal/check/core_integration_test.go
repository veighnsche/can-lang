package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// T09 (Gate 1/2 core contract integration): T02-T08 converge on one locked
// revision. Per-lane suites cannot see the seams below, so each test stages
// real vendor dependency instances and exercises one named cross-feature
// case: an owner record through a generic/package boundary, a variant leaf
// across imports, a fixture link after an alias rename, and an error report
// after a lineage change.

// coreDep stages one locked vendor dependency: edge-named directory,
// optional lineage identity, source files relative to its src root, and
// the exact error registry text the lock pins.
type coreDep struct {
	lineage  string
	files    map[string]string
	registry string
}

// coreFixture stages a root project plus locked vendor dependencies and
// returns the loaded graph with the checked program.
func coreFixture(t *testing.T, main map[string]string, deps map[string]coreDep) (*project.Graph, *Program, error) {
	t.Helper()
	root := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	edges := map[string]any{}
	for _, edge := range sortedCoreKeys(deps) {
		edges[edge] = "libs/" + edge
	}
	manifest, err := json.Marshal(map[string]any{"source_root": "src", "dependencies": edges, "error_registry": "can.errors.json"})
	if err != nil {
		t.Fatal(err)
	}
	write("can.project.json", string(manifest))
	write("can.errors.json", `{"active":[],"retired":[]}`)
	for name, text := range main {
		write(name, text)
	}
	lockEdges := map[string]any{}
	lockProjects := map[string]any{}
	for _, edge := range sortedCoreKeys(deps) {
		dep := deps[edge]
		dir := "libs/" + edge
		manifestText := `{"source_root":"src","error_registry":"can.errors.json"}`
		id := "can.project.dependency/" + edge
		if dep.lineage != "" {
			manifestText = `{"source_root":"src","project":"` + dep.lineage + `","error_registry":"can.errors.json"}`
			id = "can.project.lineage/" + dep.lineage
		}
		manifestData := []byte(manifestText)
		write(dir+"/can.project.json", manifestText)
		registry := dep.registry
		if registry == "" {
			registry = `{"active":[],"retired":[]}`
		}
		write(dir+"/can.errors.json", registry)
		var sources []project.SourceBytes
		for _, name := range sortedCoreKeys(dep.files) {
			write(dir+"/src/"+name, dep.files[name])
			sources = append(sources, project.SourceBytes{Path: name, Bytes: []byte(dep.files[name])})
		}
		parsed, err := project.ParseRegistry([]byte(registry))
		if err != nil {
			t.Fatal(err)
		}
		digest, err := project.SourceDigest(sources)
		if err != nil {
			t.Fatal(err)
		}
		fixtureDigest, err := project.FixtureDigest(nil)
		if err != nil {
			t.Fatal(err)
		}
		lockEdges[edge] = map[string]any{"target": id, "path": dir}
		lockProjects[id] = map[string]any{"lineage": dep.lineage, "manifest_sha256": project.Digest(manifestData), "source_sha256": digest, "fixtures_sha256": fixtureDigest, "error_registry": parsed, "edges": map[string]any{}}
	}
	lock, err := json.Marshal(map[string]any{"edges": lockEdges, "projects": lockProjects})
	if err != nil {
		t.Fatal(err)
	}
	write("can.lock.json", string(lock))
	graph, err := project.Load(root)
	if err != nil {
		return nil, nil, err
	}
	program, err := CheckProgram(graph)
	if err != nil {
		return graph, nil, err
	}
	return graph, program, nil
}

func sortedCoreKeys[V any](in map[string]V) []string {
	out := make([]string, 0, len(in))
	for key := range in {
		out = append(out, key)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// assertCoreConsumerSpan pins a cross-feature rejection to the offending
// consumer file. Structured spans carry a Located error; region-level
// diagnostics name the consumer path in the message instead.
func assertCoreConsumerSpan(t *testing.T, name string, err error) {
	t.Helper()
	if located, ok := source.AsLocated(err); ok {
		if !strings.HasSuffix(located.File, "src/app/main.can") {
			t.Fatalf("%s located at %q, want the consumer file", name, located.File)
		}
		return
	}
	if !strings.Contains(err.Error(), "src/app/main.can") {
		t.Fatalf("%s diagnosis names no consumer span: %v", name, err)
	}
}

func coreMain() string {
	return "fn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n"
}

// coreMailDep is a vendor mail package: the owner record plus its factory
// and projection, with no catalogue imports.
func coreMailDep() coreDep {
	return coreDep{files: map[string]string{"mail.can": `package mail
    provides [email, make_email, email_address]
    uses []
owner record email
    str address
fn email make_email
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok email("a@b")
    ok email(address)
fn str email_address
    emits []
    given
        email mail
    asserts
        sample: email("a@b") => ok "a@b"
    ok mail.address
`}}
}

func coreBoxDep() coreDep {
	return coreDep{files: map[string]string{"lib.can": `package lib
    provides [box, wrap]
    uses []
record box<item>
    item value
fn box<item> wrap<item>
    emits []
    given
        item value
    asserts
        sample: 1 => ok box(1)
    ok box(value)
`}}
}

func TestCoreOwnerRecordThroughGenericPackageBoundary(t *testing.T) {
	deps := map[string]coreDep{"post": coreMailDep(), "tools": coreBoxDep(), "rival": coreMailDep()}
	deps["post"] = coreDep{lineage: "shop_post", files: deps["post"].files}
	deps["tools"] = coreDep{lineage: "shop_tools", files: deps["tools"].files}
	deps["rival"] = coreDep{lineage: "shop_rival", files: deps["rival"].files}
	header := "package app\n    provides []\n    uses [post::mail, tools::lib, rival::mail as other, codec, bytes]\n"
	positive := header + `fn str describe
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok "a@b"
    mail::email held = call mail::make_email(address)
    lib::box<mail::email> cell = call lib::wrap(held)
    match held
        mail::email => ok call mail::email_address(cell.value)
fn bool same_address
    emits []
    given
        str first
        str second
    asserts
        sample: "a@b", "a@b" => ok true
    mail::email one = call mail::make_email(first)
    mail::email two = call mail::make_email(second)
    ok one is two
` + coreMain()
	if _, _, err := coreFixture(t, map[string]string{"src/app/main.can": positive}, deps); err != nil {
		t.Fatalf("owner value through generic instance boundary rejected: %v", err)
	}
	cases := map[string]struct{ app, want string }{
		"generic codec encode": {`fn bytes::buffer expose
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok call bytes::from_utf8("x")
    mail::email held = call mail::make_email(address)
    lib::box<mail::email> cell = call lib::wrap(held)
    ok call codec::encode_json<lib::box<mail::email>>(cell)
`, "not codec-admissible"},
		"generic codec decode": {`fn lib::box<mail::email> load_owned
    emits [codec::invalid_data]
    given
        str text
    asserts
        sample: "{\"value\":{\"address\":\"a@b\"}}" => ok call lib::wrap(call mail::make_email("a@b"))
    match chain
        call bytes::from_utf8(text) as bytes::buffer encoded
        call codec::decode_json<lib::box<mail::email>>(encoded) as lib::box<mail::email> decoded
        codec::invalid_data
        ok => ok decoded
`, "not codec-admissible"},
		"field read through instantiation": {`fn str peek
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok "a@b"
    mail::email held = call mail::make_email(address)
    lib::box<mail::email> cell = call lib::wrap(held)
    mail::email inner = cell.value
    ok inner.address
`, "representation is confined"},
		"twin instance construction": {`fn other::email forge
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok call other::make_email("a@b")
    ok other::email(address)
`, "only be constructed in its declaring package"},
		"twin instance confusion": {`fn str smuggle
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok "a@b"
    ok call mail::email_address(call other::make_email(address))
`, "does not fit expected type"},
	}
	for name, tc := range cases {
		t.Run(strings.ReplaceAll(name, " ", "_"), func(t *testing.T) {
			_, _, err := coreFixture(t, map[string]string{"src/app/main.can": header + tc.app + coreMain()}, deps)
			if err == nil {
				t.Fatalf("owner/generic seam %s admitted", name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("owner/generic seam %s diagnosis %q lacks %q", name, err.Error(), tc.want)
			}
			assertCoreConsumerSpan(t, "owner/generic seam "+name, err)
		})
	}
}

func TestCoreVariantLeafAcrossImports(t *testing.T) {
	geo := coreDep{lineage: "shop_geo", files: map[string]string{"shapes.can": `package shapes
    provides [circle, rectangle, triangle, shape, unit, tagged]
    uses []
record circle
    int radius
record rectangle
    int width
    int height
record triangle
    int base
variant shape
    circle
    rectangle
record unit
variant tagged<item>
    unit
`}}
	twin := coreDep{lineage: "shop_geo2", files: map[string]string{"shapes.can": `package shapes
    provides [circle, shape]
    uses []
record circle
    int radius
variant shape
    circle
`}}
	deps := map[string]coreDep{"geo": geo, "geo2": twin}
	header := "package app\n    provides []\n    uses [geo::shapes, geo2::shapes as other]\n"
	positive := header + `fn shapes::shape convert
    emits []
    given
        shapes::circle value
    asserts
        sample: shapes::circle(3) => ok shapes::circle(3)
    ok value
fn int area_units
    emits []
    given
        shapes::shape value
    asserts
        round: shapes::circle(3) => ok 3
        square: shapes::rectangle(3, 4) => ok 3
    match value
        shapes::circle(bind radius) => ok radius
        shapes::rectangle(bind width, _) => ok width
fn shapes::tagged<str> rewrap
    emits []
    given
        shapes::tagged<int> value
    asserts
        sample: shapes::unit() => ok shapes::unit()
    ok value
` + coreMain()
	if _, _, err := coreFixture(t, map[string]string{"src/app/main.can": positive}, deps); err != nil {
		t.Fatalf("variant leaf across imports rejected: %v", err)
	}
	cases := map[string]struct{ app, want string }{
		"unlisted leaf": {`fn shapes::shape convert
    emits []
    given
        shapes::triangle value
    asserts
        sample: shapes::triangle(3) => ok shapes::circle(3)
    ok value
`, "does not fit expected type"},
		"twin instance leaf": {`fn shapes::shape convert
    emits []
    given
        other::circle value
    asserts
        sample: other::circle(3) => ok shapes::circle(3)
    ok value
`, "does not fit expected type"},
		"twin instance variant": {`fn other::shape convert
    emits []
    given
        shapes::shape value
    asserts
        sample: shapes::circle(3) => ok other::circle(3)
    ok value
`, "does not fit expected type"},
		"missing leaf arm": {`fn int area_units
    emits []
    given
        shapes::shape value
    asserts
        round: shapes::circle(3) => ok 3
    match value
        shapes::circle(bind radius) => ok radius
`, "ordinary match is not exhaustive"},
	}
	for name, tc := range cases {
		t.Run(strings.ReplaceAll(name, " ", "_"), func(t *testing.T) {
			_, _, err := coreFixture(t, map[string]string{"src/app/main.can": header + tc.app + coreMain()}, deps)
			if err == nil {
				t.Fatalf("variant/import seam %s admitted", name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("variant/import seam %s diagnosis %q lacks %q", name, err.Error(), tc.want)
			}
			assertCoreConsumerSpan(t, "variant/import seam "+name, err)
		})
	}
}

func TestCoreScenarioLinkAfterAliasRename(t *testing.T) {
	helper := `package helper
    provides [read, checkout]
    uses [text]
scenario checkout
fn str read
    emits []
    asserts
        unit: => ok "fixture"
    match call text::from_int(7)
        when
            scenario checkout: 7 => ok "fixture"
            unit: 7 => ok "fixture"
        ok str result => ok result
`
	deps := map[string]coreDep{"vendor": {lineage: "shop_vendor", files: map[string]string{"helper.can": helper}}}
	app := func(uses, link, call string) string {
		return "package app\n    provides []\n    uses [" + uses + "]\nfn str read_customer\n    emits []\n    asserts\n        customer: => ok \"fixture\"" + link + "\n    ok call " + call + "\n" + coreMain()
	}
	program, err := func() (*Program, error) {
		_, program, err := coreFixture(t, map[string]string{"src/app/main.can": app("vendor::helper as h", " link h::checkout", "h::read()")}, deps)
		return program, err
	}()
	if err != nil {
		t.Fatalf("scenario link through instance alias rejected: %v", err)
	}
	helperID := scenarioPackageID(t, program, "helper")
	if _, links := scenarioRootLinks(t, program, "customer"); len(links) != 1 || links[0] != helperID+"::checkout" {
		t.Fatalf("aliased instance link resolved to %v", links)
	}
	for name, tc := range map[string]struct{ uses, link, call, want string }{
		"renamed alias keeps stale link":   {"vendor::helper as hh", " link h::checkout", "hh::read()", "stale scenario link"},
		"pre-alias qualifier after rename": {"vendor::helper as h", " link helper::checkout", "h::read()", "stale scenario link"},
		"unknown scenario through alias":   {"vendor::helper as h", " link h::missing", "h::read()", "stale scenario link"},
	} {
		t.Run(strings.ReplaceAll(name, " ", "_"), func(t *testing.T) {
			_, _, err := coreFixture(t, map[string]string{"src/app/main.can": app(tc.uses, tc.link, tc.call)}, deps)
			if err == nil {
				t.Fatalf("scenario/alias seam %s admitted", name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("scenario/alias seam %s diagnosis %q lacks %q", name, err.Error(), tc.want)
			}
			assertCoreConsumerSpan(t, "scenario/alias seam "+name, err)
		})
	}
}

func TestCoreScenarioTwoLinksThroughInstanceAlias(t *testing.T) {
	helper := `package helper
    provides [read, checkout, retry]
    uses [text]
scenario checkout
scenario retry
fn str read
    emits []
    asserts
        unit: => ok "fixture"
    match call text::from_int(7)
        when
            scenario checkout: 7 => ok "first"
            scenario retry: 7 => ok "second"
        ok str result => ok result
`
	deps := map[string]coreDep{"vendor": {lineage: "shop_vendor", files: map[string]string{"helper.can": helper}}}
	program, err := func() (*Program, error) {
		_, program, err := coreFixture(t, map[string]string{"src/app/main.can": "package app\n    provides []\n    uses [vendor::helper as h]\nfn str read_customer\n    emits []\n    asserts\n        customer: => ok \"first\" link h::checkout, h::retry\n    ok call h::read()\n" + coreMain()}, deps)
		return program, err
	}()
	if err != nil {
		t.Fatal(err)
	}
	helperID := scenarioPackageID(t, program, "helper")
	_, links := scenarioRootLinks(t, program, "customer")
	if len(links) != 2 || links[0] != helperID+"::checkout" || links[1] != helperID+"::retry" {
		t.Fatalf("sequential instance links = %v", links)
	}
}

func coreTwinModelDep(value string) coreDep {
	return coreDep{files: map[string]string{"model.can": `package model
    provides [item, failed, risky]
    uses []
record item
    ` + value + `
error failed(str reason)
fn int risky
    emits [failed]
    given
        int value
    asserts
        sample: 0 => failed("bad")
    failed("bad")
`}, registry: `{"active":["model::failed"],"retired":[]}`}
}

func TestCoreErrorAcrossTwinInstances(t *testing.T) {
	deps := map[string]coreDep{
		"left":  {lineage: "shop_left", files: coreTwinModelDep("int value").files, registry: coreTwinModelDep("").registry},
		"right": {lineage: "shop_right", files: coreTwinModelDep("str label").files, registry: coreTwinModelDep("").registry},
	}
	header := "package app\n    provides []\n    uses [left::model as first, right::model as second]\n"
	positive := header + `fn int caller
    emits [first::failed, second::failed]
    asserts
        sample: => first::failed("bad")
    match call first::risky(0)
        first::failed
        ok int got => relay call second::risky(got)
` + coreMain()
	graph, _, err := coreFixture(t, map[string]string{"src/app/main.can": positive}, deps)
	if err != nil {
		t.Fatalf("twin error composition rejected: %v", err)
	}
	left, kind, ok := graph.ResolveErrorReport("can.error.v2:can.project.lineage/shop_left/model::failed")
	if !ok || left == nil || left.Lineage != "shop_left" || kind != "model::failed" {
		t.Fatalf("left report misattributed: %+v %q %v", left, kind, ok)
	}
	right, kind, ok := graph.ResolveErrorReport("can.error.v2:can.project.lineage/shop_right/model::failed")
	if !ok || right == nil || right.Lineage != "shop_right" || kind != "model::failed" {
		t.Fatalf("right report misattributed: %+v %q %v", right, kind, ok)
	}
	if left == right {
		t.Fatal("same-kind reports from twin instances share one owner")
	}
	cases := map[string]struct{ app, want string }{
		"undeclared twin escape": {`fn int caller
    emits [first::failed]
    asserts
        sample: => first::failed("bad")
    match call first::risky(0)
        first::failed
        ok int got => relay call second::risky(got)
`, "undeclared escaping domain error"},
		"wrong twin arm": {`fn int caller
    emits [second::failed]
    asserts
        sample: => second::failed("bad")
    match call second::risky(0)
        first::failed
        ok int got => ok got
`, "error arm is outside matched bound"},
	}
	for name, tc := range cases {
		t.Run(strings.ReplaceAll(name, " ", "_"), func(t *testing.T) {
			_, _, err := coreFixture(t, map[string]string{"src/app/main.can": header + tc.app + coreMain()}, deps)
			if err == nil {
				t.Fatalf("error/instance seam %s admitted", name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error/instance seam %s diagnosis %q lacks %q", name, err.Error(), tc.want)
			}
			assertCoreConsumerSpan(t, "error/instance seam "+name, err)
		})
	}
}

func TestCoreErrorReportAfterLineageChange(t *testing.T) {
	deps := map[string]coreDep{
		"left": {lineage: "shop_left", files: coreTwinModelDep("int value").files, registry: coreTwinModelDep("").registry},
	}
	renamedUses := "package app\n    provides []\n    uses [left::model as alpha]\nrecord holder\n    alpha::item one\n" + coreMain()
	graph, _, err := coreFixture(t, map[string]string{"src/app/main.can": renamedUses}, deps)
	if err != nil {
		t.Fatal(err)
	}
	// An alias rename keeps the lineage-qualified report identity: the
	// local qualifier never leaks into the archived identity.
	owner, kind, ok := graph.ResolveErrorReport("can.error.v2:can.project.lineage/shop_left/model::failed")
	if !ok || owner == nil || owner.Lineage != "shop_left" || kind != "model::failed" {
		t.Fatalf("report misattributed after alias rename: %+v %q %v", owner, kind, ok)
	}
	moved := map[string]coreDep{
		"left": {lineage: "shop_left_v2", files: coreTwinModelDep("int value").files, registry: coreTwinModelDep("").registry},
	}
	relocated, _, err := coreFixture(t, map[string]string{"src/app/main.can": renamedUses}, moved)
	if err != nil {
		t.Fatal(err)
	}
	// A lineage change fails closed: the pre-change archived identity
	// resolves nowhere instead of silently following the renamed owner.
	if _, _, ok := relocated.ResolveErrorReport("can.error.v2:can.project.lineage/shop_left/model::failed"); ok {
		t.Fatal("pre-change report identity still resolves after lineage change")
	}
	owner, kind, ok = relocated.ResolveErrorReport("can.error.v2:can.project.lineage/shop_left_v2/model::failed")
	if !ok || owner == nil || owner.Lineage != "shop_left_v2" || kind != "model::failed" {
		t.Fatalf("post-change report misattributed: %+v %q %v", owner, kind, ok)
	}
}
