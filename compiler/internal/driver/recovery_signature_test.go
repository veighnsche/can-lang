package driver

import (
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"strings"
	"testing"
)

const signatureRecoverySource = `package app
    provides []
    uses []
record settings
    str title
fn str render
    emits {}
    given
        settings config
    asserts
        sample: settings("title") => ok "title"
    ok config.
`

func TestRecoverySignatureIndependentOfBody(t *testing.T) {
	for _, tc := range []struct{ name, text string }{
		{"incomplete member", signatureRecoverySource},
		{"invalid result", strings.Replace(signatureRecoverySource, "ok config.", "ok 42", 1)},
		{"invalid assertion", strings.Replace(strings.Replace(signatureRecoverySource, "ok config.", "ok config.title", 1), `=> ok "title"`, `=> ok 42`, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := reviewSnapshot(t, tc.text, nil)
			if snapshot.Program == nil {
				t.Fatal("missing partial program")
			}
			found := false
			for _, fn := range snapshot.Program.Functions {
				if fn.Symbol.Name != "render" {
					continue
				}
				found = true
				if fn.Instance != "" || fn.Region != nil {
					t.Fatal("invalid body retained executable region or unexpected specialization")
				}
				signature := fn.Signature
				if !types.Equal(signature, signature) || signature.Kind() != types.Callable {
					t.Fatal("independent sealed signature missing")
				}
				inputs := signature.Inputs()
				if len(inputs) != 1 || inputs[0].Kind() != types.Record || !strings.HasSuffix(inputs[0].Declaration(), "::settings") || len(inputs[0].Fields()) != 1 || inputs[0].Fields()[0].Name != "title" {
					t.Fatal("wrong proven input record")
				}
			}
			if !found {
				t.Fatal("retained declaration missing")
			}
		})
	}
}

func TestRecoverySignatureRejectsInvalidOrAmbiguousHeader(t *testing.T) {
	validBody := strings.Replace(signatureRecoverySource, "ok config.", "ok config.title", 1)
	for _, tc := range []struct{ name, text string }{
		{"invalid input", strings.Replace(validBody, "settings config", "missing_settings config", 1)},
		{"invalid bound", strings.Replace(validBody, "emits {}", "emits {missing_error}", 1)},
		{"ambiguous", validBody + `fn str render
    emits {}
    given
        settings other
    asserts
        sample: settings("title") => ok "title"
    ok other.title
`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := reviewSnapshot(t, tc.text, nil)
			if snapshot.Program == nil {
				return
			}
			for _, fn := range snapshot.Program.Functions {
				if fn.Symbol.Name == "render" && (fn.Signature != nil || fn.Region != nil) {
					t.Fatal("invalid or ambiguous declaration published signature/body proof")
				}
			}
		})
	}
}

func TestRecoverySignaturePreservesValidExecutableFunction(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": strings.Replace(signatureRecoverySource, "ok config.", "ok config.title", 1)})
	snapshot, err := CheckSnapshot(root, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Diagnostics) != 0 {
		t.Fatalf("valid signature fixture failed: %+v", snapshot.Diagnostics)
	}
	program, err := check.CheckAssertionProgram(snapshot.Graph)
	if err != nil || program == nil {
		t.Fatalf("strict valid program rejected: %v", err)
	}
	for _, fn := range program.Functions {
		if fn.Symbol.Name == "render" {
			if fn.Signature == nil || fn.Region == nil {
				t.Fatal("strict signature/body proof missing")
			}
			return
		}
	}
	t.Fatal("valid function missing")
}
