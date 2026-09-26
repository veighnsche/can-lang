package check

import (
	"fmt"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// codeUnnecessaryLocal is the stable check-pipeline code for the demoted C8
// shape: a final local that merely aliases a simple value into the terminal.
// It is advisory only: programs carrying it check, emit, build and assert
// normally, and CLI exit codes stay 0 for warnings-only runs.
const codeUnnecessaryLocal = "CAN-CHECK-UNNECESSARY-LOCAL"

// Warning is one advisory check-pipeline finding: a canonical file, 1-based
// line/column, the pipeline's own code and a short message. Warnings never
// fail a build; drivers print them and editor bridges map them to warning
// severity. Positions are resolved at collection time so consumers never
// need the source bytes.
type Warning struct {
	Code    string
	File    string
	Line    int
	Column  int
	Span    source.Span
	Message string
}

// Format renders the CLI-identical diagnostic line:
// file:line:col: CODE: message.
func (w Warning) Format() string {
	return fmt.Sprintf("%s:%d:%d: %s: %s", w.File, w.Line, w.Column, w.Code, w.Message)
}

// unnecessaryLocalWarning reports the C8 four-clause shape as an advisory
// warning. A nil Warn callback drops the finding; direct CheckRegion callers
// without a collector stay silent rather than failing.
func (c *regionChecker) unnecessaryLocalWarning(diagnostic *UnnecessaryLocal) {
	if c.context.Warn == nil || diagnostic == nil {
		return
	}
	line, column := 1, 1
	if c.context.File != nil {
		if position, err := c.context.File.Position(diagnostic.Span.Start); err == nil {
			line, column = position.Line, position.Column
		}
	}
	file := ""
	if c.context.File != nil {
		file = c.context.File.Name()
	}
	c.context.Warn(Warning{Code: codeUnnecessaryLocal, File: file, Line: line, Column: column, Span: diagnostic.Span, Message: fmt.Sprintf("accidental alias %s — consider inlining", diagnostic.Name)})
}

// collectWarnings appends uniquely and sorts for deterministic reports:
// generic symbolic bodies and provisional inference re-check regions that
// the concrete pass also checks, so duplicates collapse on identity.
func collectWarnings(into []Warning, warning Warning) []Warning {
	for _, prior := range into {
		if prior == warning {
			return into
		}
	}
	into = append(into, warning)
	sort.Slice(into, func(i, j int) bool {
		if into[i].File != into[j].File {
			return into[i].File < into[j].File
		}
		if into[i].Span.Start != into[j].Span.Start {
			return into[i].Span.Start < into[j].Span.Start
		}
		return into[i].Code < into[j].Code
	})
	return into
}
