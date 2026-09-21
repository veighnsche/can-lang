package resolve

import (
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"strings"
	"testing"
)

const nativeConnection = "connection service\n    endpoint \"http://localhost:1\"\n    timeout_ms 1000\n"

func TestNativeKindsAndGeneratedNames(t *testing.T) {
	text := header("app", "weights, classify, service", "") + nativeConnection + `record weights choice float classify from service
    emits []
    confidence as certainty
    asks "Choose"
        first "First" => ok %
        second "Second" => ok %

fetch str load from service
    emits []
    get "/"

llm str generate from service
    emits []
    state
        str input
    asks input
`
	world, err := buildFiles(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	file := fileNamed(world, "app", "main.can")
	question := world.Packages["app"].Scope.Symbols["classify"]
	if !question.Eligible(QuestionUse) || question.Eligible(CallUse) || question.Eligible(ReferenceUse) {
		t.Fatal("question gained ordinary call eligibility")
	}
	record := world.Packages["app"].Scope.Symbols["weights"]
	if record.Kind != Record || !record.Constructible || !record.Public || record.ID == question.ID {
		t.Fatal("generated nominal identity missing")
	}
	fields := record.Declaration.(*syntax.RecordDecl).Fields
	if len(fields) != 3 || fields[0].Name.Text != "first" || fields[2].Name.Text != "certainty" {
		t.Fatal("generated field order missing")
	}
	if _, err := file.Lookup(nil, syntax.QualifiedName{Name: "service"}, ConnectionUse); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Lookup(nil, syntax.QualifiedName{Name: "generate"}, ReferenceUse); err == nil {
		t.Fatal("LLM acquired ordinary reference")
	}
	if _, err := file.Lookup(nil, syntax.QualifiedName{Name: "load"}, ReferenceUse); err != nil {
		t.Fatal(err)
	}
}
func TestNativeSignatureCollisionsAndVisibility(t *testing.T) {
	question := `record weights choice float classify from service
    emits []
    asks "Choose"
        first "First" => ok %
`
	for name, text := range map[string]string{
		"hidden generated record":   header("app", "classify, service", "") + nativeConnection + question,
		"private connection":        header("app", "weights, classify", "") + nativeConnection + question,
		"same generated name":       header("app", "", "") + nativeConnection + strings.Replace(question, "record weights", "record classify", 1),
		"ordinary name collision":   header("app", "", "") + nativeConnection + "record weights\n" + question,
		"wrong connection kind":     header("app", "", "") + "record service\n" + question,
		"duplicate state":           header("app", "", "") + nativeConnection + "llm str generate from service\n    emits []\n    given\n        str input\n    state\n        str input\n    asks input\n",
		"duplicate generated field": header("app", "", "") + nativeConnection + strings.Replace(question, "    asks", "    confidence as first\n    asks", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := buildFiles(t, map[string]string{"src/main.can": text}); err == nil {
				t.Fatal("accepted invalid native signature")
			}
		})
	}
}
