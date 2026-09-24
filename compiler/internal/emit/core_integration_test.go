package emit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

// T09 (Gate 1/2 core contract integration): the integrated program lowers
// end to end. Vendor instances carrying an owner record, a closed variant
// and a scenario converge in one app through aliased imports; twin
// same-name owner instances lower as distinct identities.

func coreEmitProgram(t *testing.T) *check.Program {
	t.Helper()
	root := t.TempDir()
	mail := `package mail
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
`
	deps := map[string]struct {
		lineage  string
		files    map[string]string
		registry string
	}{
		"post":   {lineage: "shop_post", files: map[string]string{"mail.can": mail}},
		"rival":  {lineage: "shop_rival", files: map[string]string{"mail.can": mail}},
		"geo":    {lineage: "shop_geo", files: map[string]string{"shapes.can": "package shapes\n    provides [circle, rectangle, shape]\n    uses []\nrecord circle\n    int radius\nrecord rectangle\n    int width\n    int height\nvariant shape\n    circle\n    rectangle\n"}},
		"vendor": {lineage: "shop_vendor", files: map[string]string{"helper.can": scenarioEmitHelper}},
	}
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
	manifestEdges := map[string]any{}
	for _, edge := range []string{"geo", "post", "rival", "vendor"} {
		manifestEdges[edge] = "libs/" + edge
	}
	manifest, err := json.Marshal(map[string]any{"source_root": "src", "dependencies": manifestEdges, "error_registry": "can.errors.json"})
	if err != nil {
		t.Fatal(err)
	}
	write("can.project.json", string(manifest))
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/app/main.can", `package app
    provides []
    uses [post::mail, rival::mail as other, geo::shapes, vendor::helper as h]
fn str describe
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok "a@b"
    mail::email held = call mail::make_email(address)
    match held
        mail::email => ok call mail::email_address(held)
fn str describe_rival
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok "a@b"
    other::email held = call other::make_email(address)
    match held
        other::email => ok call other::email_address(held)
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
fn str read_customer
    emits []
    asserts
        customer: => ok "fixture" link h::checkout
    ok call h::read()
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    str first = call describe("a@b")
    str second = call describe_rival("a@b")
    match first is second
        false => ok
        true => ok
`)
	lockEdges := map[string]any{}
	lockProjects := map[string]any{}
	for _, edge := range []string{"geo", "post", "rival", "vendor"} {
		dep := deps[edge]
		dir := "libs/" + edge
		manifestText := `{"source_root":"src","project":"` + dep.lineage + `","error_registry":"can.errors.json"}`
		manifestData := []byte(manifestText)
		write(dir+"/can.project.json", manifestText)
		registry := dep.registry
		if registry == "" {
			registry = `{"active":[],"retired":[]}`
		}
		write(dir+"/can.errors.json", registry)
		var sources []project.SourceBytes
		for name, text := range dep.files {
			write(dir+"/src/"+name, text)
			sources = append(sources, project.SourceBytes{Path: name, Bytes: []byte(text)})
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
		id := "can.project.lineage/" + dep.lineage
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
		t.Fatal(err)
	}
	program, err := check.CheckProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func TestCoreIntegratedProgramLowers(t *testing.T) {
	program := coreEmitProgram(t)
	helperID := ""
	for _, fn := range program.Functions {
		if fn.Symbol.Package.Name == "helper" {
			helperID = fn.Symbol.Package.ID
		}
	}
	if helperID != "can.project.lineage/shop_vendor/helper" {
		t.Fatalf("helper instance identity: %q", helperID)
	}
	modules, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	assertions, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, artifact := range append(modules, assertions...) {
		joined += string(artifact.Bytes) + "\n"
	}
	if !strings.Contains(joined, "$canRecord(") {
		t.Fatal("integrated owner construction did not lower through the nominal record helper")
	}
	if !strings.Contains(joined, `"links":["`+helperID+`::checkout"]`) {
		t.Fatal("integrated scenario link lost its instance identity")
	}
	// Twin same-name owner factories lower as distinct lineage-qualified
	// identities. The variant-only dependency carries no functions, so
	// its types erase without a runtime lineage marker.
	for _, identity := range []string{
		"can.project.lineage/shop_post/mail::make_email",
		"can.project.lineage/shop_rival/mail::make_email",
		"can.project.lineage/shop_vendor/helper::read",
	} {
		if !strings.Contains(joined, identity) {
			t.Fatalf("integrated output omits identity %s", identity)
		}
	}
}
