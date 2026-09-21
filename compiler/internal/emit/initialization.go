package emit

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

type InitializedValues struct {
	Code     string
	Bindings map[string]string
}

func InitializationImports(path string) string {
	return "import { captureStandard as $canCaptureStandard } from " + quote(path) + ";\n"
}

// Initialization emits synchronous native statements before the main invocation.
// A fault throws its opaque occurrence immediately, preventing later initializers
// and main from running. The owning entry supervisor reports that occurrence.
func Initialization(plan []ir.Initializer, namedArms map[string]string) (InitializedValues, error) {
	bindings := map[string]string{}
	for id, name := range namedArms {
		bindings[id] = name
	}
	emitter := ExpressionEmitter{Bindings: bindings}
	var out strings.Builder
	for i, entry := range plan {
		if _, exists := bindings[entry.Identity]; exists || entry.Identity == "" || entry.Value == nil || !types.Assignable(entry.Value.Type, entry.Type) {
			return InitializedValues{}, fmt.Errorf("invalid checked initialization plan")
		}
		lowered, err := emitter.Lower(entry.Value)
		if err != nil {
			return InitializedValues{}, err
		}
		name := fmt.Sprintf("$canInitial%d", i)
		fmt.Fprintf(&out, "let %s;\ntry {\n%s%s = %s;\n} catch ($canInitialCause) {\nthrow $canCaptureStandard($canInitialCause, { source: %s, start: %d, end: %d, invocation: [%s] });\n}\n", name, lowered.Statements, name, lowered.Value, quote(entry.Source), entry.Span.Start, entry.Span.End, quote("initialization:"+entry.QualifiedName))
		bindings[entry.Identity] = name
	}
	return InitializedValues{Code: out.String(), Bindings: bindings}, nil
}
