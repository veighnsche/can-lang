package emit

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// ClosedRecoveryProof ties one concrete checked closed-recovery function
// to its native boolean branch. Identity is the concrete function
// identity, Target the exact final async binding, Source the defining
// source ID and Region the full checked region. Input is the single int
// input binding, Threshold the integer literal text of the proven
// input-literal `>` comparison, Reason the validated literal reason and
// Predicate the comparison node whose defining spans mark the hot
// branch. It is separate from authored, collection, core, integer
// worker, map-leaf and map-batch proof.
type ClosedRecoveryProof struct {
	Identity  string
	Target    string
	Source    string
	Region    *ir.Region
	Input     string
	Threshold string
	Reason    string
	Predicate *ir.Expression
}

// closedRecoveryRequire is the only canonical invocation a closed
// recovery admits; closedRecoveryTarget its only exact final binding.
const (
	closedRecoveryRequire = "can.std.checks@1::require"
	closedRecoveryTarget  = "$canChecks.require"
	closedRecoveryFailed  = "can.std.checks@1::failed"
)

var closedRecoveryInt = regexp.MustCompile(`^-?[0-9]+$`)

// checkedClosedRecoveries proves every program function whose checked
// region is a closed recovery: one int input, bool result, no errors or
// escapes, no body statements and a terminal same-region completion
// match over a single canonical checks::require call. The call takes a
// pure int-input `>` integer-literal comparison and a literal reason
// through checked positional preparation, and exactly two literal arms
// return false on checks::failed and true on ok. Anything else —
// payload access, standard arms, extra/effectful/reordered arguments,
// foreign preparation, extra statements, wrong types or any other IR
// node — never qualifies. Proof depends only on checked IR, never on
// workload names or emitted text.
func (assembly *programAssembly) checkedClosedRecoveries() map[string]*ClosedRecoveryProof {
	proof := make(map[string]*ClosedRecoveryProof)
	for _, fn := range assembly.program.Functions {
		entry, ok := closedRecoveryEntry(assembly, fn)
		if !ok {
			continue
		}
		proof[entry.Identity] = entry
	}
	return proof
}

func closedRecoveryEntry(assembly *programAssembly, fn *check.ProgramFunction) (*ClosedRecoveryProof, bool) {
	if fn == nil || fn.Region == nil || fn.Symbol == nil || fn.Symbol.Source.ID == "" {
		return nil, false
	}
	region := fn.Region
	if region.Kind != ir.FunctionRegion || region.Result == nil || region.Result.Declaration() != "bool" {
		return nil, false
	}
	if len(region.Errors) != 0 || len(region.Escapes) != 0 {
		return nil, false
	}
	if len(region.Inputs) != 1 {
		return nil, false
	}
	input := region.Inputs[0]
	if input.Identity == "" || input.Type == nil || input.Type.Declaration() != "int" {
		return nil, false
	}
	if region.Body == nil || len(region.Body.Steps) != 0 || region.Body.Terminal == nil {
		return nil, false
	}
	identity := fn.Identity()
	target, ok := assembly.functions[identity]
	if identity == "" || !ok || target == "" {
		return nil, false
	}
	entry := &ClosedRecoveryProof{
		Identity: identity,
		Target:   target,
		Source:   fn.Symbol.Source.ID,
		Region:   region,
		Input:    input.Identity,
	}
	if !closedRecoveryTerminal(assembly, entry, region, input) {
		return nil, false
	}
	if entry.Threshold == "" || entry.Predicate == nil {
		return nil, false
	}
	return entry, true
}

// closedRecoveryTerminal admits the complete single-require terminal: a
// two-arm completion match over the proven require call whose named
// domain arm returns literal false and whose ok arm returns literal
// true. Arm order is irrelevant; the arm set is exact.
func closedRecoveryTerminal(assembly *programAssembly, entry *ClosedRecoveryProof, region *ir.Region, input ir.Local) bool {
	node := region.Body.Terminal
	if node == nil || node.Kind != ir.MatchCompletion || node.RegionID != region.ID || node.SelfTail {
		return false
	}
	if node.Value != nil || node.Call != nil || node.Block != nil || node.Inherit != "" || node.Match == nil {
		return false
	}
	match := node.Match
	if match.Call == nil || match.ValueResult != nil || len(match.Values) != 0 || len(match.Arms) != 2 {
		return false
	}
	if !closedRecoveryCall(assembly, entry, match.Call, input) {
		return false
	}
	var failed, accept *ir.Arm
	for i := range match.Arms {
		arm := &match.Arms[i]
		if arm.Forward || arm.Value != nil || arm.Body == nil || len(arm.Patterns) != 0 {
			return false
		}
		switch arm.Outcome {
		case "domain":
			if failed != nil || arm.Error == nil || arm.Error.Declaration() != closedRecoveryFailed {
				return false
			}
			if arm.Binding != nil && (arm.Binding.Identity == "" || arm.Binding.Type == nil) {
				return false
			}
			if !closedRecoveryLiteral(arm.Body, region, "false") {
				return false
			}
			failed = arm
		case "ok":
			if accept != nil || arm.Error != nil || arm.Binding != nil {
				return false
			}
			if !closedRecoveryLiteral(arm.Body, region, "true") {
				return false
			}
			accept = arm
		default:
			return false
		}
	}
	return failed != nil && accept != nil
}

// closedRecoveryLiteral admits a success completion returning precisely
// the named boolean literal and nothing else.
func closedRecoveryLiteral(node *ir.Completion, region *ir.Region, want string) bool {
	if node == nil || node.Kind != ir.SuccessCompletion || node.RegionID != region.ID || node.SelfTail {
		return false
	}
	if node.Value == nil || node.Call != nil || node.Block != nil || node.Match != nil || node.Inherit != "" {
		return false
	}
	value := node.Value
	if value == nil || value.Type == nil || value.Type.Declaration() != "bool" {
		return false
	}
	if value.Kind != ir.Literal || len(value.Inputs) != 0 || value.Text != want {
		return false
	}
	return mapLeafBare(value, nil, "")
}

// closedRecoveryCall admits the single plain require step with the
// exact checked contract, failure identity and resolved target, plus
// pure positional arguments through the checker's ordered preparation.
func closedRecoveryCall(assembly *programAssembly, entry *ClosedRecoveryProof, call *ir.Invocation, input ir.Local) bool {
	if call == nil || len(call.Steps) != 1 || call.Result == nil || call.Result.Kind() != types.Void {
		return false
	}
	step := &call.Steps[0]
	if step.Identity != closedRecoveryRequire || step.Site == "" || step.SuccessBinding == "" || step.Result == nil {
		return false
	}
	if step.Callee != nil || step.Native != nil || step.Array != nil || step.Asset != nil || step.Fixtures != nil || step.SQL != nil || step.FormAction != nil || step.JSONFetch != nil || step.Action != nil {
		return false
	}
	if step.Receiver {
		return false
	}
	if call.Result.Identity() != step.Result.Identity() || step.Result.Kind() != types.Void {
		return false
	}
	contract := step.Contract
	if contract == nil || contract.Result() == nil || contract.Result().Kind() != types.Void {
		return false
	}
	contractInputs := contract.Inputs()
	if len(contractInputs) != 2 || contractInputs[0] == nil || contractInputs[0].Declaration() != "bool" || contractInputs[1] == nil || contractInputs[1].Declaration() != "str" {
		return false
	}
	contractErrors := contract.Errors()
	if len(contractErrors) != 1 || contractErrors[0] == nil || contractErrors[0].Declaration() != closedRecoveryFailed {
		return false
	}
	if len(step.Errors) != 1 || step.Errors[0] == nil || step.Errors[0].Identity() != contractErrors[0].Identity() {
		return false
	}
	if len(call.Errors) != 1 || call.Errors[0] == nil || call.Errors[0].Identity() != contractErrors[0].Identity() {
		return false
	}
	resolved, ok := assembly.functions[step.Identity]
	if !ok || resolved == "" || resolved != closedRecoveryTarget {
		return false
	}
	if len(step.Arguments) != 2 {
		return false
	}
	// Like the leaf proof, every call argument passes through checked
	// positional preparation: each prepared local carries the original
	// argument value evaluated once in order. The recovery proof
	// additionally requires pure original values — an int-input `>`
	// integer-literal comparison and a string literal — so the native
	// branch can evaluate the predicate exactly once.
	values := make([]*ir.Expression, 2)
	if len(step.Prepare) == 0 {
		values[0], values[1] = step.Arguments[0], step.Arguments[1]
	} else {
		if len(step.Prepare) != 2 {
			return false
		}
		for position, prepared := range step.Prepare {
			if prepared.Value == nil || prepared.Value.Type == nil || prepared.Local.Identity == "" || prepared.Local.Type == nil || prepared.Local.Type.Identity() != prepared.Value.Type.Identity() {
				return false
			}
			values[position] = prepared.Value
			argument := step.Arguments[position]
			if argument == nil || argument.Kind != ir.Binding || argument.Text != prepared.Local.Identity || argument.Type == nil || argument.Type.Identity() != prepared.Local.Type.Identity() || len(argument.Inputs) != 0 {
				return false
			}
			if !mapLeafBare(argument, nil, "") {
				return false
			}
		}
	}
	threshold, predicate, ok := closedRecoveryPredicate(values[0], input)
	if !ok {
		return false
	}
	reason := values[1]
	if reason == nil || reason.Type == nil || reason.Type.Declaration() != "str" || reason.Kind != ir.Literal || len(reason.Inputs) != 0 {
		return false
	}
	if !mapLeafBare(reason, nil, "") {
		return false
	}
	entry.Threshold, entry.Reason, entry.Predicate = threshold, reason.Text, predicate
	return true
}

// closedRecoveryPredicate admits precisely an int-input `>`
// integer-literal comparison with no other node. It returns the literal
// text and the comparison node for exact span marks.
func closedRecoveryPredicate(node *ir.Expression, input ir.Local) (string, *ir.Expression, bool) {
	if node == nil || node.Type == nil || node.Type.Declaration() != "bool" {
		return "", nil, false
	}
	if node.Kind != ir.Comparison || len(node.Operators) != 1 || node.Operators[0] != ">" || len(node.Inputs) != 2 {
		return "", nil, false
	}
	// The comparison legitimately carries its operator and the
	// checker's strict ordered-comparison mode, so its auxiliary
	// fields are checked directly instead of through the
	// operator-free bare helper used for every other node.
	if node.Coordination != nil || node.Callable != nil || node.Invocation != nil || node.Match != nil {
		return "", nil, false
	}
	if len(node.Fields) != 0 || len(node.Equality) != 1 || node.Equality[0] != ir.Strict {
		return "", nil, false
	}
	for _, spread := range node.Spread {
		if spread {
			return "", nil, false
		}
	}
	left, right := node.Inputs[0], node.Inputs[1]
	if left == nil || left.Type == nil || left.Type.Declaration() != "int" || left.Kind != ir.Binding || len(left.Inputs) != 0 || left.Text != input.Identity {
		return "", nil, false
	}
	if !mapLeafBare(left, nil, "") {
		return "", nil, false
	}
	if right == nil || right.Type == nil || right.Type.Declaration() != "int" || right.Kind != ir.Literal || len(right.Inputs) != 0 || !closedRecoveryInt.MatchString(right.Text) {
		return "", nil, false
	}
	if !mapLeafBare(right, nil, "") {
		return "", nil, false
	}
	return right.Text, node, true
}

// provenClosedRecovery reports the closed-recovery proof for an exact
// proven region/target pair. Function regions carry their concrete
// function identity; stale, rebound or unknown pairs never qualify.
// The returned entry is assembly-owned and must not be mutated.
func (e *RegionEmitter) provenClosedRecovery(regionID, target string) (*ClosedRecoveryProof, bool) {
	if e.closedRecoveries == nil || regionID == "" || target == "" {
		return nil, false
	}
	entry, ok := e.closedRecoveries[regionID]
	if !ok || entry == nil || entry.Identity != regionID || entry.Target != target {
		return nil, false
	}
	if entry.Region == nil || entry.Predicate == nil || entry.Threshold == "" || !closedRecoveryInt.MatchString(entry.Threshold) {
		return nil, false
	}
	return entry, true
}

// closedRecoveryImport reports whether the output path defines a proven
// closed-recovery function needing the private branch imports. The
// caller gates on Bun emission; browser modules never import them.
func (assembly *programAssembly) closedRecoveryImport(path string) bool {
	if assembly == nil || assembly.program == nil || len(assembly.closedRecoveries) == 0 {
		return false
	}
	for _, fn := range assembly.program.Functions {
		if fn == nil || fn.Symbol == nil || fn.Symbol.Source.OutputPath != path {
			continue
		}
		if _, ok := assembly.closedRecoveries[fn.Identity()]; ok {
			return true
		}
	}
	return false
}

// ClosedRecoveryBranch emits the private native boolean branch
// prepended to a proven closed-recovery async function. Admission
// needs no $canOrigin and allocates no origin object or array: only
// an undefined assertion context, no ambient owner and a primitive
// bigint input enter, and the native `>` comparison evaluates exactly
// once. True returns a branded success without touching the global
// occurrence counter; false advances that counter exactly once — the
// same single ID the consumed domain failure would allocate — before
// returning a branded failure-free success. No reason record, domain
// token, extra Promise chain or mutable payload is created. A cold
// synchronous fault is caught and boxed with the original region
// source/span; every decline falls through to the unchanged async
// body. The emitter keeps no recovery state.
func (e *RegionEmitter) ClosedRecoveryBranch(entry *ClosedRecoveryProof) (string, error) {
	if entry == nil || entry.Region == nil || entry.Predicate == nil || entry.Threshold == "" || !closedRecoveryInt.MatchString(entry.Threshold) {
		return "", fmt.Errorf("invalid closed recovery proof")
	}
	region := entry.Region
	if len(region.Inputs) != 1 {
		return "", fmt.Errorf("invalid closed recovery region")
	}
	marks := ""
	if e.SourceID != "" {
		marks = integerWorkerMark(e.SourceID, entry.Predicate, "comparison")
		if len(entry.Predicate.Inputs) == 2 && entry.Predicate.Inputs[1] != nil {
			marks += integerWorkerMark(e.SourceID, entry.Predicate.Inputs[1], "literal")
		}
	}
	source := e.SourceID
	if source == "" {
		source = region.Source
	}
	cold := fmt.Sprintf("{source:%s,start:%d,end:%d,invocation:[%s]}", quote(source), region.Span.Start, region.Span.End, quote(region.ID))
	var out strings.Builder
	out.WriteString("try {\n")
	out.WriteString("if ($canContext === undefined && !$canAmbientOwnerPresent() && typeof $canArg0 === \"bigint\") {\n")
	fmt.Fprintf(&out, "%sif ($canArg0 > %sn) return $canSuccess(true);\n", marks, entry.Threshold)
	out.WriteString("$canAllocateOccurrenceID();\nreturn $canSuccess(false);\n}\n")
	fmt.Fprintf(&out, "} catch ($canError) { return $canCaught($canError, %s); }\n", cold)
	return out.String(), nil
}
