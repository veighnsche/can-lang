package driver

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

func TestRecoveryStagedBodyAnnotations(t *testing.T) {
	for _, later := range []bool{false, true} {
		name := "annotation-only"
		step := ""
		tokens := []string{"missing_type", "item"}
		if later {
			name = "independent-later-call"
			step = "    int other = call missing_later()\n"
			tokens = append(tokens, "missing_later")
		}
		t.Run(name, func(t *testing.T) {
			text := reviewHeader + `record settings
    str title
fn int damaged
    emits {}
    given
        settings config
    asserts
        sample: settings("title") => ok 1
    missing_type broken = 1
    item also_broken = 1
    int blocked = broken
` + step + `    ok 1
fn int dependent
    emits {}
    asserts
        sample: => ok 1
    ok call damaged(settings("title"))
fn str healthy
    emits {}
    asserts
        sample: => ok "valid"
    ok call identity<str>("valid")
fn item identity<item>
    emits {}
    given
        item value
    asserts
        number: 3 => ok 3
        text: "valid" => ok "valid"
    item alias = value
    ok alias
`
			snapshot := reviewSnapshot(t, text, nil)
			reviewErrorTokens(t, snapshot, tokens...)
			count := 0
			for _, diagnostic := range snapshot.Diagnostics {
				if diagnostic.Severity == "error" {
					count++
				}
			}
			if count != len(tokens) {
				t.Fatalf("failed annotations repeated or cascaded: %+v", snapshot.Diagnostics)
			}
			if snapshot.Program == nil {
				t.Fatal("independent headers were discarded")
			}
			foundDamaged, foundDependent, foundHealthy, foundGeneric := false, false, false, false
			for _, fn := range snapshot.Program.Functions {
				switch fn.Symbol.Name {
				case "damaged":
					foundDamaged = true
					if fn.Region != nil || fn.Signature == nil || !types.Equal(fn.Signature, fn.Signature) {
						t.Fatal("failed body lost sealed header or retained executable IR")
					}
					inputs := fn.Signature.Inputs()
					if len(inputs) != 1 || !strings.HasSuffix(inputs[0].Declaration(), "::settings") {
						t.Fatal("wrong independently proven input record")
					}
					if snapshot.Analysis.Unit(UnitID(fn.Symbol.ID)).Status != source.UnitInvalid {
						t.Fatal("direct annotation failure was replaced by blocked status")
					}
				case "dependent":
					foundDependent = true
					if fn.Region != nil || snapshot.Analysis.Unit(UnitID(fn.Symbol.ID)).Status != source.UnitBlocked {
						t.Fatal("dependent region escaped invalid body closure")
					}
				case "healthy":
					foundHealthy = true
					if fn.Region == nil {
						t.Fatal("independent healthy body was lost")
					}
				case "identity":
					if fn.Instance != "" && len(fn.TypeArguments) == 1 && fn.TypeArguments[0].Declaration() == "str" {
						foundGeneric = true
						if fn.Region == nil {
							t.Fatal("same-spelling annotation failure poisoned generic instance")
						}
					}
				}
			}
			if !foundDamaged || !foundDependent || !foundHealthy || !foundGeneric {
				t.Fatalf("missing independent evidence: damaged=%t dependent=%t healthy=%t generic=%t", foundDamaged, foundDependent, foundHealthy, foundGeneric)
			}
		})
	}
}

func TestRecoveryStagedConstructorGathering(t *testing.T) {
	for _, test := range []struct{ name, expression, token string }{
		{"unknown-name", "missing_constructor(1)", "missing_constructor"},
		{"synthetic-arity", "pair<int>(1, 2)", "pair"},
		{"synthetic-argument", "pair<missing_type, int>(1, 2)", "missing_type"},
	} {
		t.Run(test.name, func(t *testing.T) {
			text := reviewHeader + `record pair<a, b>
    a first
    b second
fn int damaged
    emits {}
    asserts
        sample: => ok 1
    int broken = ` + test.expression + `
    int other = call missing_later()
    ok 1
`
			snapshot := reviewSnapshot(t, text, nil)
			reviewErrorTokens(t, snapshot, test.token, "missing_later")
			count := 0
			for _, diagnostic := range snapshot.Diagnostics {
				if diagnostic.Severity == "error" {
					count++
				}
			}
			if count != 2 {
				t.Fatalf("constructor gather failure duplicated or cascaded: %+v", snapshot.Diagnostics)
			}
			if snapshot.Program == nil {
				t.Fatal("missing partial program")
			}
			for _, fn := range snapshot.Program.Functions {
				if fn.Symbol.Name == "damaged" {
					if fn.Signature == nil || fn.Region != nil {
						t.Fatal("constructor failure discarded header or retained invalid IR")
					}
					return
				}
			}
			t.Fatal("missing independently valid header")
		})
	}
}
