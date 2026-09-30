package project

import (
	"errors"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func jsonFindings(t *testing.T, err error) []*JSONError {
	t.Helper()
	if err == nil {
		return nil
	}
	if many, ok := err.(interface{ Unwrap() []error }); ok {
		var findings []*JSONError
		for _, child := range many.Unwrap() {
			findings = append(findings, jsonFindings(t, child)...)
		}
		return findings
	}
	located, ok := err.(*JSONError)
	if !ok {
		t.Fatalf("configuration error lost JSON provenance: %T %v", err, err)
	}
	return []*JSONError{located}
}

func jsonTokens(t *testing.T, data string, err error) []string {
	t.Helper()
	var tokens []string
	for _, issue := range jsonFindings(t, err) {
		if issue.Span.Start < 0 || issue.Span.End < issue.Span.Start || issue.Span.End > len(data) {
			t.Fatalf("bad JSON span: %+v", issue)
		}
		tokens = append(tokens, data[issue.Span.Start:issue.Span.End])
	}
	return tokens
}

func TestConfigManifestIndependentExactRanges(t *testing.T) {
	data := `{
 "source_root":"../src", "error_registry":3,
 "dependencies":{"bad-key":"vendor", "good":"../missing"},
 "assets":{"image":"../bad"}, "extra":true,
 "sql":{"query":{"dialect":"oracle","statement":"SELECT 1","parameters":["param","param"],"parameter_type":"row","row_type":"app::row","cardinality":"nope","row_limit_parameter":4}}
}`
	manifest, err := ParseManifest([]byte(data))
	tokens := jsonTokens(t, data, err)
	want := []string{`"../src"`, `3`, `"bad-key"`, `"../missing"`, `"../bad"`, `"extra"`, `"oracle"`, `"param"`, `"row"`, `"nope"`}
	if len(tokens) != len(want) {
		t.Fatalf("findings = %v; want %v", tokens, want)
	}
	for _, token := range want {
		found := false
		for _, got := range tokens {
			found = found || got == token
		}
		if !found {
			t.Errorf("missing exact range %s in %v", token, tokens)
		}
	}
	if manifest.Invalid["source_root"] == nil || manifest.InvalidDependencies["good"] == nil || manifest.InvalidSQL["query"] == nil {
		t.Fatalf("invalid component evidence missing: %+v", manifest)
	}
	for _, issue := range jsonFindings(t, err) {
		if data[issue.Span.Start:issue.Span.End] == `"param"` {
			if issue.Span.Start != strings.LastIndex(data, `"param"`) || len(issue.Related) != 1 || issue.Related[0].Start != strings.Index(data, `"param"`) {
				t.Fatalf("duplicate parameter origin: %+v", issue)
			}
		}
	}
}

func TestConfigNestedLockAndRegistryRanges(t *testing.T) {
	data := `{"edges":{"bad-key":{"target":"can.project.root","path":"x"},"valid":{"target":"wrong","path":"../outside"}},"projects":{"can.project.lineage/shared":{"lineage":"Bad","manifest_sha256":"M","source_sha256":"S","fixtures_sha256":"F","error_registry":{"active":["bad",7,"pkg::failure"],"retired":["pkg::failure"]},"edges":{"child":{"target":true,"path":"param"}}}}}`
	_, err := ParseLock([]byte(data))
	tokens := jsonTokens(t, data, err)
	for _, want := range []string{`"bad-key"`, `"wrong"`, `"../outside"`, `"Bad"`, `"M"`, `"S"`, `"F"`, `"bad"`, `7`, `"pkg::failure"`, `true`} {
		found := false
		for _, got := range tokens {
			found = found || got == want
		}
		if !found {
			t.Errorf("missing %s in %v", want, tokens)
		}
	}
	for _, issue := range jsonFindings(t, err) {
		if data[issue.Span.Start:issue.Span.End] == `"pkg::failure"` && issue.Span.Start != strings.LastIndex(data, `"pkg::failure"`) {
			t.Errorf("retired conflict points at active declaration: %+v", issue)
		}
	}
}

func TestConfigMissingNestedFieldInsertion(t *testing.T) {
	data := `{"edges":{"shared":{"target":"can.project.lineage/shared"}},"projects":{}}`
	_, err := ParseLock([]byte(data))
	findings := jsonFindings(t, err)
	if len(findings) != 1 {
		t.Fatalf("missing field cascaded: %v", err)
	}
	want := strings.Index(data, `"}}`) + 1
	if findings[0].Span != (source.Span{Start: want, End: want}) {
		t.Fatalf("missing nested field at %+v; want insertion %d", findings[0].Span, want)
	}
}

func TestConfigDuplicateKeysAndEscapeSpans(t *testing.T) {
	data := `{"source_root":"src","source\u005froot":"src2","error_registry":"e","error_registry":"f"}`
	_, err := ParseManifest([]byte(data))
	findings := jsonFindings(t, err)
	if len(findings) != 2 {
		t.Fatalf("independent duplicate keys: %v", err)
	}
	for _, finding := range findings {
		if len(finding.Related) != 1 {
			t.Fatalf("duplicate lacks origin: %+v", finding)
		}
	}
	malformed := `{"source_root":"\ud800","error_registry":"\udc00"}`
	_, err = ParseManifest([]byte(malformed))
	tokens := jsonTokens(t, malformed, err)
	if len(tokens) != 2 || tokens[0] != `\ud800` || tokens[1] != `\udc00` {
		t.Fatalf("escape ranges: %v", tokens)
	}
}

func TestConfigErrorsKeepRangesThroughSourceConversion(t *testing.T) {
	data := "{\r\n\"source_root\":\"😀\",\r\n\"error_registry\":null,\"unknown\":1}"
	_, err := ParseManifest([]byte(data))
	located := configError("/config/can.project.json", []byte(data), err)
	many, ok := located.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("independent config findings flattened: %v", located)
	}
	if len(many.Unwrap()) != 2 {
		t.Fatalf("findings: %v", located)
	}
	file, fileErr := source.New("/config/can.project.json", data)
	if fileErr != nil {
		t.Fatal(fileErr)
	}
	for _, child := range many.Unwrap() {
		var issue *source.LocatedError
		if !errors.As(child, &issue) || issue.File != file.Name() {
			t.Fatalf("wrong attribution: %v", child)
		}
		position, err := file.UTF16Position(issue.Span.Start)
		if err != nil || position.Line != 2 {
			t.Fatalf("wrong UTF16 position %+v %v", position, err)
		}
		token := data[issue.Span.Start:issue.Span.End]
		if token != `null` && token != `"unknown"` {
			t.Fatalf("widened config range: %q", token)
		}
	}
}

func TestConfigMalformedJSONUsesExactEOFAndIndependentScalarRanges(t *testing.T) {
	missing := `{"source_root":"src",`
	_, err := ParseManifest([]byte(missing))
	findings := jsonFindings(t, err)
	if len(findings) != 1 || findings[0].Span != (source.Span{Start: len(missing), End: len(missing)}) {
		t.Fatalf("missing syntax should point to EOF insertion: %v", err)
	}
	independent := `{"source_root":"\ud800","error_registry":"e","error_registry":"f"}`
	_, err = ParseManifest([]byte(independent))
	tokens := jsonTokens(t, independent, err)
	if len(tokens) != 2 || tokens[0] != `\ud800` || tokens[1] != `"error_registry"` {
		t.Fatalf("scalar error hid independent duplicate: %v", tokens)
	}
}

func TestConfigSQLIndependentCardinalityAndLimitTypes(t *testing.T) {
	data := `{"source_root":"src","error_registry":"e","sql":{"query":{"dialect":"sqlite","statement":"SELECT 1","parameters":[],"parameter_type":"app::params","row_type":"app::result","cardinality":"wrong","row_limit_parameter":"bad"}}}`
	_, err := ParseManifest([]byte(data))
	tokens := jsonTokens(t, data, err)
	if len(tokens) != 2 || tokens[0] != `"wrong"` || tokens[1] != `"bad"` {
		t.Fatalf("independent SQL type error hidden: %v", tokens)
	}
}
