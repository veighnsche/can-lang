package driver

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

const driverStringLadder = `package app
    provides []
    uses []
fn bool allowed
    emits {}
    given
        str ext
    asserts
        sample: ".html" => ok true
    match ext is ".html"
        true => ok true
        false => match ext is ".css"
            true => ok true
            false => ok false
`

func TestStringMatchLadderDiagnostics(t *testing.T) {
	for _, damaged := range []bool{false, true} {
		name := "healthy"
		text := driverStringLadder
		if damaged {
			name = "unrelated error in same body"
			text = strings.Replace(text, "    match ext", "    int broken = call missing()\n    match ext", 1)
		}
		t.Run(name, func(t *testing.T) {
			root := writeBridgeProject(t, map[string]string{"src/main.can": text})
			report, exit, raw := checkJSON(t, root)
			if report.Result == nil || report.Result.Accepted == damaged || !damaged && exit != 0 || damaged && exit != 1 {
				t.Fatalf("warning changed acceptance or error exit: %d, %s", exit, raw)
			}
			count := 0
			for _, diagnostic := range report.Result.Diagnostics {
				if diagnostic.Code == "CAN-CHECK-STRING-MATCH-LADDER" {
					count++
					if diagnostic.Severity != "warning" || diagnostic.Location == nil || diagnostic.File == nil {
						t.Fatalf("missing advisory source contract: %+v", diagnostic)
					}
				}
			}
			if count != 1 {
				t.Fatalf("expected one structured CLI warning, got %s", raw)
			}
			snapshot, err := CheckSnapshot(root, "", nil)
			if err != nil {
				t.Fatal(err)
			}
			count = 0
			for _, diagnostic := range snapshot.Diagnostics {
				if diagnostic.Code != "CAN-CHECK-STRING-MATCH-LADDER" {
					continue
				}
				count++
				file, err := source.New(diagnostic.File, text)
				if err != nil {
					t.Fatal(err)
				}
				start, e1 := file.Offset(source.UTF16Position{Line: diagnostic.Line, Character: diagnostic.Start})
				end, e2 := file.Offset(source.UTF16Position{Line: diagnostic.EndLine, Character: diagnostic.End})
				if e1 != nil || e2 != nil || text[start:end] != `ext is ".html"` || diagnostic.Severity != "warning" {
					t.Fatalf("editor severity or first-condition span lost: %+v", diagnostic)
				}
			}
			if count != 1 {
				t.Fatalf("expected one editor warning, got %+v", snapshot.Diagnostics)
			}
		})
	}
}
