package driver

import (
	"context"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRecoveryIndependentBodiesWarningAndStrictGate(t *testing.T) {
	text := `package app
    provides []
    uses []

fn int first
    emits {}
    asserts
        sample: => ok 1
    ok call missing_first()

fn int second
    emits {}
    asserts
        sample: => ok 2
    int broken = call missing_second()
    int other = call missing_third()
    ok broken

fn int warning
    emits {}
    asserts
        sample: => ok 1
    int alias = 1
    ok alias
`
	root := writeBridgeProject(t, map[string]string{"src/main.can": text})
	snapshot, err := CheckSnapshot(root, filepath.Join(root, "src/main.can"), nil)
	if err != nil {
		t.Fatal(err)
	}
	var errors, warnings int
	for _, d := range snapshot.Diagnostics {
		if d.Severity == "warning" {
			warnings++
		} else if d.Severity == "error" {
			errors++
			if !strings.Contains(d.Message, "missing_") {
				t.Errorf("cascade: %+v", d)
			}
			if d.End-d.Start != len("missing_first") && d.End-d.Start != len("missing_second") && d.End-d.Start != len("missing_third") {
				t.Errorf("imprecise callee: %+v", d)
			}
		}
	}
	if errors != 3 || warnings != 1 {
		t.Fatalf("want three independent errors plus warning; got %+v", snapshot.Diagnostics)
	}
	if program, err := check.CheckAssertionProgram(snapshot.Graph); err == nil || program != nil {
		t.Fatal("invalid program escaped strict gate")
	}
}

func TestRecoveryOverlayOnlyAndSyntaxSibling(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/good.can": bridgeMain, "src/bad.can": "package app\n    provides []\n    uses []\n\nfn int bad(\n\nrecord retained\n"})
	overlay := project.NewOverlay()
	path := filepath.Join(filepath.Dir(canonical(t, filepath.Join(root, "src/good.can"))), "new.can")
	if err := overlay.Set(path, 7, strings.Replace(bridgeSecond, "config.display_name", "call missing_new()", 1)); err != nil {
		t.Fatal(err)
	}
	snapshot, err := CheckSnapshot(root, path, overlay)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Graph == nil || snapshot.World == nil || snapshot.Analysis == nil {
		t.Fatal("partial facts missing")
	}
	found := false
	for _, d := range snapshot.Diagnostics {
		if d.File == path && strings.Contains(d.Message, "missing_new") {
			found = true
		}
	}
	if !found {
		t.Fatalf("new unsaved source not checked: %+v", snapshot.Diagnostics)
	}
}

func TestRecoveryCancelledSnapshot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if snapshot, err := CheckSnapshotContext(ctx, "unused", "", nil); err != context.Canceled || snapshot != nil {
		t.Fatalf("%+v %v", snapshot, err)
	}
}

func TestRecoveryMalformedDeclarationBlocksDependentCall(t *testing.T) {
	text := strings.Replace(bridgeMain, "fn int helper\n", "fn int helper(\n", 1)
	root := writeBridgeProject(t, map[string]string{"src/main.can": text})
	snapshot, err := CheckSnapshot(root, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Code == "CAN-CHECK" {
			t.Fatalf("dependent cascade: %+v", diagnostic)
		}
	}
}

func TestRecoveryExactIndependentFamilies(t *testing.T) {
	text := strings.Replace(bridgeMain, "ok call helper(seed)", "ok call helper(\"wrong\")", 1)
	text += `
fn int operator_bad
    emits {}
    asserts
        sample: => ok 1
    ok 1 + true

fn int member_bad
    emits {}
    asserts
        sample: => ok 1
    ok config.absent

fn int duplicated
    emits {}
    asserts
        sample: => ok 1
        sample: => ok 1
    ok 1

record bad_fields
    unknown_one first
    unknown_two second
    int duplicate
    int duplicate
`
	root := writeBridgeProject(t, map[string]string{"src/main.can": text})
	snapshot, err := CheckSnapshot(root, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{`"wrong"`: 0, "+": 0, "absent": 0, "sample": 0, "unknown_one": 0, "unknown_two": 0, "duplicate": 0}
	file, _ := source.New(filepath.Join(root, "src/main.can"), text)
	errors := 0
	for _, d := range snapshot.Diagnostics {
		if d.Severity != "error" {
			continue
		}
		errors++
		start, e1 := file.Offset(source.UTF16Position{Line: d.Line, Character: d.Start})
		end, e2 := file.Offset(source.UTF16Position{Line: d.EndLine, Character: d.End})
		if e1 != nil || e2 != nil {
			t.Fatalf("invalid range: %+v", d)
		}
		token := text[start:end]
		if _, ok := want[token]; !ok {
			t.Errorf("unexpected primary span %q: %+v", token, d)
		} else {
			want[token]++
		}
		if (token == "sample" || token == "duplicate") && len(d.Related) == 0 {
			t.Errorf("duplicate lacks original location: %+v", d)
		}
	}
	for token, count := range want {
		if count != 1 {
			t.Errorf("token %q diagnostic count %d; all %+v", token, count, snapshot.Diagnostics)
		}
	}
	if errors != len(want) {
		t.Errorf("want %d independent errors, got %d", len(want), errors)
	}
}

func TestRecoveryRegistryAndInitializers(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": `package app
    provides []
    uses []
int failed = "wrong"
int dependent = failed
int valid = 3
fn int broken
    emits {}
    asserts
        sample: => ok 1
    ok call unknown_function()
`})
	if err := os.WriteFile(filepath.Join(root, "can.errors.json"), []byte(`{"active":[`), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := CheckSnapshot(root, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Program == nil {
		t.Fatal("independent program facts lost")
	}
	if len(snapshot.Program.Initializers) != 1 || !strings.HasSuffix(snapshot.Program.Initializers[0].Identity, "::valid") {
		t.Fatalf("invalid initializer dependency leaked: %+v", snapshot.Program.Initializers)
	}
	found := false
	for _, d := range snapshot.Diagnostics {
		if strings.Contains(d.Message, "unknown_function") {
			found = true
		}
	}
	if !found {
		t.Fatalf("registry error hid body: %+v", snapshot.Diagnostics)
	}
}

func TestRecoverySpecializationRollbackAndDependentIR(t *testing.T) {
	header := "package app\n    provides []\n    uses []\n"
	generic := header + `fn item identity<item>
    emits {}
    given
        item value
    asserts
        number: 3 => ok 3
    ok value
`
	broken := header + `fn int broken
    emits {}
    asserts
        sample: => ok 1
    str text = call identity<str>("invalid request provenance")
    ok "wrong"
`
	healthy := header + `fn str healthy
    emits {}
    asserts
        sample: => ok "valid"
    ok call identity<str>("valid")
fn int dependent
    emits {}
    asserts
        sample: => ok 1
    ok call broken()
`
	root := writeBridgeProject(t, map[string]string{"src/a_bad.can": broken, "src/generic.can": generic, "src/z_good.can": healthy})
	snapshot, err := CheckSnapshot(root, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	var strInstances int
	for _, fn := range snapshot.Program.Functions {
		if fn.Symbol.Name == "identity" && len(fn.TypeArguments) == 1 && fn.TypeArguments[0].Declaration() == "str" {
			strInstances++
			if fn.Region == nil {
				t.Fatal("failed sibling poisoned healthy specialization")
			}
			for _, request := range fn.Requests {
				if strings.Contains(request, "a_bad.can") {
					t.Fatalf("failed unit request escaped rollback: %s", request)
				}
			}
		}
		if fn.Symbol.Name == "dependent" && fn.Region != nil {
			t.Fatal("dependent invalid executable region retained")
		}
		if fn.Symbol.Name == "healthy" && fn.Region == nil {
			t.Fatal("independent body lost")
		}
	}
	if strInstances != 1 {
		t.Fatalf("want one committed str instance, got %d; diagnostics %+v", strInstances, snapshot.Diagnostics)
	}
}

func TestRecoveryNativeAndSQLFamilies(t *testing.T) {
	text := `package app
    provides []
    uses [ai]
connection classifier
    endpoint "http://localhost:1"
    timeout_ms 1000
    metadata
        protocol "typesafe_systemone_v1"
        model "jev-latest"
choice bool first from classifier
    emits {ai::invalid_question, ai::invalid_answer}
    asks 42
        yes "Yes" => ok true
        no "No" => ok false
choice bool second from classifier
    emits {ai::invalid_question, ai::invalid_answer}
    asks 43
        yes "Yes" => ok true
        no "No" => ok false
record row
    int id
fn int independent
    emits {}
    asserts
        sample: => ok 1
    ok call missing_body()
`
	root := writeBridgeProject(t, map[string]string{"src/main.can": text})
	manifest := `{"source_root":"src","error_registry":"can.errors.json","sql":{"first":{"dialect":"postgresql","statement":"SELECT id FROM rows LIMIT $1","parameters":[],"parameter_type":"app::missing_first","row_type":"app::row","cardinality":"one","row_limit_parameter":1},"second":{"dialect":"postgresql","statement":"SELECT id FROM rows LIMIT $1","parameters":[],"parameter_type":"app::missing_second","row_type":"app::row","cardinality":"one","row_limit_parameter":1}}}`
	if err := os.WriteFile(filepath.Join(root, "can.project.json"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := CheckSnapshot(root, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"42": 0, "43": 0, "missing_body": 0, `"app::missing_first"`: 0, `"app::missing_second"`: 0}
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Severity != "error" {
			continue
		}
		bytes, ok := fileText(snapshot.Graph, diagnostic.File)
		if !ok {
			t.Fatalf("spanless direct family diagnostic: %+v", diagnostic)
		}
		file, _ := source.New(diagnostic.File, bytes)
		start, e1 := file.Offset(source.UTF16Position{Line: diagnostic.Line, Character: diagnostic.Start})
		end, e2 := file.Offset(source.UTF16Position{Line: diagnostic.EndLine, Character: diagnostic.End})
		if e1 != nil || e2 != nil {
			t.Fatalf("invalid exact range: %+v", diagnostic)
		}
		token := bytes[start:end]
		if _, ok := want[token]; !ok {
			t.Errorf("unexpected family primary %q: %+v", token, diagnostic)
		} else {
			want[token]++
		}
	}
	for token, count := range want {
		if count != 1 {
			t.Errorf("token %q count%d; diagnostics %+v", token, count, snapshot.Diagnostics)
		}
	}
}

func TestRecoveryMessagesStableAfterPrecedingEdit(t *testing.T) {
	original := strings.Replace(bridgeMain, "ok call helper(seed)", "ok call missing_stable(seed)", 1) + "\nint invalid_initializer = call helper(1)\n"
	original = strings.Replace(original, "ok seed\n", "ok \"wrong\"\n", 1)
	root := writeBridgeProject(t, map[string]string{"src/main.can": original})
	path := canonical(t, filepath.Join(root, "src/main.can"))
	before, err := CheckSnapshot(root, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	overlay := project.NewOverlay()
	edited := "// shifted source coordinates\n" + original
	if err := overlay.Set(path, 2, edited); err != nil {
		t.Fatal(err)
	}
	after, err := CheckSnapshot(root, path, overlay)
	if err != nil {
		t.Fatal(err)
	}
	first := map[string]int{}
	second := map[string]int{}
	for _, d := range before.Diagnostics {
		if d.Severity == "error" {
			first[d.Message]++
		}
	}
	for _, d := range after.Diagnostics {
		if d.Severity == "error" {
			second[d.Message]++
		}
	}
	if len(first) != 3 || !reflect.DeepEqual(first, second) {
		t.Fatalf("semantic messages changed after positional edit: before=%v after=%v", first, second)
	}
}

func TestRecoveryCallableInitializerKeywordSpan(t *testing.T) {
	text := bridgeMain + "\ncallable int (int) emits {} stored = callable helper\n"
	root := writeBridgeProject(t, map[string]string{"src/main.can": text})
	path := canonical(t, filepath.Join(root, "src/main.can"))
	overlay := project.NewOverlay()
	var message string
	for version, current := range []string{text, strings.ReplaceAll(text, "helper", "renamed_helper")} {
		if err := overlay.Set(path, int64(version+1), current); err != nil {
			t.Fatal(err)
		}
		snapshot, err := CheckSnapshot(root, path, overlay)
		if err != nil {
			t.Fatal(err)
		}
		if len(snapshot.Diagnostics) != 1 {
			t.Fatalf("unexpected findings: %+v", snapshot.Diagnostics)
		}
		diagnostic := snapshot.Diagnostics[0]
		file, _ := source.New(path, current)
		start, err := file.Offset(source.UTF16Position{Line: diagnostic.Line, Character: diagnostic.Start})
		if err != nil {
			t.Fatal(err)
		}
		end, err := file.Offset(source.UTF16Position{Line: diagnostic.EndLine, Character: diagnostic.End})
		if err != nil {
			t.Fatal(err)
		}
		if current[start:end] != "callable" || start != strings.LastIndex(current, "callable ") {
			t.Fatalf("forbidden form highlights callee: %+v (%q)", diagnostic, current[start:end])
		}
		if version == 0 {
			message = diagnostic.Message
		} else if diagnostic.Message != message {
			t.Fatalf("renaming callee changed unrelated semantic error: %q / %q", message, diagnostic.Message)
		}
		if program, err := check.CheckAssertionProgram(snapshot.Graph); err == nil || program != nil {
			t.Fatal("forbidden callable initializer passed strict check")
		}
	}
}
