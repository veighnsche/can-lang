package emit

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// IntegerWorkerProof ties one concrete checked integer function to its
// synchronous companion. Identity is the concrete function identity or
// generic specialization, Target the exact final async binding, Companion
// the deterministic sync companion binding, Source the defining source ID
// and Region the full checked region the companion lowers.
type IntegerWorkerProof struct {
	Identity  string
	Target    string
	Companion string
	Source    string
	Region    *ir.Region
}

// checkedIntegerWorkers proves every program function whose checked region
// is a closed integer expression: int-only inputs/result, no errors or
// escapes, no body statements and one success completion over int
// literals, bound region inputs, unary minus and binary +,-,*. Anything
// else — division, fields, calls, match, coordination, resource-bearing or
// non-integer shapes — never qualifies. Specialization request records
// are provenance, not effects, so generic instances qualify exactly like
// concrete functions. Proof depends only on checked IR, never on workload
// names, annotations or emitted text.
func (assembly *programAssembly) checkedIntegerWorkers() map[string]*IntegerWorkerProof {
	proof := make(map[string]*IntegerWorkerProof)
	for _, fn := range assembly.program.Functions {
		entry, ok := integerWorkerEntry(assembly, fn)
		if !ok {
			continue
		}
		proof[entry.Identity] = entry
	}
	return proof
}

func integerWorkerEntry(assembly *programAssembly, fn *check.ProgramFunction) (*IntegerWorkerProof, bool) {
	if fn == nil || fn.Region == nil || fn.Symbol == nil || fn.Symbol.Source.ID == "" {
		return nil, false
	}
	region := fn.Region
	if region.Result == nil || region.Result.Declaration() != "int" {
		return nil, false
	}
	if len(region.Errors) != 0 || len(region.Escapes) != 0 {
		return nil, false
	}
	inputs := make(map[string]bool, len(region.Inputs))
	for _, input := range region.Inputs {
		if input.Identity == "" || input.Type == nil || input.Type.Declaration() != "int" {
			return nil, false
		}
		inputs[input.Identity] = true
	}
	if region.Body == nil || len(region.Body.Steps) != 0 || region.Body.Terminal == nil {
		return nil, false
	}
	terminal := region.Body.Terminal
	if terminal.Kind != ir.SuccessCompletion || terminal.Value == nil || terminal.Call != nil || terminal.Block != nil || terminal.Match != nil || terminal.Inherit != "" || terminal.SelfTail {
		return nil, false
	}
	if !integerWorkerExpression(terminal.Value, inputs) {
		return nil, false
	}
	identity := fn.Identity()
	target, ok := assembly.functions[identity]
	if identity == "" || !ok || target == "" {
		return nil, false
	}
	return &IntegerWorkerProof{
		Identity:  identity,
		Target:    target,
		Companion: target + "$Int",
		Source:    fn.Symbol.Source.ID,
		Region:    region,
	}, true
}

// integerWorkerExpression admits only closed integer arithmetic: int
// literals, bound region input references, unary minus and binary +,-,*
// with exact arity and integer types. Division, bitwise, comparison,
// fields, calls, match and coordination are rejected.
func integerWorkerExpression(node *ir.Expression, inputs map[string]bool) bool {
	if node == nil || node.Type == nil || node.Type.Declaration() != "int" {
		return false
	}
	if node.Coordination != nil || node.Callable != nil || node.Invocation != nil || node.Match != nil {
		return false
	}
	if len(node.Fields) != 0 {
		return false
	}
	for _, spread := range node.Spread {
		if spread {
			return false
		}
	}
	switch node.Kind {
	case ir.Literal:
		return len(node.Inputs) == 0
	case ir.Binding:
		return len(node.Inputs) == 0 && inputs[node.Text]
	case ir.Unary:
		return node.Text == "-" && len(node.Inputs) == 1 && integerWorkerExpression(node.Inputs[0], inputs)
	case ir.Binary:
		if node.Text != "+" && node.Text != "-" && node.Text != "*" {
			return false
		}
		return len(node.Inputs) == 2 && integerWorkerExpression(node.Inputs[0], inputs) && integerWorkerExpression(node.Inputs[1], inputs)
	default:
		return false
	}
}

// provenIntegerWorker reports the companion binding for an exact proven
// integer-worker identity/target pair. Stale, rebound or unknown pairs
// never qualify.
func (e *RegionEmitter) provenIntegerWorker(identity, target string) (string, bool) {
	if e.integerWorkers == nil || identity == "" || target == "" {
		return "", false
	}
	entry, ok := e.integerWorkers[identity]
	if !ok || entry == nil || entry.Target != target || entry.Companion == "" {
		return "", false
	}
	return entry.Companion, true
}

// integerWorkerMark attributes one lowered expression to its own defining
// source as a mapping token only. Template substitution moves definition
// nodes into use regions, so their spans address the definition file; every
// other node belongs to the region's module. Unlike markNode it assigns no
// $canOrigin and allocates no origin object, so successful worker elements
// carry no failure metadata. The cold catch owns the single origin.
func integerWorkerMark(sourceID string, node *ir.Expression, operation string) string {
	source := sourceID
	if node.Source != "" {
		source = node.Source
	}
	return mappingMark(source, node.Span, operation)
}

// IntegerCompanion emits the synchronous bigint companion for a proven
// integer function alongside its unchanged async entry. It reuses native
// expression lowering with region inputs bound to companion parameters in
// order, and mapping-only marks keep each node's defining source/span.
// The successful path declares and assigns no $canOrigin and allocates no
// origin object or array. Failures escape through a cold catch that throws
// the private failure value from caught at the original region
// source/span/invocation; the existing array boundary carries it onward.
// The emitter keeps no worker state: no cache reset or region assignment.
func (e *RegionEmitter) IntegerCompanion(entry *IntegerWorkerProof) (string, error) {
	if entry == nil || entry.Region == nil || entry.Companion == "" || !jsBinding.MatchString(entry.Companion) {
		return "", fmt.Errorf("invalid integer worker proof")
	}
	region := entry.Region
	if region.Body == nil || region.Body.Terminal == nil || region.Body.Terminal.Value == nil {
		return "", fmt.Errorf("invalid integer worker region")
	}
	bindings := make(map[string]string, len(region.Inputs))
	parameters := make([]string, len(region.Inputs))
	for i, input := range region.Inputs {
		parameters[i] = fmt.Sprintf("$w%d: bigint", i)
		bindings[input.Identity] = fmt.Sprintf("$w%d", i)
	}
	emitter := ExpressionEmitter{Bindings: bindings, Browser: e.Browser}
	if e.SourceID != "" {
		emitter.Mark = func(node *ir.Expression) string { return integerWorkerMark(e.SourceID, node, string(node.Kind)) }
	}
	lowered, err := emitter.Lower(region.Body.Terminal.Value)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	prefix := ""
	if e.SourceID != "" {
		prefix = mappingMark(e.SourceID, region.Span, "function")
	}
	source := e.SourceID
	if source == "" {
		source = region.Source
	}
	cold := fmt.Sprintf("{source:%s,start:%d,end:%d,invocation:[%s]}", quote(source), region.Span.Start, region.Span.End, quote(region.ID))
	fmt.Fprintf(&out, "function %s(%s): bigint {\n%stry {\n%sreturn %s;\n} catch ($canError) { throw $canCaught($canError, %s).value; }\n}\n",
		entry.Companion, strings.Join(parameters, ", "), prefix, lowered.Statements, lowered.Value, cold)
	return out.String(), nil
}
