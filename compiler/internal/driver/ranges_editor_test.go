package driver

import (
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

func TestIssueDiagnosticsPreserveMultilineSpan(t *testing.T) {
	text := "first\nsecond\n"
	file, err := source.New("multi.can", text)
	if err != nil {
		t.Fatal(err)
	}
	span := source.Span{Start: 2, End: 9} // "rst\nseco" crosses lines 0-1
	sourceErr := &project.SourceError{
		Path:        "multi.can",
		File:        file,
		Diagnostics: []syntax.Diagnostic{{Code: "syntax", Message: "crosses lines", Span: span}},
		Message:     "multi.can:1:3: syntax: crosses lines",
	}
	got := convertedSourceIssues(sourceErr, text)
	if len(got) != 1 {
		t.Fatalf("diagnostics = %+v", got)
	}
	diagnostic := got[0]
	if diagnostic.Line != 0 || diagnostic.EndLine != 1 {
		t.Fatalf("multiline widened to one line: %+v", diagnostic)
	}
	if diagnostic.Start != 2 || diagnostic.End != 3 {
		t.Fatalf("multiline columns wrong: %+v", diagnostic)
	}
}

func TestIssueDiagnosticsPreserveEOFInsertion(t *testing.T) {
	text := "ok\n"
	file, err := source.New("eof.can", text)
	if err != nil {
		t.Fatal(err)
	}
	span := source.Span{Start: len(text), End: len(text)}
	sourceErr := &project.SourceError{
		Path:        "eof.can",
		File:        file,
		Diagnostics: []syntax.Diagnostic{{Code: "syntax", Message: "missing end", Span: span}},
		Message:     "eof.can:2:1: syntax: missing end",
	}
	got := convertedSourceIssues(sourceErr, text)
	if len(got) != 1 {
		t.Fatalf("diagnostics = %+v", got)
	}
	diagnostic := got[0]
	if diagnostic.Line != 1 || diagnostic.EndLine != 1 || diagnostic.Start != 0 || diagnostic.End != 0 {
		t.Fatalf("eof insertion not preserved: %+v", diagnostic)
	}
}

func convertedSourceIssues(err *project.SourceError, text string) []Diagnostic {
	graph := &project.Graph{Inputs: map[string][]byte{err.Path: []byte(text)}}
	var result []Diagnostic
	for _, issue := range errorIssues(err) {
		result = append(result, issueDiagnostic(graph, issue))
	}
	return result
}
