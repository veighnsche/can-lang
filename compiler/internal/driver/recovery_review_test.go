package driver

import (
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"strings"
	"testing"
)

const reviewHeader = "package app\n    provides []\n    uses []\n"

func reviewSnapshot(t *testing.T, text string, extra map[string]string) *Snapshot {
	t.Helper()
	files := map[string]string{"src/main.can": text}
	for path, text := range extra {
		files[path] = text
	}
	root := writeBridgeProject(t, files)
	snapshot, err := CheckSnapshot(root, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if program, err := check.CheckAssertionProgram(snapshot.Graph); err == nil || program != nil {
		t.Fatal("invalid source passed strict emission gate")
	}
	return snapshot
}
func reviewErrorTokens(t *testing.T, snapshot *Snapshot, tokens ...string) {
	t.Helper()
	got := map[string]int{}
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Severity != "error" {
			continue
		}
		text, ok := fileText(snapshot.Graph, diagnostic.File)
		if !ok {
			t.Errorf("missing source attribution: %+v", diagnostic)
			continue
		}
		file, _ := source.New(diagnostic.File, text)
		start, e1 := file.Offset(source.UTF16Position{Line: diagnostic.Line, Character: diagnostic.Start})
		end, e2 := file.Offset(source.UTF16Position{Line: diagnostic.EndLine, Character: diagnostic.End})
		if e1 != nil || e2 != nil {
			t.Fatalf("invalid range %+v", diagnostic)
		}
		got[text[start:end]]++
	}
	for _, token := range tokens {
		if got[token] == 0 {
			t.Errorf("missing exact %q: %+v", token, snapshot.Diagnostics)
		}
		delete(got, token)
	}
	if len(got) != 0 {
		t.Errorf("unexpected diagnostic ranges %+v: %+v", got, snapshot.Diagnostics)
	}
}
func TestRecoveryReviewDuplicateBodies(t *testing.T) {
	text := reviewHeader + `fn int duplicate
    emits {}
    asserts
        sample: => ok 1
    ok call missing_first()
fn int duplicate
    emits {}
    asserts
        sample: => ok 1
    ok call missing_second()
fn int dependent
    emits {}
    asserts
        sample: => ok 1
    ok call duplicate()
`
	snapshot := reviewSnapshot(t, text, nil)
	reviewErrorTokens(t, snapshot, "duplicate", "missing_first", "missing_second")
	if snapshot.Program == nil {
		t.Fatal("missing partial program")
	}
	for _, fn := range snapshot.Program.Functions {
		if fn.Region != nil {
			t.Fatalf("ambiguous or dependent IR retained: %s", fn.Identity())
		}
	}
}
func TestRecoveryReviewSignatureFields(t *testing.T) {
	text := reviewHeader + `fn missing_result broken
    emits {missing_error}
    given
        missing_first first
        missing_second second
    asserts
        sample: 1, 2 => ok 1
    ok 1
`
	snapshot := reviewSnapshot(t, text, nil)
	reviewErrorTokens(t, snapshot, "missing_result", "missing_error", "missing_first", "missing_second")
}
func TestRecoveryReviewNativeSignatureFields(t *testing.T) {
	text := strings.Replace(reviewHeader, "uses []", "uses [ai]", 1) + `connection classifier
    endpoint "http://localhost:1"
    timeout_ms 1000
choice missing_result broken from classifier
    emits {missing_error}
    given
        missing_first first
        missing_second second
    asks "Question"
        yes "Yes" => ok true
        no "No" => ok false
`
	snapshot := reviewSnapshot(t, text, nil)
	reviewErrorTokens(t, snapshot, "missing_result", "missing_error", "missing_first", "missing_second")
}
func TestRecoveryReviewConnectionFields(t *testing.T) {
	snapshot := reviewSnapshot(t, reviewHeader+`connection broken
    endpoint "invalid"
    timeout_ms 0
    headers
        host = "blocked"
        x_invalid = "bad\nvalue"
    metadata
        protocol "invalid_protocol"
        model ""
`, nil)
	reviewErrorTokens(t, snapshot, `"invalid"`, "0", "host", `"bad\nvalue"`, `"invalid_protocol"`, `""`)
}
func TestRecoveryReviewSQLFields(t *testing.T) {
	manifest := `{"source_root":"src","error_registry":"can.errors.json","sql":{"bad":{"dialect":"postgresql","statement":"SELECT 1 LIMIT $1","parameters":[],"parameter_type":"app::missing_param","row_type":"app::missing_row","cardinality":"one","row_limit_parameter":1}}}`
	snapshot := reviewSnapshot(t, reviewHeader+"record row\n    int value\n", map[string]string{"can.project.json": manifest})
	reviewErrorTokens(t, snapshot, `"app::missing_param"`, `"app::missing_row"`)
}
func TestRecoveryReviewFetchEntries(t *testing.T) {
	text := strings.Replace(reviewHeader, "uses []", "uses [http]", 1) + `connection service
    endpoint "http://localhost:1"
    timeout_ms 1000
fetch str broken from service
    emits {http::request_failed}
    get 42
    query
        first = 43
        second = 44
    headers
        content_type = "text/plain"
        content_type = "text/plain"
        host = "blocked"
        x_bad = "bad\nvalue"
        x_number = 45
`
	snapshot := reviewSnapshot(t, text, nil)
	reviewErrorTokens(t, snapshot, "42", "43", "44", "content_type", "host", `"bad\nvalue"`, "45")
	for _, native := range snapshot.Program.Natives {
		if native.Fetch != nil || len(native.Regions) != 0 {
			t.Fatal("failed fetch published native IR")
		}
	}
	for _, diagnostic := range snapshot.Diagnostics {
		if strings.Contains(diagnostic.Message, "duplicate request header") && len(diagnostic.Related) == 0 {
			t.Fatal("duplicate lacks first header origin")
		}
	}
}
func TestRecoveryReviewArgumentChildren(t *testing.T) {
	text := reviewHeader + `fn int add
    emits {}
    given
        int first
        int second
    asserts
        sample: 1, 2 => ok 3
    ok first + second
fn int broken
    emits {}
    asserts
        sample: => ok 3
    ok call add(call missing_first(), call missing_second())
`
	reviewErrorTokens(t, reviewSnapshot(t, text, nil), "missing_first", "missing_second")
}
func TestRecoveryReviewMatchArmChildren(t *testing.T) {
	text := reviewHeader + `fn int broken
    emits {}
    asserts
        sample: => ok 1
    match true
        false => ok call missing_first()
        true => ok call missing_second()
`
	reviewErrorTokens(t, reviewSnapshot(t, text, nil), "missing_first", "missing_second")
}
func TestRecoveryReviewSameBodyWarning(t *testing.T) {
	text := reviewHeader + `fn int broken
    emits {}
    asserts
        sample: => ok 1
    int damaged = call missing_first()
    int alias = 1
    ok alias
`
	snapshot := reviewSnapshot(t, text, nil)
	reviewErrorTokens(t, snapshot, "missing_first")
	warnings := 0
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Severity == "warning" {
			warnings++
		}
	}
	if warnings != 1 {
		t.Fatalf("independent alias warning lost: %+v", snapshot.Diagnostics)
	}
}
func TestRecoveryReviewAssertionAndNestedCallClosure(t *testing.T) {
	text := reviewHeader + `fn int broken
    emits {}
    asserts
        sample: => ok "wrong"
    ok 1
fn int dependent
    emits {}
    asserts
        sample: => ok 2
    ok 1 + call broken()
`
	snapshot := reviewSnapshot(t, text, nil)
	reviewErrorTokens(t, snapshot, `"wrong"`)
	for _, fn := range snapshot.Program.Functions {
		if fn.Region != nil {
			t.Fatalf("invalid declaration closure retained %s", fn.Identity())
		}
	}
}
func TestRecoveryReviewAbsentIndex(t *testing.T) {
	if NewBuilder("/root").Seal().Index != nil {
		t.Fatal("absent index masquerades as complete empty facts")
	}
}

func TestRecoveryReviewNativeAssertionClosure(t *testing.T) {
	text := strings.Replace(reviewHeader, "uses []", "uses [http]", 1) + `connection service
    endpoint "http://localhost:1"
    timeout_ms 1000
fetch str broken from service
    emits {http::request_failed}
    asserts
        sample: => ok 42
    get "/"
fn str dependent
    emits {http::request_failed}
    asserts
        sample: => ok "text"
    relay call broken()
`
	snapshot := reviewSnapshot(t, text, nil)
	reviewErrorTokens(t, snapshot, "42")
	for _, native := range snapshot.Program.Natives {
		if native.Fetch != nil || native.Symbol.Invalid == nil {
			t.Fatal("assertion-invalid native retains valid plan")
		}
	}
	for _, fn := range snapshot.Program.Functions {
		if fn.Region != nil {
			t.Fatal("caller of assertion-invalid native retains region")
		}
	}
	if len(snapshot.Program.Assertions) != 0 {
		t.Fatal("invalid declaration assertion roots retained")
	}
}
func TestRecoveryReviewNativeDependsOnBrokenFunction(t *testing.T) {
	text := strings.Replace(reviewHeader, "uses []", "uses [http]", 1) + `connection service
    endpoint "http://localhost:1"
    timeout_ms 1000
fn str path
    emits {}
    asserts
        sample: => ok 42
    ok "/"
fetch str dependent from service
    emits {http::request_failed}
    asserts
        sample: => ok "text"
            using raw "response.json"
    get call path()
`
	raw := `{"schema":"can.native-fixture.v1","environment":{},"target":"can.project.root/app::dependent","exchange":{"request":{"method":"GET","url":"http://localhost:1/","headers":[],"body":{"bytes_base64":""}},"outcome":{"response":{"status":200,"headers":[],"body_base64":"dGV4dA=="}}}}`
	snapshot := reviewSnapshot(t, text, map[string]string{"src/response.json": raw})
	reviewErrorTokens(t, snapshot, "42")
	for _, native := range snapshot.Program.Natives {
		if native.Fetch != nil || native.Symbol.Invalid == nil {
			t.Fatal("native caller of invalid function retains IR")
		}
	}
}
func TestRecoveryReviewCompletionArmChildren(t *testing.T) {
	text := strings.Replace(reviewHeader, "uses []", "uses [codec]", 1) + `fn int number
    emits {codec::invalid_data}
    asserts
        sample: => ok 1
    ok 1
fn int broken
    emits {}
    asserts
        sample: => ok 1
    match call number()
        codec::invalid_data => ok call missing_first()
        ok int value => ok call missing_second()
`
	reviewErrorTokens(t, reviewSnapshot(t, text, nil), "missing_first", "missing_second")
}

func TestRecoveryReviewConnectionDuplicateAndValue(t *testing.T) {
	snapshot := reviewSnapshot(t, reviewHeader+`connection broken
    endpoint "http://localhost:1"
    timeout_ms 1000
    headers
        x_probe = "ok"
        x_probe = "bad\nvalue"
`, nil)
	reviewErrorTokens(t, snapshot, "x_probe", `"bad\nvalue"`)
	for _, diagnostic := range snapshot.Diagnostics {
		if strings.Contains(diagnostic.Message, "duplicate header") && len(diagnostic.Related) == 0 {
			t.Fatal("duplicate origin missing")
		}
	}
}

func TestRecoveryReviewConnectionMetadataDuplicateAndValue(t *testing.T) {
	snapshot := reviewSnapshot(t, reviewHeader+`connection broken
    endpoint "http://localhost:1"
    timeout_ms 1000
    metadata
        protocol "typesafe_systemone_v1"
        protocol "bad"
`, nil)
	reviewErrorTokens(t, snapshot, "protocol", `"bad"`)
	for _, diagnostic := range snapshot.Diagnostics {
		if strings.Contains(diagnostic.Message, "duplicate metadata") && len(diagnostic.Related) == 0 {
			t.Fatal("duplicate metadata origin missing")
		}
	}
}
func TestRecoveryReviewNestedTypeChildren(t *testing.T) {
	cases := []struct {
		name, annotation string
		tokens           []string
	}{
		{"callable", "callable missing_result (missing_first, missing_second) emits {missing_error}", []string{"missing_result", "missing_first", "missing_second", "missing_error"}},
		{"generic", "pair<missing_first, missing_second>", []string{"missing_first", "missing_second"}},
		{"unknown generic head", "missing_head<missing_first, missing_second>", []string{"missing_head", "missing_first", "missing_second"}},
		{"nested", "pair<callable missing_result (missing_first) emits {}, missing_second[]>", []string{"missing_result", "missing_first", "missing_second"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := reviewHeader + `record pair<a, b>
    a first
    b second
record broken
    ` + tc.annotation + ` field
`
			reviewErrorTokens(t, reviewSnapshot(t, text, nil), tc.tokens...)
		})
	}
}

func TestRecoveryReviewMatchScrutineeChildren(t *testing.T) {
	text := reviewHeader + `fn int broken
    emits {}
    asserts
        sample: => ok 1
    match missing_first, missing_second
        _, _ => ok 1
`
	snapshot := reviewSnapshot(t, text, nil)
	reviewErrorTokens(t, snapshot, "missing_first", "missing_second")
	for _, fn := range snapshot.Program.Functions {
		if fn.Region != nil {
			t.Fatal("failed match scrutinees published region")
		}
	}
}

func TestRecoveryReviewAliasWarningToken(t *testing.T) {
	text := "\ufeff" + reviewHeader + `str greeting = "😀"
fn str value
    emits {}
    asserts
        sample: => ok "😀"
    str alias = greeting
    ok alias
`
	root := writeBridgeProject(t, map[string]string{"src/main.can": text})
	snapshot, err := CheckSnapshot(root, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	warnings := 0
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Severity != "warning" {
			t.Fatalf("unexpected diagnostic: %+v", diagnostic)
		}
		warnings++
		file, _ := source.New(diagnostic.File, text)
		start, e1 := file.Offset(source.UTF16Position{Line: diagnostic.Line, Character: diagnostic.Start})
		end, e2 := file.Offset(source.UTF16Position{Line: diagnostic.EndLine, Character: diagnostic.End})
		if e1 != nil || e2 != nil || text[start:end] != "alias" || diagnostic.Start != 8 || diagnostic.End != 13 {
			t.Fatalf("warning is not the indented alias token: %+v", diagnostic)
		}
	}
	if warnings != 1 {
		t.Fatalf("want one warning: %+v", snapshot.Diagnostics)
	}
	if program, err := check.CheckAssertionProgram(snapshot.Graph); err != nil || program == nil {
		t.Fatalf("warning rejected valid program: %v", err)
	}
}
