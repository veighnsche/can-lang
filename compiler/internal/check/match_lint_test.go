package check

import (
	"strings"
	"testing"
)

const stringLadderFunction = `fn bool allowed
    emits {}
    given
        str ext
        str other
    asserts
        sample: ".html", ".css" => ok true
`

func TestStringMatchLadderWarning(t *testing.T) {
	tests := []struct {
		name, body, condition string
	}{
		{"two decisions", `    match ext is ".html"
        true => ok true
        false => match ext is ".css"
            false => ok false
            true => ok true
`, `ext is ".html"`},
		{"maximal chain", `    match ext is ".html"
        false => match ext is ".css"
            false => match ext is ".js"
                false => match ext is ".mjs"
                    false => ok false
                    true => ok true
                true => ok true
            true => ok true
        true => ok true
`, `ext is ".html"`},
		{"grouped conditions", `    match (ext is ".html" or ext is ".css")
        true => ok true
        false => match ".png" is ext or ".jpg" is (ext)
            true => ok false
            false => ok false
`, `(ext is ".html" or ext is ".css")`},
		{"value match", `    bool allowed = match ext is ".html"
        true => true
        false => match ext is ".css"
            true => true
            false => false
    ok allowed
`, `ext is ".html"`},
		{"nested in success arm", `    match true
        false => ok false
        true => match ext is ".html"
            true => ok true
            false => match ext is ".css"
                true => ok true
                false => ok false
`, `ext is ".html"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			text := programHeader + stringLadderFunction + test.body + programMain + "    ok\n"
			program, err := programFixture(t, map[string]string{"src/main.can": text})
			if err != nil {
				t.Fatal(err)
			}
			var warnings []Warning
			for _, warning := range program.Warnings {
				if warning.Code == codeStringMatchLadder {
					warnings = append(warnings, warning)
				}
			}
			if len(warnings) != 1 {
				t.Fatalf("expected one advisory for the chain, got %+v", warnings)
			}
			warning := warnings[0]
			if warning.Severity != SeverityWarning || text[warning.Span.Start:warning.Span.End] != test.condition || warning.Span.Start != strings.Index(text, test.condition) {
				t.Fatalf("wrong severity or first-condition span: %+v", warning)
			}
			if !strings.Contains(warning.Message, "match ext") || !strings.Contains(warning.Message, "|") || !strings.Contains(warning.Message, "_") {
				t.Fatalf("missing actionable syntax: %+v", warning)
			}
		})
	}
}

func TestStringMatchLadderExclusions(t *testing.T) {
	tests := []struct{ name, body string }{
		{"single predicate", `    match ext is ".html"
        true => ok true
        false => ok false
`},
		{"different bindings", `    match ext is ".html"
        true => ok true
        false => match other is ".css"
            true => ok true
            false => ok false
`},
		{"mixed disjunction", `    match ext is ".html" or other is ".css"
        true => ok true
        false => match ext is ".js"
            true => ok true
            false => ok false
`},
		{"overlapping decoded cases", `    match ext is "\q"
        true => ok true
        false => match ext is "\\q"
            true => ok true
            false => ok false
`},
		{"overlap within group", `    match ext is ".html" or ext is ".html"
        true => ok true
        false => match ext is ".css"
            true => ok true
            false => ok false
`},
		{"negative predicate", `    match ext is not ".html"
        false => ok true
        true => match ext is ".css"
            true => ok true
            false => ok false
`},
		{"relational test", `    match ext < ".html"
        true => ok true
        false => match ext is ".css"
            true => ok true
            false => ok false
`},
		{"comparison chain", `    match ext is ".html" is ext
        true => ok true
        false => match ext is ".css"
            true => ok true
            false => ok false
`},
		{"intervening binding", `    match ext is ".html"
        true => ok true
        false => do
            str selected = other
            match ext is ".css"
                true => ok selected is ".css"
                false => ok false
`},
		{"flat matching", `    match ext
        ".html" | ".css" => ok true
        _ => ok false
`},
		{"integer subject", `    match ext.length is 1
        true => ok true
        false => match ext.length is 2
            true => ok true
            false => ok false
`},
		{"recomputed subject", `    match (call identity(ext)) is ".html"
        true => ok true
        false => match (call identity(ext)) is ".css"
            true => ok true
            false => ok false
`},
		{"projected subject", `    match [ext][0] is ".html"
        true => ok true
        false => match [ext][0] is ".css"
            true => ok true
            false => ok false
`},
	}
	identity := `fn str identity
    emits {}
    given
        str value
    asserts
        sample: ".html" => ok ".html"
    ok value
`
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			text := programHeader + identity + stringLadderFunction + test.body + programMain + "    ok\n"
			program, err := programFixture(t, map[string]string{"src/main.can": text})
			if err != nil {
				t.Fatal(err)
			}
			for _, warning := range program.Warnings {
				if warning.Code == codeStringMatchLadder {
					t.Fatalf("ordinary or unsafe logic received flattening advice: %+v", warning)
				}
			}
		})
	}
}
