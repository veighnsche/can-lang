package emit

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"strings"
)

func arrayImports() []ImportName {
	return []ImportName{{"map", "$canArrayMap"}, {"filter", "$canArrayFilter"}, {"fold", "$canArrayFold"}, {"forEach", "$canArrayForEach"}, {"find", "$canArrayFind"}, {"some", "$canArraySome"}, {"every", "$canArrayEvery"}, {"sortBy", "$canArraySortBy"}, {"concat", "$canArrayConcat"}, {"toReversed", "$canArrayToReversed"}, {"append", "$canArrayAppend"}}
}
func (e *RegionEmitter) arrayInvocation(step ir.InvocationStep, args []string) (string, error) {
	if step.Array.Name == "slice" {
		if len(args) != 3 {
			return "", fmt.Errorf("invalid checked slice reference")
		}
		return "$canSuccess($canSlice(" + strings.Join(args, ",") + "))", nil
	}
	name := map[string]string{"map": "Map", "filter": "Filter", "fold": "Fold", "for_each": "ForEach", "find": "Find", "some": "Some", "every": "Every", "sort_by": "SortBy", "concat": "Concat", "to_reversed": "ToReversed", "append": "Append"}[step.Array.Name]
	if name == "" {
		return "", fmt.Errorf("unknown checked array operation")
	}
	if step.Array.Name == "concat" || step.Array.Name == "to_reversed" || step.Array.Name == "append" {
		return "$canSuccess($canArray" + name + "(" + strings.Join(args, ",") + "))", nil
	}
	if step.Array.Name == "find" {
		if !types.Equal(step.Array.None, step.Array.None) || !types.Equal(step.Array.Some, step.Array.Some) {
			return "", fmt.Errorf("missing checked option identities")
		}
		args = append(args, "{none:"+quote(step.Array.None.Identity())+",some:"+quote(step.Array.Some.Identity())+"}")
	}
	if e.Browser {
		args = append(args, "{site:"+quote(step.Site)+",origin:"+e.origin(step.Span)+",context:$canContext,owner:$canCtx}")
	} else {
		args = append(args, "{site:"+quote(step.Site)+",origin:"+e.origin(step.Span)+",context:$canContext}")
	}
	invocation := "$canArray" + name + "(" + strings.Join(args, ",") + ")"
	if step.Array.Name == "find" {
		// The runtime constructs the two sealed nominal identities supplied above;
		// TypeScript cannot derive their generated union from identity strings.
		invocation = "(" + invocation + " as Promise<$canCompletion<" + TypeName(step.Result) + ">>)"
	}
	return invocation, nil
}
