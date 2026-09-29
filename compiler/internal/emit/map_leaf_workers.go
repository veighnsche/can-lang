package emit

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// MapLeafProof ties one concrete checked map-leaf function to its
// synchronous Completion companion. Identity is the concrete function
// identity, Target the exact final async binding, Companion the
// deterministic sync companion binding, Source the defining source ID and
// Region the full checked region the companion lowers. KeyKind is the
// primitive key declaration (int, bool or str), MapID the concrete
// map<K,int> identity, and Calls maps each used collection
// specialization key to its exact final receiver.method target proven at
// assembly time. It is separate from authored, collection, core and
// integer worker proof.
type MapLeafProof struct {
	Identity  string
	Target    string
	Companion string
	Source    string
	Region    *ir.Region
	KeyKind   string
	MapID     string
	Calls     map[string]string
}

// mapLeafOperation is one canonical map operation admitted in leaves:
// its native method name, whether it takes an int value operand, and the
// single domain error it may raise.
type mapLeafOperation struct {
	method string
	value  bool
	error  string
}

// mapLeafOperations is exactly the three finite canonical map
// operations. Anything outside this table never qualifies for leaf
// emission, including empty/build_map/remove/entries and all set calls.
var mapLeafOperations = map[string]mapLeafOperation{
	"can.std.collections@1::get":     {method: "get", value: false, error: "can.std.collections@1::key_absent"},
	"can.std.collections@1::insert":  {method: "insert", value: true, error: "can.std.collections@1::key_exists"},
	"can.std.collections@1::replace": {method: "replace", value: true, error: "can.std.collections@1::key_absent"},
}

// MapLeafCompanionSuffix names the deterministic companion binding
// derivation. The companion takes exactly the two original inputs and no
// trailing context parameter.
const MapLeafCompanionSuffix = "$Leaf"

// provenMapLeaf reports whether identity names a proven leaf whose actual
// resolved binding still agrees with the entry, returning the companion
// binding. Unknown, mismatched or rebound targets never qualify.
func (e *RegionEmitter) provenMapLeaf(identity, target string) (string, bool) {
	if e.mapLeaves == nil || identity == "" || target == "" {
		return "", false
	}
	entry, ok := e.mapLeaves[identity]
	if !ok || entry == nil || entry.Target != target || entry.Companion == "" {
		return "", false
	}
	return entry.Companion, true
}

// LeafCompanion emits the synchronous Completion companion for a proven
// map-leaf function alongside its unchanged async entry. A fresh
// isolated emitter reuses checked block/completion-match/expression
// lowering with proof-selected synchronous invocation; the calling
// emitter keeps no leaf state and its origin cache is untouched. Region
// inputs bind to companion parameters in order; mappings keep exact
// defining sources/spans; module-hoisted lazy origin slots allocate at
// most once per call site across all elements. Failures escape through
// a cold catch returning the caught Completion; the existing fold
// boundary carries it onward.
func (e *RegionEmitter) LeafCompanion(entry *MapLeafProof) (string, error) {
	if entry == nil || entry.Region == nil || entry.Companion == "" || !jsBinding.MatchString(entry.Companion) {
		return "", fmt.Errorf("invalid map leaf proof")
	}
	region := entry.Region
	if region.Body == nil || len(region.Body.Steps) != 0 || len(region.Inputs) != 2 {
		return "", fmt.Errorf("invalid map leaf region")
	}
	companion := &RegionEmitter{
		Functions: e.Functions,
		SourceID:  e.SourceID,
		Browser:   e.Browser,
		region:    region,
		leaf:      entry,
	}
	if companion.SourceID != "" {
		companion.originCacheActive = true
		companion.originCachePrefix = "$canOrigin_" + entry.Companion + "_"
		companion.originCache = map[originCacheKey]int{}
		companion.expression.Mark = func(node *ir.Expression) string { return companion.markNode(node, string(node.Kind)) }
	}
	companion.expression.Bindings = map[string]string{
		region.Inputs[0].Identity: "$w0",
		region.Inputs[1].Identity: "$w1",
	}
	companion.expression.Browser = e.Browser
	body, err := companion.block(region.Body)
	if err != nil {
		return "", err
	}
	origin := companion.origin(region.Span)
	prefix := ""
	if companion.SourceID != "" {
		prefix = mappingMark(companion.SourceID, region.Span, "function") + "let $canOrigin = " + origin + ";\n"
		origin = "$canOrigin"
	}
	var slots strings.Builder
	for index := range companion.originSlots {
		fmt.Fprintf(&slots, "var %s%d: Readonly<{source: string; start: number; end: number; invocation: readonly string[]}> | undefined;\n", companion.originCachePrefix, index)
	}
	params := "$w0: " + TypeName(region.Inputs[0].Type) + ", $w1: " + TypeName(region.Inputs[1].Type)
	return fmt.Sprintf("function %s(%s): $canCompletion<%s> {\n%stry {\n%s} catch ($canCause) { return $canCaught($canCause, %s); }\n}\n", entry.Companion, params, TypeName(region.Result), prefix, body, origin) + slots.String(), nil
}

// Frozen M2-M4 shared contracts (implemented under G70-G72 against this
// freeze; the proof above only admits bodies that satisfy them):
//
// Companion: `function <target>$Leaf($w0: <map>, $w1: <key>):
// $canCompletion<<map>>` — synchronous, own-data length exactly 2,
// returning an authenticated Completion, never a Promise. Proven map
// calls lower to `$canInvokeSync(() =>
// $canMapMethodWorker(<receiver.method>)!(args...), origin)` at the
// original authored span; the assertion erases at runtime and unknown
// worker identity fails closed through that same boundary without
// replaying partial effects.
//
// Descriptor: separate optional eighth ownCallable argument
// `{companion, keyKind, origin}` with exactly those three own keys.
// Companion must be a function with own-data length 2; keyKind exactly
// "int", "bool" or "str"; origin a non-proxy object. Proxy checks
// precede own-data reads; malformed metadata throws without running.
//
// Queries: compiler-private `mapMethodWorker(fn)` returns the exact
// factory-owned sync body or undefined (private WeakMap identity, no
// foreign inspection); `mapLeafWorker(callable)` returns the registered
// `{run, keyKind}` or undefined from its own WeakMap, never consulting
// integer or forwarding admission.
//
// Traversal: array fold only, with context and owner strictly
// undefined, over real non-proxy ordinary frozen dense arrays of the
// matching primitive kind. Native reduce drives invokeSync per
// element; the first non-ok carrier stops traversal and is retained;
// the final result is a fresh Completion before Promise exposure.

// checkedMapLeaves proves every program function whose checked region is
// a closed map leaf: exactly (map<K,int>, K) inputs with K int/bool/str,
// the identical map result, no errors or escapes, no body statements and
// a terminal over statement-free success, relay or completion-match
// blocks. Expressions admit scoped typed literals/bindings and int unary
// minus/+,-,*; calls admit one plain step to the checked canonical
// get/insert/replace specialization of that SAME map type. Anything
// else — fields, division, ordinary matches, explicit domain creation,
// authored/recursive/indirect calls, native expressions, fixtures,
// array/asset/SQL/FormAction/JSONFetch/Action calls, coordination,
// resources, browser or mismatched proof — never qualifies.
// Specialization request records are provenance, not effects. Proof
// depends only on checked IR, never on workload names or emitted text.
func (assembly *programAssembly) checkedMapLeaves() map[string]*MapLeafProof {
	proof := make(map[string]*MapLeafProof)
	for _, fn := range assembly.program.Functions {
		entry, ok := mapLeafEntry(assembly, fn)
		if !ok {
			continue
		}
		proof[entry.Identity] = entry
	}
	return proof
}

func mapLeafEntry(assembly *programAssembly, fn *check.ProgramFunction) (*MapLeafProof, bool) {
	if fn == nil || fn.Region == nil || fn.Symbol == nil || fn.Symbol.Source.ID == "" {
		return nil, false
	}
	region := fn.Region
	if region.Kind != ir.FunctionRegion || region.Result == nil {
		return nil, false
	}
	mapID, keyKind, ok := mapLeafMapType(region.Result)
	if !ok {
		return nil, false
	}
	if len(region.Errors) != 0 || len(region.Escapes) != 0 {
		return nil, false
	}
	if len(region.Inputs) != 2 {
		return nil, false
	}
	if region.Inputs[0].Identity == "" || region.Inputs[0].Type == nil || region.Inputs[0].Type.Identity() != mapID {
		return nil, false
	}
	if region.Inputs[1].Identity == "" || region.Inputs[1].Type == nil || region.Inputs[1].Type.Declaration() != keyKind {
		return nil, false
	}
	if !mapLeafKnownMap(assembly, mapID) {
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
	entry := &MapLeafProof{
		Identity:  identity,
		Target:    target,
		Companion: target + MapLeafCompanionSuffix,
		Source:    fn.Symbol.Source.ID,
		Region:    region,
		KeyKind:   keyKind,
		MapID:     mapID,
		Calls:     map[string]string{},
	}
	scope := map[string]*ir.Local{
		region.Inputs[0].Identity: {Identity: region.Inputs[0].Identity, Type: region.Inputs[0].Type},
		region.Inputs[1].Identity: {Identity: region.Inputs[1].Identity, Type: region.Inputs[1].Type},
	}
	if !mapLeafCompletion(assembly, entry, region, region.Body.Terminal, scope) {
		return nil, false
	}
	return entry, true
}

// mapLeafMapType admits exactly map<K,int> with primitive K int/bool/str,
// returning the concrete map identity and key declaration.
func mapLeafMapType(typ *types.Type) (string, string, bool) {
	if typ == nil {
		return "", "", false
	}
	args := typ.Arguments()
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return "", "", false
	}
	key := args[0].Declaration()
	if key != "int" && key != "bool" && key != "str" {
		return "", "", false
	}
	if args[1].Declaration() != "int" {
		return "", "", false
	}
	return typ.Identity(), key, true
}

// mapLeafKnownMap anchors the leaf map identity to a genuine checked map
// specialization of this program: Entry-selected factory kind with the
// identical collection identity. Collection-free bodies over unknown map
// shapes stay conservative.
func mapLeafKnownMap(assembly *programAssembly, mapID string) bool {
	if assembly == nil || assembly.program == nil || mapID == "" {
		return false
	}
	for _, special := range assembly.program.Collections {
		if special == nil || special.Collection == nil || special.Entry == nil {
			continue
		}
		if special.Collection.Identity() == mapID {
			return true
		}
	}
	return false
}

// mapLeafCompletion admits only same-region success, relay and
// completion-match terminals. Success carries a leaf value; relay
// carries one proven map call; match carries one proven map call with
// ok/domain/standard/forwarding arms over recursive completions. Scope
// grows only from checked inputs, success bindings and arm bindings.
func mapLeafCompletion(assembly *programAssembly, entry *MapLeafProof, region *ir.Region, node *ir.Completion, scope map[string]*ir.Local) bool {
	if node == nil || node.RegionID != region.ID || node.SelfTail {
		return false
	}
	switch node.Kind {
	case ir.SuccessCompletion:
		if node.Value == nil || node.Call != nil || node.Block != nil || node.Match != nil || node.Inherit != "" {
			return false
		}
		return mapLeafValue(entry, node.Value, scope)
	case ir.RelayCompletion:
		if node.Value != nil || node.Block != nil || node.Match != nil || node.Inherit != "" {
			return false
		}
		_, ok := mapLeafCall(assembly, entry, node.Call, scope)
		return ok
	case ir.MatchCompletion:
		if node.Value != nil || node.Call != nil || node.Block != nil || node.Inherit != "" || node.Match == nil {
			return false
		}
		return mapLeafMatch(assembly, entry, region, node.Match, scope)
	default:
		return false
	}
}

// mapLeafMatch admits one proven map call scrutinee with completion arms
// only. Ordinary data matches, value matches and empty arm lists are
// rejected; every domain error must belong to the scrutinee call.
func mapLeafMatch(assembly *programAssembly, entry *MapLeafProof, region *ir.Region, match *ir.Match, scope map[string]*ir.Local) bool {
	if match == nil || match.Call == nil || match.ValueResult != nil || len(match.Values) != 0 || len(match.Arms) == 0 {
		return false
	}
	step, ok := mapLeafCall(assembly, entry, match.Call, scope)
	if !ok {
		return false
	}
	allowed := map[string]bool{}
	for _, failure := range step.Errors {
		if failure == nil {
			return false
		}
		allowed[failure.Identity()] = true
	}
	for _, arm := range match.Arms {
		if !mapLeafArm(assembly, entry, region, step, arm, scope, allowed) {
			return false
		}
	}
	return true
}

// mapLeafArm admits one completion arm. Forwarding arms return the
// scrutinee carrier untouched; ok arms bind the exact call result;
// domain arms name a scrutinee error; standard arms bind the opaque
// snapshot. Non-forward arms carry recursive completions, never
// expression-position values.
func mapLeafArm(assembly *programAssembly, entry *MapLeafProof, region *ir.Region, step *ir.InvocationStep, arm ir.Arm, scope map[string]*ir.Local, allowed map[string]bool) bool {
	if arm.Forward {
		return true
	}
	if arm.Body == nil || arm.Value != nil {
		return false
	}
	switch arm.Outcome {
	case "ok":
		if arm.Error != nil {
			return false
		}
		if arm.Binding != nil {
			if arm.Binding.Identity == "" || arm.Binding.Type == nil || step.Result == nil || arm.Binding.Type.Identity() != step.Result.Identity() {
				return false
			}
		}
	case "domain":
		if arm.Error == nil || !allowed[arm.Error.Identity()] {
			return false
		}
		if arm.Binding != nil && (arm.Binding.Identity == "" || arm.Binding.Type == nil) {
			return false
		}
	case "standard":
		if arm.Error != nil {
			return false
		}
		if arm.Binding != nil && (arm.Binding.Identity == "" || arm.Binding.Type == nil) {
			return false
		}
	default:
		return false
	}
	extended := make(map[string]*ir.Local, len(scope)+1)
	for key, local := range scope {
		extended[key] = local
	}
	if arm.Binding != nil {
		extended[arm.Binding.Identity] = arm.Binding
	}
	if step.SuccessBinding != "" && step.Result != nil {
		extended[step.SuccessBinding] = &ir.Local{Identity: step.SuccessBinding, Type: step.Result}
	}
	return mapLeafCompletion(assembly, entry, region, arm.Body, extended)
}

// mapLeafCall admits one plain invocation step to the checked canonical
// get/insert/replace specialization of the leaf map type: Entry-selected
// factory, exact operation/contract/error agreement and the exact final
// receiver.method target. It records the resolved target for emission.
func mapLeafCall(assembly *programAssembly, entry *MapLeafProof, call *ir.Invocation, scope map[string]*ir.Local) (*ir.InvocationStep, bool) {
	if call == nil || len(call.Steps) != 1 || call.Result == nil {
		return nil, false
	}
	step := &call.Steps[0]
	if step.Identity == "" || step.Site == "" || step.SuccessBinding == "" || step.Result == nil {
		return nil, false
	}
	if step.Callee != nil || step.Native != nil || step.Array != nil || step.Asset != nil || step.Fixtures != nil || step.SQL != nil || step.FormAction != nil || step.JSONFetch != nil || step.Action != nil {
		return nil, false
	}
	if step.Receiver {
		return nil, false
	}
	if call.Result.Identity() != step.Result.Identity() {
		return nil, false
	}
	special := assembly.program.Collections[step.Identity]
	if special == nil || special.Collection == nil || special.Entry == nil || special.Contract == nil {
		return nil, false
	}
	operation, ok := mapLeafOperations[special.Operation]
	if !ok {
		return nil, false
	}
	if special.Collection.Identity() != entry.MapID {
		return nil, false
	}
	if !mapLeafContract(entry, operation, special.Contract) {
		return nil, false
	}
	if step.Contract == nil || step.Contract.Identity() != special.Contract.Identity() {
		return nil, false
	}
	if step.Result.Identity() != special.Contract.Result().Identity() {
		return nil, false
	}
	contractErrors := special.Contract.Errors()
	if len(step.Errors) != len(contractErrors) {
		return nil, false
	}
	for i, failure := range step.Errors {
		if failure == nil || contractErrors[i] == nil || failure.Identity() != contractErrors[i].Identity() {
			return nil, false
		}
	}
	receiver, ok := assembly.collectionNames[entry.MapID]
	if !ok || receiver == "" {
		return nil, false
	}
	resolved, ok := assembly.functions[step.Identity]
	if !ok || resolved == "" || resolved != receiver+"."+operation.method {
		return nil, false
	}
	want := 2
	if operation.value {
		want = 3
	}
	if len(step.Arguments) != want {
		return nil, false
	}
	// The checker pins every call argument through preparation: each
	// prepared local carries the original argument value evaluated once
	// in order, and step arguments bind those locals positionally. The
	// companion reuses identical invocation lowering, so preparation is
	// preserved exactly — but only when every prepared value validates
	// as the leaf argument for its position and the bindings correspond
	// exactly. Unvalidated, reordered or foreign preparation is rejected
	// rather than silently changing evaluation.
	validate := func(position int, node *ir.Expression) bool {
		switch position {
		case 0:
			return mapLeafMapArgument(entry, node, scope)
		case 1:
			return mapLeafKeyArgument(entry, node, scope)
		default:
			return operation.value && mapLeafIntExpression(node, scope)
		}
	}
	if len(step.Prepare) == 0 {
		for position, argument := range step.Arguments {
			if !validate(position, argument) {
				return nil, false
			}
		}
	} else {
		if len(step.Prepare) != want {
			return nil, false
		}
		for position, prepared := range step.Prepare {
			if prepared.Value == nil || prepared.Value.Type == nil || prepared.Local.Identity == "" || prepared.Local.Type == nil || prepared.Local.Type.Identity() != prepared.Value.Type.Identity() {
				return nil, false
			}
			if !validate(position, prepared.Value) {
				return nil, false
			}
			argument := step.Arguments[position]
			if argument == nil || argument.Kind != ir.Binding || argument.Text != prepared.Local.Identity || argument.Type == nil || argument.Type.Identity() != prepared.Local.Type.Identity() || len(argument.Inputs) != 0 {
				return nil, false
			}
			if !mapLeafBare(argument, nil, "") {
				return nil, false
			}
		}
	}
	if previous, seen := entry.Calls[step.Identity]; seen && previous != resolved {
		return nil, false
	}
	entry.Calls[step.Identity] = resolved
	return step, true
}

// mapLeafContract verifies the exact operation signature over the leaf
// map: (map, key) or (map, key, int) inputs, int result for get and the
// map itself for insert/replace, with the single canonical domain error.
func mapLeafContract(entry *MapLeafProof, operation mapLeafOperation, contract *types.Type) bool {
	if contract == nil || contract.Result() == nil {
		return false
	}
	inputs := contract.Inputs()
	want := 2
	if operation.value {
		want = 3
	}
	if len(inputs) != want || inputs[0] == nil || inputs[1] == nil || inputs[0].Identity() != entry.MapID || inputs[1].Declaration() != entry.KeyKind {
		return false
	}
	if operation.value {
		if inputs[2] == nil || inputs[2].Declaration() != "int" {
			return false
		}
		if contract.Result().Identity() != entry.MapID {
			return false
		}
	} else if contract.Result().Declaration() != "int" {
		return false
	}
	failures := contract.Errors()
	if len(failures) != 1 || failures[0] == nil || failures[0].Declaration() != operation.error {
		return false
	}
	return true
}

// mapLeafValue admits leaf result values: the map itself, its key, or
// int arithmetic over scope. Map-typed bindings must name the exact leaf
// map; key bindings and literals must name the exact key kind.
func mapLeafValue(entry *MapLeafProof, node *ir.Expression, scope map[string]*ir.Local) bool {
	if node == nil || node.Type == nil {
		return false
	}
	switch node.Type.Identity() {
	case entry.MapID:
		return mapLeafMapArgument(entry, node, scope)
	}
	switch node.Type.Declaration() {
	case "int":
		return mapLeafIntExpression(node, scope)
	default:
		if node.Kind == ir.Binding || node.Kind == ir.Literal {
			return mapLeafKeyArgument(entry, node, scope)
		}
		return false
	}
}

// mapLeafMapArgument admits only a scoped binding of the exact leaf map.
func mapLeafMapArgument(entry *MapLeafProof, node *ir.Expression, scope map[string]*ir.Local) bool {
	if node == nil || node.Type == nil || node.Type.Identity() != entry.MapID {
		return false
	}
	if node.Kind != ir.Binding || len(node.Inputs) != 0 {
		return false
	}
	return mapLeafBare(node, scope, entry.MapID)
}

// mapLeafKeyArgument admits a scoped key binding or a key-kind literal.
func mapLeafKeyArgument(entry *MapLeafProof, node *ir.Expression, scope map[string]*ir.Local) bool {
	if node == nil || node.Type == nil || node.Type.Declaration() != entry.KeyKind {
		return false
	}
	switch node.Kind {
	case ir.Binding:
		if len(node.Inputs) != 0 {
			return false
		}
		local, ok := scope[node.Text]
		return ok && local != nil && local.Type != nil && local.Type.Identity() == node.Type.Identity() && mapLeafBare(node, nil, "")
	case ir.Literal:
		return len(node.Inputs) == 0 && mapLeafBare(node, nil, "")
	default:
		return false
	}
}

// mapLeafIntExpression admits closed int arithmetic: int literals, int
// bindings, unary minus and binary +,-,* with exact arity and types.
func mapLeafIntExpression(node *ir.Expression, scope map[string]*ir.Local) bool {
	if node == nil || node.Type == nil || node.Type.Declaration() != "int" {
		return false
	}
	if node.Coordination != nil || node.Callable != nil || node.Invocation != nil || node.Match != nil {
		return false
	}
	if len(node.Fields) != 0 || len(node.Operators) != 0 || len(node.Equality) != 0 {
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
		if len(node.Inputs) != 0 {
			return false
		}
		local, ok := scope[node.Text]
		return ok && local != nil && local.Type != nil && local.Type.Declaration() == "int"
	case ir.Unary:
		return node.Text == "-" && len(node.Inputs) == 1 && mapLeafIntExpression(node.Inputs[0], scope)
	case ir.Binary:
		if node.Text != "+" && node.Text != "-" && node.Text != "*" {
			return false
		}
		return len(node.Inputs) == 2 && mapLeafIntExpression(node.Inputs[0], scope) && mapLeafIntExpression(node.Inputs[1], scope)
	default:
		return false
	}
}

// mapLeafBare rejects auxiliary node fields and, when scope is given,
// resolves the binding identity to the required type identity.
func mapLeafBare(node *ir.Expression, scope map[string]*ir.Local, want string) bool {
	if node.Coordination != nil || node.Callable != nil || node.Invocation != nil || node.Match != nil {
		return false
	}
	if len(node.Fields) != 0 || len(node.Operators) != 0 || len(node.Equality) != 0 {
		return false
	}
	for _, spread := range node.Spread {
		if spread {
			return false
		}
	}
	if scope == nil {
		return true
	}
	local, ok := scope[node.Text]
	return ok && local != nil && local.Type != nil && local.Type.Identity() == want
}
