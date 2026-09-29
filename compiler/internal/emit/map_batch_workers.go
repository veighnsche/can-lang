package emit

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// MapBatchProof ties one concrete checked batch-transition function to its
// two synchronous bigint companions. Identity is the concrete function
// identity, Target the exact final async binding, Absent/Present the
// deterministic value companions, Source the defining source ID and Region
// the full checked region. KeyKind is the primitive key declaration (int,
// bool or str), MapID the concrete map<K,int> identity, Receiver the exact
// emitted factory receiver, and Calls maps each of the three collection
// specialization keys to its exact final receiver.method target proven at
// assembly time. AbsentValue/PresentValue are the validated pure int
// prepared values, Previous the present-arm old-value binding, and GetSpan
// the first (get) call-site span for the cold boundary. It is separate
// from authored, collection, core, integer worker and map-leaf proof.
type MapBatchProof struct {
	Identity     string
	Target       string
	Absent       string
	Present      string
	Source       string
	Region       *ir.Region
	KeyKind      string
	MapID        string
	Receiver     string
	Calls        map[string]string
	AbsentValue  *ir.Expression
	PresentValue *ir.Expression
	Previous     string
	GetSpan      source.Span
}

// MapBatchAbsentSuffix and MapBatchPresentSuffix derive the deterministic
// value companions. Absent takes exactly the input key; Present takes
// exactly the input key and the old int value.
const (
	MapBatchAbsentSuffix  = "$BatchAbsent"
	MapBatchPresentSuffix = "$BatchPresent"
)

// Frozen batch reduction contracts (implemented under G77-G79 against
// this freeze; the proof above only admits bodies that satisfy them):
//
// Companions: `function <target>$BatchAbsent($w0: <key>): bigint` and
// `function <target>$BatchPresent($w0: <key>, $w1: bigint): bigint` —
// synchronous pure bigint lowers of the validated insert/replace value
// expressions with mapping-only marks, no $canOrigin or origin object on
// success. Failures escape through a cold catch throwing the private
// failure value from caught at the first (get) call-site source/span
// with invocation [region]; the fold boundary carries it onward.
//
// Descriptor: separate optional ninth ownCallable argument
// `{absent, present, keyKind, origin, factory}` with exactly those five
// own keys. Absent/present must be functions with own-data lengths 1/2;
// keyKind exactly "int", "bool" or "str"; origin the frozen first
// call-site origin object; factory the exact canonical get-method
// function of the proven receiver. Proxy checks precede own-data reads;
// malformed metadata throws without running; unknown factory or any
// capture declines registration and keeps leaf/generic behavior.
//
// Queries: compiler-private `mapBatchCapability(fn)` returns the exact
// factory-owned runner or undefined (private WeakMap keyed by the exact
// get/insert/replace method objects, no foreign inspection);
// `mapBatchWorker(callable)` returns the registered `{absent, present,
// keyKind, batch, origin}` or undefined from its own WeakMap, never
// consulting other admissions.
//
// Traversal: array fold only, checked before the leaf path, with context
// and owner strictly undefined, over real non-proxy ordinary frozen
// dense arrays of the matching primitive kind. The factory runner
// validates the genuine initial map and all-bigint values, clones its
// private Map once, applies native has/get/set with the companions in
// order, and publishes through the existing own() boundary once; empty
// input returns success(initial) after validation without cloning.
// Invalid map state declines before the first visit with no effect; the
// first transition throw stops and is retained with the first-boundary
// origin. No per-word token, Completion, Promise, snapshot or origin
// object on the admitted successful path.

// checkedMapBatches proves every program function whose checked region
// is a closed batch transition: exactly (map<K,int>, K) inputs with K
// int/bool/str, the identical map result, no errors or escapes, no body
// statements and a terminal matching the complete get→absent-insert /
// present-replace tree. The get receives the original map and input key
// bindings; its absent arm inserts with a pure int expression and its
// present arm binds the old int and replaces with a pure int expression,
// each returning precisely the new map binding. Only the matching
// canonical domain arm may return the original map, and only under the
// condition the guarded private Map state makes impossible. Anything
// else — standard/forward arms, extra arms, alternate success payloads,
// literal or rebound map/key arguments, division, fields, ordinary
// matches, authored/recursive/indirect calls, native expressions,
// fixtures, array/asset/SQL/FormAction/JSONFetch/Action calls,
// coordination, resources, browser or mismatched proof — never
// qualifies. Proof depends only on checked IR, never on workload names
// or emitted text.
func (assembly *programAssembly) checkedMapBatches() map[string]*MapBatchProof {
	proof := make(map[string]*MapBatchProof)
	for _, fn := range assembly.program.Functions {
		entry, ok := mapBatchEntry(assembly, fn)
		if !ok {
			continue
		}
		proof[entry.Identity] = entry
	}
	return proof
}

func mapBatchEntry(assembly *programAssembly, fn *check.ProgramFunction) (*MapBatchProof, bool) {
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
	mapInput, keyInput := region.Inputs[0], region.Inputs[1]
	if mapInput.Identity == "" || mapInput.Type == nil || mapInput.Type.Identity() != mapID {
		return nil, false
	}
	if keyInput.Identity == "" || keyInput.Type == nil || keyInput.Type.Declaration() != keyKind {
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
	receiver, ok := assembly.collectionNames[mapID]
	if !ok || receiver == "" {
		return nil, false
	}
	entry := &MapBatchProof{
		Identity: identity,
		Target:   target,
		Absent:   target + MapBatchAbsentSuffix,
		Present:  target + MapBatchPresentSuffix,
		Source:   fn.Symbol.Source.ID,
		Region:   region,
		KeyKind:  keyKind,
		MapID:    mapID,
		Receiver: receiver,
		Calls:    map[string]string{},
	}
	if !mapBatchTree(assembly, entry, region, mapInput, keyInput) {
		return nil, false
	}
	if len(entry.Calls) != 3 || entry.AbsentValue == nil || entry.PresentValue == nil || entry.Previous == "" {
		return nil, false
	}
	return entry, true
}

// mapBatchTree admits the complete get→absent-insert / present-replace
// terminal: a two-arm completion match over the proven get whose absent
// arm nests the insert transition and whose present arm binds the old
// int and nests the replace transition. Arm order is irrelevant; the arm
// set is exact.
func mapBatchTree(assembly *programAssembly, entry *MapBatchProof, region *ir.Region, mapInput, keyInput ir.Local) bool {
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
	scope := map[string]*ir.Local{
		mapInput.Identity: {Identity: mapInput.Identity, Type: mapInput.Type},
		keyInput.Identity: {Identity: keyInput.Identity, Type: keyInput.Type},
	}
	get, ok := mapBatchCall(assembly, entry, match.Call, "get", mapInput, keyInput, scope)
	if !ok || len(get.Errors) != 1 || get.Errors[0] == nil {
		return false
	}
	entry.GetSpan = get.Span
	var absent, present *ir.Arm
	for i := range match.Arms {
		arm := &match.Arms[i]
		if arm.Forward || arm.Value != nil || arm.Body == nil || len(arm.Patterns) != 0 {
			return false
		}
		switch arm.Outcome {
		case "domain":
			if absent != nil || arm.Error == nil || arm.Error.Identity() != get.Errors[0].Identity() {
				return false
			}
			if arm.Binding != nil && (arm.Binding.Identity == "" || arm.Binding.Type == nil) {
				return false
			}
			absent = arm
		case "ok":
			if present != nil || arm.Error != nil || arm.Binding == nil {
				return false
			}
			if arm.Binding.Identity == "" || arm.Binding.Type == nil || get.Result == nil || arm.Binding.Type.Identity() != get.Result.Identity() {
				return false
			}
			present = arm
		default:
			return false
		}
	}
	if absent == nil || present == nil {
		return false
	}
	entry.Previous = present.Binding.Identity
	extended := make(map[string]*ir.Local, len(scope)+1)
	for key, local := range scope {
		extended[key] = local
	}
	extended[present.Binding.Identity] = present.Binding
	absentValue, ok := mapBatchTransition(assembly, entry, region, absent.Body, "insert", mapInput, keyInput, scope)
	if !ok {
		return false
	}
	presentValue, ok := mapBatchTransition(assembly, entry, region, present.Body, "replace", mapInput, keyInput, extended)
	if !ok {
		return false
	}
	entry.AbsentValue, entry.PresentValue = absentValue, presentValue
	return true
}

// mapBatchTransition admits one nested insert/replace transition: a
// two-arm completion match over the proven call whose canonical domain
// arm returns the original map (impossible under the guarded private
// Map state) and whose ok arm binds the new map and returns precisely
// that binding. It returns the validated pure int value expression.
func mapBatchTransition(assembly *programAssembly, entry *MapBatchProof, region *ir.Region, node *ir.Completion, want string, mapInput, keyInput ir.Local, scope map[string]*ir.Local) (*ir.Expression, bool) {
	if node == nil || node.Kind != ir.MatchCompletion || node.RegionID != region.ID || node.SelfTail {
		return nil, false
	}
	if node.Value != nil || node.Call != nil || node.Block != nil || node.Inherit != "" || node.Match == nil {
		return nil, false
	}
	match := node.Match
	if match.Call == nil || match.ValueResult != nil || len(match.Values) != 0 || len(match.Arms) != 2 {
		return nil, false
	}
	step, value, ok := mapBatchValueCall(assembly, entry, match.Call, want, mapInput, keyInput, scope)
	if !ok || len(step.Errors) != 1 || step.Errors[0] == nil {
		return nil, false
	}
	var recovery, accept *ir.Arm
	for i := range match.Arms {
		arm := &match.Arms[i]
		if arm.Forward || arm.Value != nil || arm.Body == nil || len(arm.Patterns) != 0 {
			return nil, false
		}
		switch arm.Outcome {
		case "domain":
			if recovery != nil || arm.Error == nil || arm.Error.Identity() != step.Errors[0].Identity() {
				return nil, false
			}
			if arm.Binding != nil && (arm.Binding.Identity == "" || arm.Binding.Type == nil) {
				return nil, false
			}
			if !mapBatchReturn(entry, arm.Body, region, mapInput.Identity) {
				return nil, false
			}
			recovery = arm
		case "ok":
			if accept != nil || arm.Error != nil || arm.Binding == nil {
				return nil, false
			}
			if arm.Binding.Identity == "" || arm.Binding.Type == nil || arm.Binding.Type.Identity() != entry.MapID {
				return nil, false
			}
			if !mapBatchReturn(entry, arm.Body, region, arm.Binding.Identity) {
				return nil, false
			}
			accept = arm
		default:
			return nil, false
		}
	}
	if recovery == nil || accept == nil {
		return nil, false
	}
	return value, true
}

// mapBatchReturn admits a success completion returning precisely the
// named map binding and nothing else.
func mapBatchReturn(entry *MapBatchProof, node *ir.Completion, region *ir.Region, want string) bool {
	if node == nil || node.Kind != ir.SuccessCompletion || node.RegionID != region.ID || node.SelfTail {
		return false
	}
	if node.Value == nil || node.Call != nil || node.Block != nil || node.Match != nil || node.Inherit != "" {
		return false
	}
	value := node.Value
	if value == nil || value.Type == nil || value.Type.Identity() != entry.MapID {
		return false
	}
	if value.Kind != ir.Binding || len(value.Inputs) != 0 || value.Text != want {
		return false
	}
	return mapLeafBare(value, nil, "")
}

// mapBatchCall admits one plain invocation step to the checked canonical
// operation of the batch map type with the original map/key bindings.
// It records the resolved target for emission.
func mapBatchCall(assembly *programAssembly, entry *MapBatchProof, call *ir.Invocation, want string, mapInput, keyInput ir.Local, scope map[string]*ir.Local) (*ir.InvocationStep, bool) {
	step, _, ok := mapBatchStep(assembly, entry, call, want, mapInput, keyInput, scope)
	return step, ok
}

// mapBatchValueCall additionally returns the validated pure int value
// expression for insert/replace calls.
func mapBatchValueCall(assembly *programAssembly, entry *MapBatchProof, call *ir.Invocation, want string, mapInput, keyInput ir.Local, scope map[string]*ir.Local) (*ir.InvocationStep, *ir.Expression, bool) {
	return mapBatchStep(assembly, entry, call, want, mapInput, keyInput, scope)
}

func mapBatchStep(assembly *programAssembly, entry *MapBatchProof, call *ir.Invocation, want string, mapInput, keyInput ir.Local, scope map[string]*ir.Local) (*ir.InvocationStep, *ir.Expression, bool) {
	if call == nil || len(call.Steps) != 1 || call.Result == nil {
		return nil, nil, false
	}
	step := &call.Steps[0]
	if step.Identity == "" || step.Site == "" || step.SuccessBinding == "" || step.Result == nil {
		return nil, nil, false
	}
	if step.Callee != nil || step.Native != nil || step.Array != nil || step.Asset != nil || step.Fixtures != nil || step.SQL != nil || step.FormAction != nil || step.JSONFetch != nil || step.Action != nil {
		return nil, nil, false
	}
	if step.Receiver {
		return nil, nil, false
	}
	if call.Result.Identity() != step.Result.Identity() {
		return nil, nil, false
	}
	special := assembly.program.Collections[step.Identity]
	if special == nil || special.Collection == nil || special.Entry == nil || special.Contract == nil {
		return nil, nil, false
	}
	operation, ok := mapLeafOperations[special.Operation]
	if !ok || operation.method != want {
		return nil, nil, false
	}
	if special.Collection.Identity() != entry.MapID {
		return nil, nil, false
	}
	shim := &MapLeafProof{MapID: entry.MapID, KeyKind: entry.KeyKind}
	if !mapLeafContract(shim, operation, special.Contract) {
		return nil, nil, false
	}
	if step.Contract == nil || step.Contract.Identity() != special.Contract.Identity() {
		return nil, nil, false
	}
	if step.Result.Identity() != special.Contract.Result().Identity() {
		return nil, nil, false
	}
	contractErrors := special.Contract.Errors()
	if len(step.Errors) != len(contractErrors) {
		return nil, nil, false
	}
	for i, failure := range step.Errors {
		if failure == nil || contractErrors[i] == nil || failure.Identity() != contractErrors[i].Identity() {
			return nil, nil, false
		}
	}
	receiver, ok := assembly.collectionNames[entry.MapID]
	if !ok || receiver == "" || receiver != entry.Receiver {
		return nil, nil, false
	}
	resolved, ok := assembly.functions[step.Identity]
	if !ok || resolved == "" || resolved != receiver+"."+operation.method {
		return nil, nil, false
	}
	wantArgs := 2
	if operation.value {
		wantArgs = 3
	}
	if len(step.Arguments) != wantArgs {
		return nil, nil, false
	}
	// Like the leaf proof, every call argument passes through checked
	// positional preparation: each prepared local carries the original
	// argument value evaluated once in order. The batch proof is
	// stricter — map/key positions must name the original input
	// bindings, never literals or rebound locals — so the native loop
	// can substitute the fold element for the input key soundly.
	var value *ir.Expression
	validate := func(position int, node *ir.Expression) bool {
		switch position {
		case 0:
			return mapBatchInputBinding(node, mapInput, entry.MapID)
		case 1:
			return mapBatchInputBinding(node, keyInput, keyInput.Type.Identity())
		default:
			if !operation.value || !mapLeafIntExpression(node, scope) {
				return false
			}
			value = node
			return true
		}
	}
	if len(step.Prepare) == 0 {
		for position, argument := range step.Arguments {
			if !validate(position, argument) {
				return nil, nil, false
			}
		}
	} else {
		if len(step.Prepare) != wantArgs {
			return nil, nil, false
		}
		for position, prepared := range step.Prepare {
			if prepared.Value == nil || prepared.Value.Type == nil || prepared.Local.Identity == "" || prepared.Local.Type == nil || prepared.Local.Type.Identity() != prepared.Value.Type.Identity() {
				return nil, nil, false
			}
			if !validate(position, prepared.Value) {
				return nil, nil, false
			}
			argument := step.Arguments[position]
			if argument == nil || argument.Kind != ir.Binding || argument.Text != prepared.Local.Identity || argument.Type == nil || argument.Type.Identity() != prepared.Local.Type.Identity() || len(argument.Inputs) != 0 {
				return nil, nil, false
			}
			if !mapLeafBare(argument, nil, "") {
				return nil, nil, false
			}
		}
	}
	if operation.value && value == nil {
		return nil, nil, false
	}
	if previous, seen := entry.Calls[step.Identity]; seen && previous != resolved {
		return nil, nil, false
	}
	entry.Calls[step.Identity] = resolved
	return step, value, true
}

// mapBatchInputBinding admits only a bare binding to the exact original
// input local. Literals and rebound locals never qualify: the native
// loop substitutes the fold element for the input key.
func mapBatchInputBinding(node *ir.Expression, want ir.Local, wantType string) bool {
	if node == nil || node.Type == nil || node.Type.Identity() != wantType {
		return false
	}
	if node.Kind != ir.Binding || len(node.Inputs) != 0 || node.Text != want.Identity {
		return false
	}
	return mapLeafBare(node, nil, "")
}

// provenMapBatch reports the batch proof for an exact proven
// batch-transition identity/target pair. Stale, rebound or unknown
// pairs never qualify. The returned entry is assembly-owned and must
// not be mutated.
func (e *RegionEmitter) provenMapBatch(identity, target string) (*MapBatchProof, bool) {
	if e.mapBatches == nil || identity == "" || target == "" {
		return nil, false
	}
	entry, ok := e.mapBatches[identity]
	if !ok || entry == nil || entry.Target != target || entry.Absent == "" || entry.Present == "" {
		return nil, false
	}
	return entry, true
}

// batchKeyType maps a proven key kind to its emitted parameter type.
func batchKeyType(keyKind string) (string, error) {
	switch keyKind {
	case "int":
		return "bigint", nil
	case "bool":
		return "boolean", nil
	case "str":
		return "string", nil
	default:
		return "", fmt.Errorf("invalid batch key kind")
	}
}

// BatchCompanions emits the two synchronous bigint value companions for
// a proven batch transition alongside its unchanged async entry, each
// without an export prefix. Each companion lowers one validated pure
// int expression with mapping-only marks: no $canOrigin, origin object
// or per-word metadata on success. Failures escape through a cold
// catch throwing the private failure value from caught at the first
// (get) call-site source/span with invocation [region]; the existing
// fold boundary carries it onward. The emitter keeps no batch state.
func (e *RegionEmitter) BatchCompanions(entry *MapBatchProof) (absent, present string, err error) {
	if entry == nil || entry.Region == nil || entry.AbsentValue == nil || entry.PresentValue == nil || entry.Previous == "" {
		return "", "", fmt.Errorf("invalid map batch proof")
	}
	if !jsBinding.MatchString(entry.Absent) || !jsBinding.MatchString(entry.Present) {
		return "", "", fmt.Errorf("invalid map batch proof")
	}
	region := entry.Region
	if len(region.Inputs) != 2 {
		return "", "", fmt.Errorf("invalid map batch region")
	}
	keyType, err := batchKeyType(entry.KeyKind)
	if err != nil {
		return "", "", err
	}
	absent, err = e.batchValue(entry, entry.Absent, entry.AbsentValue,
		map[string]string{region.Inputs[1].Identity: "$w0"}, []string{"$w0: " + keyType})
	if err != nil {
		return "", "", err
	}
	present, err = e.batchValue(entry, entry.Present, entry.PresentValue,
		map[string]string{region.Inputs[1].Identity: "$w0", entry.Previous: "$w1"},
		[]string{"$w0: " + keyType, "$w1: bigint"})
	if err != nil {
		return "", "", err
	}
	return absent, present, nil
}

func (e *RegionEmitter) batchValue(entry *MapBatchProof, name string, value *ir.Expression, bindings map[string]string, params []string) (string, error) {
	emitter := ExpressionEmitter{Bindings: bindings, Browser: e.Browser}
	if e.SourceID != "" {
		emitter.Mark = func(node *ir.Expression) string { return integerWorkerMark(e.SourceID, node, string(node.Kind)) }
	}
	lowered, err := emitter.Lower(value)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	prefix := ""
	if e.SourceID != "" {
		prefix = mappingMark(e.SourceID, entry.Region.Span, "function")
	}
	source := e.SourceID
	if source == "" {
		source = entry.Region.Source
	}
	cold := fmt.Sprintf("{source:%s,start:%d,end:%d,invocation:[%s]}", quote(source), entry.GetSpan.Start, entry.GetSpan.End, quote(entry.Region.ID))
	fmt.Fprintf(&out, "function %s(%s): bigint {\n%stry {\n%sreturn %s;\n} catch ($canError) { throw $canCaught($canError, %s).value; }\n}\n",
		name, strings.Join(params, ", "), prefix, lowered.Statements, lowered.Value, cold)
	return out.String(), nil
}
