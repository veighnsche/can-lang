package emit

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
)

// A03: document cases emit the document response mode on the existing URL
// machinery. The checked declarations below stand in for the C03 action
// checker, which owns the GET/html agreement and case rules; A owns the
// grammar slice above and this emission mapping.
func documentProgram(t *testing.T, mutate func(*check.ActionDeclaration)) string {
	t.Helper()
	program := actionEmitProgram(t, map[string]string{"src/web/web.can": actionEmitWeb})
	if len(program.Actions) != 2 {
		t.Fatalf("checked %d actions, want 2", len(program.Actions))
	}
	for _, action := range program.Actions {
		if action.Method == "GET" {
			mutate(action)
		}
	}
	return emittedBody(t, program)
}

func TestActionDocumentEmissionFreezesMode(t *testing.T) {
	joined := documentProgram(t, func(action *check.ActionDeclaration) {
		action.Body = "html"
		for i := range action.Cases {
			action.Cases[i].Swap = "document"
		}
	})
	webID := "can.project.root/web"
	for _, want := range []string{
		`"method":"GET"`,
		`"path":"/invoices/:invoice_id"`,
		`"captures":[{"name":"invoice_id","type":"str"}]`,
		`"input":{"mode":"none"}`,
		`"body":"html"`,
		`"cases":[{"leaf":"` + webID + `::found","status":200,"document":true},{"leaf":"` + webID + `::missing","status":403,"document":true},{"leaf":"` + webID + `::unavailable","status":503,"document":true}]`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("emitted document action omits %s", want)
		}
	}
	if strings.Contains(joined, `"swap":"document"`) {
		t.Fatal("document mode leaked into the swap policy")
	}
	// The JSON POST action in the same program is byte-identical.
	if !strings.Contains(joined, `"cases":[{"leaf":"`+webID+`::saved","status":200},{"leaf":"`+webID+`::rejected","status":422},{"leaf":"`+webID+`::stale","status":409},{"leaf":"`+webID+`::denied","status":403},{"leaf":"`+webID+`::busy","status":503}]`) {
		t.Fatal("JSON action cases moved under document emission")
	}
}

func TestActionDocumentExcludedFromSwapTable(t *testing.T) {
	joined := documentProgram(t, func(action *check.ActionDeclaration) {
		action.Body = "html"
		for i := range action.Cases {
			action.Cases[i].Swap = "document"
		}
	})
	for _, line := range strings.Split(joined, "\n") {
		if !strings.Contains(line, "$canHTML=$canCreateHTML(") {
			continue
		}
		if !strings.HasSuffix(strings.TrimSpace(line), "},[]);") {
			t.Fatalf("document cases leaked into the swap table: %s", line)
		}
	}
}

func TestActionDocumentMixedWithSwapInner(t *testing.T) {
	joined := documentProgram(t, func(action *check.ActionDeclaration) {
		action.Body = "html"
		action.Cases[0].Swap = "document"
		action.Cases[1].Swap = "inner"
		action.Cases[2].Swap = "document"
	})
	if !strings.Contains(joined, `"status":200,"document":true`) || !strings.Contains(joined, `"status":403,"swap":"inner"`) || !strings.Contains(joined, `"status":503,"document":true`) {
		t.Fatalf("mixed modes not preserved: %.800s", joined)
	}
	want := `},[],[{"method":"GET","segments":["invoices","{}"],"cases":[{"status":403,"swap":"inner"}]}]);`
	if !strings.Contains(joined, want) {
		t.Fatalf("swap table lacks only the fragment case: %.800s", joined)
	}
}

func TestActionCaseMetadataRejectsUnknownMode(t *testing.T) {
	if _, err := actionCaseMetadata("leaf", 200, "outer"); err == nil {
		t.Fatal("unknown case mode accepted")
	}
	for _, mode := range []string{"", "inner", "document"} {
		if _, err := actionCaseMetadata("leaf", 200, mode); err != nil {
			t.Fatalf("selected mode %q rejected: %v", mode, err)
		}
	}
}
