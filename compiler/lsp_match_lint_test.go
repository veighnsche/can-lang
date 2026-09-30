package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLSPStringMatchLadderWarning(t *testing.T) {
	const prefix = `package app
    provides [allowed]
    uses []
fn bool allowed
    emits {}
    given
        str ext
    asserts
        html: ".html" => ok true
`
	const ladder = `    match ext is ".html"
        true => ok true
        false => match ext is ".css"
            true => ok true
            false => ok false
`
	const flat = `    match ext
        ".html" | ".css" => ok true
        _ => ok false
`
	for _, test := range []struct {
		name, body string
		count      int
	}{{"ladder warns", ladder, 1}, {"flat cases are clean", flat, 0}} {
		t.Run(test.name, func(t *testing.T) {
			text := prefix + test.body
			root := writeServerProject(t, map[string]string{"src/main.can": text})
			uri := uriFromPath(filepath.Join(root, "src/main.can"))
			frames := runExchange(t, []string{didOpen(uri, text, 1)})
			published := lastPublishFor(frames, uri)
			if published == nil {
				t.Fatal("no diagnostics publication")
			}
			diagnostics := diagnosticsOf(t, published)
			if len(diagnostics) != test.count {
				t.Fatalf("unexpected diagnostics: %+v", diagnostics)
			}
			if test.count == 0 {
				return
			}
			diagnostic := diagnostics[0]
			if diagnostic["severity"] != 2.0 || diagnostic["code"] != "CAN-CHECK-STRING-MATCH-LADDER" {
				t.Fatalf("not the advisory warning: %+v", diagnostic)
			}
			rangeValue := diagnostic["range"].(map[string]any)
			start := rangeValue["start"].(map[string]any)
			end := rangeValue["end"].(map[string]any)
			if start["line"] != float64(strings.Count(prefix, "\n")) || start["character"] != 10.0 || end["line"] != start["line"] || end["character"] != 24.0 {
				t.Fatalf("warning does not highlight the first condition: %+v", rangeValue)
			}
		})
	}
}
