package emit

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

func programImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/collections/array.ts", Names: arrayImports()},
		{Target: runtime + "/assert/context.ts", Names: []ImportName{{"callContext", "$canCallContext"}}},
		{Target: runtime + "/coordination.ts", Names: []ImportName{{"settle", "$canCoordinateSettle"}, {"handle", "$canCoordinateHandle"}, {"aggregate", "$canCoordinateAggregate"}}},
		{Target: runtime + "/owner.ts", TypeOnly: true, Names: []ImportName{{"Participant", "$canParticipant"}}},
		{Target: runtime + "/bytes.ts", Names: []ImportName{{"byteLength", "$canByteLength"}}},
		{Target: runtime + "/callable.ts", Names: []ImportName{{"ownCallable", "$canOwnCallable"}, {"callableInstance", "$canCallableInstance"}}},
		{Target: runtime + "/assert/fixtures.ts", Names: []ImportName{{"withFixture", "$canWithFixture"}}},
		{Target: runtime + "/completion.ts", Names: []ImportName{{"success", "$canSuccess"}, {"failure", "$canFailure"}, {"value", "$canValue"}, {"invoke", "$canInvoke"}, {"caught", "$canCaught"}, {"errorType", "$canErrorType"}, {"errorPayload", "$canErrorPayload"}}},
		{Target: runtime + "/completion.ts", TypeOnly: true, Names: []ImportName{{"Completion", "$canCompletion"}, {"AssertionContext", "$canAssertionContext"}}},
		{Target: runtime + "/data.ts", Names: []ImportName{{"record", "$canRecord"}, {"update", "$canUpdate"}, {"array", "$canArray"}, {"recordIdentity", "$canRecordIdentity"}}},
		{Target: runtime + "/primitive.ts", Names: []ImportName{{"intDivide", "$canIntDivide"}, {"intRemainder", "$canIntRemainder"}, {"intPower", "$canIntPower"}, {"index", "$canIndex"}, {"slice", "$canSlice"}}},
		{Target: runtime + "/failure.ts", Names: []ImportName{{"captureStandard", "$canCaptureStandard"}, {"isStandardFailure", "$canIsStandardFailure"}, {"standardFailureMessage", "$canStandardMessage"}, {"standardFailureKind", "$canFailureKind"}, {"standardFailureMessage", "$canFailureMessage"}, {"standardFailureOccurrenceID", "$canFailureOccurrenceID"}}},
	}
}

// ProgramModules keeps every source module inert. One shared initialization
// function creates error evidence and evaluates the ordered top-level values;
// the entry supervisor invokes it before calling the checked main region.
func ProgramModules(program *check.Program, runtime string, dependencies []ir.Artifact) ([]ir.Artifact, error) {
	return programModules(program, runtime, dependencies, false)
}
func AssertionModules(program *check.Program, runtime string, dependencies []ir.Artifact) ([]ir.Artifact, error) {
	return programModules(program, runtime, dependencies, true)
}
func programModules(program *check.Program, runtime string, dependencies []ir.Artifact, assertions bool) ([]ir.Artifact, error) {
	if program == nil || (!assertions && (program.Entry == nil || program.Entry.Region == nil)) {
		return nil, fmt.Errorf("emission requires a checked entry")
	}
	const statePath = "program/state.ts"
	functions := map[string]string{
		"can.std.bytes@1::from_utf8": "$canBytes.fromUTF8",
		"can.std.bytes@1::to_utf8":   "$canBytes.toUTF8",
		"can.std.bytes@1::from_ints": "$canBytes.fromInts",
		"can.std.bytes@1::to_ints":   "$canBytes.toInts",
		"can.std.bytes@1::empty":     "$canBytes.empty",
		"can.std.io@1::stdout_write": "$canCLI.stdoutWrite",
		"can.std.io@1::stderr_write": "$canCLI.stderrWrite",
	}
	functions["can.intrinsic.str@1::includes"] = "$canText.includes"
	functions["can.intrinsic.str@1::starts_with"] = "$canText.startsWith"
	functions["can.intrinsic.str@1::ends_with"] = "$canText.endsWith"
	functions["can.intrinsic.str@1::to_lower_case"] = "$canText.toLowerCase"
	functions["can.intrinsic.str@1::to_upper_case"] = "$canText.toUpperCase"
	functions["can.intrinsic.str@1::trim"] = "$canText.trim"
	functions["can.intrinsic.str@1::slice"] = "$canText.slice"
	functions["can.intrinsic.str@1::split"] = "$canText.split"
	functions["can.intrinsic.str@1::replace_all"] = "$canText.replaceAll"
	functions["can.std.text@1::join"] = "$canText.join"
	functions["can.std.text@1::scalars"] = "$canText.scalars"
	functions["can.std.text@1::from_scalars"] = "$canText.fromScalars"
	functions["can.std.text@1::graphemes"] = "$canText.graphemes"
	functions["can.std.text@1::normalize_nfc"] = "$canText.normalizeNFC"
	functions["can.std.number@1::divmod"] = "$canAmounts.divmod"
	functions["can.std.number@1::euclidean_divmod"] = "$canAmounts.euclideanDivmod"
	functions["can.std.number@1::round_ratio_half_even"] = "$canAmounts.roundRatioHalfEven"
	functions["can.std.text@1::from_int"] = "$canNumbers.fromInt"
	functions["can.std.text@1::from_float"] = "$canNumbers.fromFloat"
	functions["can.std.text@1::from_bool"] = "$canNumbers.fromBool"
	functions["can.std.text@1::to_int"] = "$canNumbers.toInt"
	functions["can.std.text@1::to_float"] = "$canNumbers.toFloat"
	functions["can.std.text@1::to_bool"] = "$canNumbers.toBool"
	functions["can.std.number@1::int_to_float"] = "$canNumbers.intToFloat"
	functions["can.std.number@1::float_to_int"] = "$canNumbers.floatToInt"
	functions["can.std.number@1::bool_to_int"] = "$canNumbers.boolToInt"
	functions["can.std.number@1::int_to_bool"] = "$canNumbers.intToBool"
	functions["can.std.number@1::floor"] = "$canNumbers.floor"
	functions["can.std.number@1::ceil"] = "$canNumbers.ceil"
	functions["can.std.number@1::trunc"] = "$canNumbers.trunc"
	functions["can.std.number@1::round"] = "$canNumbers.round"
	functions["can.std.number@1::is_finite"] = "$canNumbers.isFinite"
	functions["can.std.number@1::is_nan"] = "$canNumbers.isNaN"
	codecIDs := make([]string, 0, len(program.Codecs))
	for id := range program.Codecs {
		codecIDs = append(codecIDs, id)
	}
	sort.Strings(codecIDs)
	codecNames := map[string]string{}
	for i, id := range codecIDs {
		name := fmt.Sprintf("$canCodec%d", i)
		codecNames[id] = name
		method := "encode"
		if program.Codecs[id].Operation == "can.std.codec@1::decode_json" {
			method = "decode"
		}
		functions[id] = name + "." + method
	}

	collectionNames := map[string]string{}
	collectionTypes := map[string]*check.CollectionSpecialization{}
	collectionIDs := []string{}
	for _, special := range program.Collections {
		id := special.Collection.Identity()
		if collectionTypes[id] == nil {
			collectionIDs = append(collectionIDs, id)
			collectionTypes[id] = special
		}
	}
	sort.Strings(collectionIDs)
	for i, id := range collectionIDs {
		collectionNames[id] = fmt.Sprintf("$canCollection%d", i)
	}
	for id, special := range program.Collections {
		method := strings.Split(special.Operation, "::")[1]
		if method == "empty_map" || method == "empty_set" {
			method = "empty"
		}
		functions[id] = collectionNames[special.Collection.Identity()] + "." + method
	}
	bindings := map[string]string{}
	nativeNames := map[string]string{}
	nativePaths := map[string]string{}
	questions := map[string]*ir.Question{}
	judges := false
	fetches := false
	for i, native := range program.Natives {
		if native.Question == nil && native.Judge == nil && native.Fetch == nil && native.ArmDescription == nil {
			continue
		}
		name := fmt.Sprintf("$canNative%d", i)
		if native.ArmDescription == nil {
			nativeNames[native.Symbol.ID] = name
			nativePaths[native.Symbol.ID] = native.Symbol.Source.OutputPath
		}
		if native.Question != nil {
			questions[native.Symbol.ID] = native.Question
		}
		if native.Fetch != nil {
			functions[native.Symbol.ID] = name
			fetches = true
		}
		if native.Judge != nil {
			functions[native.Symbol.ID] = name
			judges = true
		}
		for j, region := range native.Regions {
			nativeNames[region.ID] = fmt.Sprintf("%sHandler%d", name, j)
			nativePaths[region.ID] = native.Symbol.Source.OutputPath
		}
	}
	connectionNames := map[string]string{}
	for _, native := range program.Natives {
		if native.Judge != nil || native.Fetch != nil {
			connectionNames[native.Connection] = ""
		}
	}
	connectionIDs := make([]string, 0, len(connectionNames))
	for id := range connectionNames {
		connectionIDs = append(connectionIDs, id)
	}
	sort.Strings(connectionIDs)
	for i, id := range connectionIDs {
		connectionNames[id] = fmt.Sprintf("$canConnection%d", i)
	}
	nativeIDs := make([]string, 0, len(nativePaths))
	for id := range nativePaths {
		nativeIDs = append(nativeIDs, id)
	}
	sort.Strings(nativeIDs)
	for i, fn := range program.Functions {
		functions[fn.Identity()] = fmt.Sprintf("$canFunction%d", i)
	}
	for _, value := range program.Initializers {
		bindings[value.Identity] = "($canValues[" + quote(value.Identity) + "] as " + TypeName(value.Type) + ")"
	}
	var errorTypes []*types.Type
	for _, typ := range program.Model.Types() {
		if typ.Kind() == types.Error {
			errorTypes = append(errorTypes, typ)
		}
	}
	bound, err := program.Registry.Bound(errorTypes)
	if err != nil {
		return nil, err
	}
	plan, err := json.Marshal(program.Registry.Plan(bound))
	if err != nil {
		return nil, err
	}
	armHandlers := map[string]string{}
	for _, native := range program.Natives {
		if native.ArmDescription != nil {
			armHandlers[native.Symbol.ID] = nativeNames[native.Regions[0].ID]
		}
	}
	initial, err := initialization(program.Initializers, nil, armHandlers, true)
	if err != nil {
		return nil, err
	}
	declarations, err := NativeTypeDeclarations(program.Model.Types())
	if err != nil {
		return nil, err
	}
	var state strings.Builder
	state.WriteString(declarations)
	for _, id := range connectionIDs {
		policy := program.Connections[id]
		headers := make([]map[string]string, 0, len(policy.Headers))
		for _, header := range policy.Headers {
			headers = append(headers, map[string]string{"name": header.Name, "value": header.Value})
		}
		connection := map[string]any{"endpoint": policy.Endpoint, "timeoutMilliseconds": policy.TimeoutMilliseconds, "maxBodyBytes": policy.MaxBodyBytes, "headers": headers}
		if policy.BearerEnvironment != "" {
			connection["bearerEnvironment"] = policy.BearerEnvironment
		}
		encoded, e := json.Marshal(connection)
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(&state, "export const %s = Object.freeze(%s);\n", connectionNames[id], encoded)
	}

	if fetches {
		state.WriteString("export let $canFetch: ReturnType<typeof $canCreateNamedFetch>;\n")
	}
	if judges {
		state.WriteString("export let $canAI: ReturnType<typeof $canCreateTypeSafe>;\n")
	}
	for _, id := range codecIDs {
		fmt.Fprintf(&state, "export let %s: ReturnType<typeof $canCreateCodec<%s>>;\n", codecNames[id], TypeName(program.Codecs[id].Data))
	}
	for _, id := range collectionIDs {
		special := collectionTypes[id]
		args := special.Collection.Arguments()
		factory := "$canCreateSet<" + TypeName(args[0]) + ">"
		if special.Entry != nil {
			factory = "$canCreateMap<" + TypeName(args[0]) + "," + TypeName(args[1]) + ">"
		}
		fmt.Fprintf(&state, "export let %s: ReturnType<typeof %s>;\n", collectionNames[id], factory)
	}
	state.WriteString("export let $canText: ReturnType<typeof $canCreateText>;\nexport let $canAmounts: ReturnType<typeof $canCreateExactAmounts>;\nexport let $canNumbers: ReturnType<typeof $canCreateNumbers>;\nexport let $canBytes: ReturnType<typeof $canCreateBytes>;\nexport let $canCLI: ReturnType<typeof $canCreateCLI>;\nexport let $canDomain: ReturnType<typeof $canCreateDomain>;\nexport const $canValues: Record<string, unknown> = Object.create(null);\nexport function $canInitialize(): void {\n")
	var invalidData, writeFailed, bytesID string
	for _, typ := range program.Model.Types() {
		switch typ.Declaration() {
		case "can.std.codec@1::invalid_data":
			invalidData = typ.Identity()
		case "can.std.io@1::write_failed":
			writeFailed = typ.Identity()
		case "can.std.bytes@1::buffer":
			bytesID = typ.Identity()
		}
	}
	fmt.Fprintf(&state, "$canDomain = $canCreateDomain(%s, (identity, value) => (identity === %s && $canIsBytes(value)) || $canIsMap(identity,value) || $canIsSet(identity,value));\n", plan, quote(bytesID))
	fmt.Fprintf(&state, "$canBytes = $canCreateBytes($canDomain, %s);\n$canCLI = $canCreateCLI($canDomain, {writeFailed: %s});\n", quote(invalidData), quote(writeFailed))
	numberIDs := map[string]string{}
	for _, typ := range program.Model.Types() {
		numberIDs[typ.Declaration()] = typ.Identity()
	}
	fmt.Fprintf(&state, "$canNumbers = $canCreateNumbers($canDomain, {inexact:%s,invalidNumber:%s,invalidTextBool:%s,invalidIntBool:%s});\n", quote(numberIDs["can.std.number@1::inexact"]), quote(numberIDs["can.std.text@1::invalid_number"]), quote(numberIDs["can.std.text@1::invalid_bool"]), quote(numberIDs["can.std.number@1::invalid_bool"]))
	fmt.Fprintf(&state, "$canAmounts = $canCreateExactAmounts($canDomain, {zeroDivisor:%s,division:%s,rounded:%s});\n", quote(numberIDs["can.std.number@1::zero_divisor"]), quote(numberIDs["can.std.number@1::division"]), quote(numberIDs["can.std.number@1::rounded"]))
	fmt.Fprintf(&state, "$canText = $canCreateText($canDomain, {emptySeparator:%s,emptyPattern:%s,invalidUnicode:%s});\n", quote(numberIDs["can.std.text@1::empty_separator"]), quote(numberIDs["can.std.text@1::empty_pattern"]), quote(numberIDs["can.std.text@1::invalid_unicode"]))
	for _, id := range collectionIDs {
		special := collectionTypes[id]
		args := special.Collection.Arguments()
		if special.Entry != nil {
			fmt.Fprintf(&state, "%s = $canCreateMap<%s,%s>($canDomain,{map:%s,entry:%s,absent:%s,exists:%s},%s);\n", collectionNames[id], TypeName(args[0]), TypeName(args[1]), quote(id), quote(special.Entry.Identity()), quote(numberIDs["can.std.collections@1::key_absent"]), quote(numberIDs["can.std.collections@1::key_exists"]), quote(args[0].Declaration()))
		} else {
			fmt.Fprintf(&state, "%s = $canCreateSet<%s>(%s,%s);\n", collectionNames[id], TypeName(args[0]), quote(id), quote(args[0].Declaration()))
		}
	}
	if fetches {
		ids := map[string]string{}
		for name, declaration := range map[string]string{"invalid": "can.std.http@1::invalid_request", "credential": "can.std.http@1::credentials_missing", "transport": "can.std.http@1::transport_failed", "timeout": "can.std.http@1::timeout", "limit": "can.std.http@1::body_limit", "status": "can.std.http@1::status_error", "header": "can.std.http@1::header", "invalidData": "can.std.codec@1::invalid_data"} {
			ids[name] = numberIDs[declaration]
		}
		encoded, e := json.Marshal(ids)
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(&state, "$canFetch=$canCreateNamedFetch($canDomain,%s,$canOriginalEnvironment);\n", encoded)
	}
	if judges {
		ids := map[string]string{}
		for _, typ := range program.Model.Types() {
			ids[typ.Declaration()] = typ.Identity()
		}
		fields := map[string]string{}
		for name, declaration := range map[string]string{"invalid": "can.std.http@1::invalid_request", "credential": "can.std.http@1::credentials_missing", "transport": "can.std.http@1::transport_failed", "timeout": "can.std.http@1::timeout", "limit": "can.std.http@1::body_limit", "status": "can.std.http@1::status_error", "header": "can.std.http@1::header", "invalidData": "can.std.codec@1::invalid_data", "invalidQuestion": "can.std.ai@1::invalid_question", "invalidAnswer": "can.std.ai@1::invalid_answer"} {
			fields[name] = ids[declaration]
		}
		encoded, e := json.Marshal(fields)
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(&state, "$canAI = $canCreateTypeSafe($canDomain,%s,$canOriginalEnvironment);\n", encoded)
	}
	for _, id := range codecIDs {
		encoded, e := json.Marshal(program.Codecs[id].Schema)
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(&state, "%s = $canCreateCodec<%s>(%s, $canDomain, %s);\n", codecNames[id], TypeName(program.Codecs[id].Data), encoded, quote(invalidData))
	}
	state.WriteString(initial.Code)
	for _, value := range program.Initializers {
		fmt.Fprintf(&state, "$canValues[%s] = %s;\n", quote(value.Identity), initial.Bindings[value.Identity])
	}
	state.WriteString("Object.freeze($canValues);\n}\n")
	imports := append(programImports(runtime), ModuleImport{Target: runtime + "/domain.ts", Names: []ImportName{{"createDomainRuntime", "$canCreateDomain"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/platform/cli.ts", Names: []ImportName{{"createCLI", "$canCreateCLI"}}}, ModuleImport{Target: runtime + "/bytes.ts", Names: []ImportName{{"isBytes", "$canIsBytes"}, {"createBytes", "$canCreateBytes"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/collections/map.ts", Names: []ImportName{{"createMap", "$canCreateMap"}, {"isMap", "$canIsMap"}}}, ModuleImport{Target: runtime + "/collections/set.ts", Names: []ImportName{{"createSet", "$canCreateSet"}, {"isSet", "$canIsSet"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/text.ts", Names: []ImportName{{"createText", "$canCreateText"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/number.ts", Names: []ImportName{{"createNumbers", "$canCreateNumbers"}, {"createExactAmounts", "$canCreateExactAmounts"}}})
	imports = append(imports, ModuleImport{Target: runtime + "/codec/json.ts", Names: []ImportName{{"createCodec", "$canCreateCodec"}}})
	if judges {
		imports = append(imports, ModuleImport{Target: runtime + "/ai/typesafe.ts", Names: []ImportName{{"createTypeSafe", "$canCreateTypeSafe"}}})
	}
	if fetches {
		imports = append(imports, ModuleImport{Target: runtime + "/transport/named.ts", Names: []ImportName{{"createNamedFetch", "$canCreateNamedFetch"}}})
	}
	if judges || fetches {
		imports = append(imports, ModuleImport{Target: runtime + "/environment.ts", Names: []ImportName{{"originalEnvironment", "$canOriginalEnvironment"}}})
	}

	for _, native := range program.Natives {
		if native.ArmDescription != nil {
			name := armHandlers[native.Symbol.ID]
			imports = append(imports, ModuleImport{Target: native.Symbol.Source.OutputPath, Names: []ImportName{{name, name}}})
		}
	}
	modules := []Module{{Path: statePath, Imports: imports, Body: state.String()}}
	byPath := map[string][]*check.ProgramFunction{}
	for _, fn := range program.Functions {
		path := fn.Symbol.Source.OutputPath
		byPath[path] = append(byPath[path], fn)
	}
	for _, path := range nativePaths {
		if _, ok := byPath[path]; !ok {
			byPath[path] = nil
		}
	}
	var paths []string
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		var body strings.Builder
		var regions []*ir.Region
		for _, fn := range byPath[path] {
			regions = append(regions, fn.Region)
		}
		for _, native := range program.Natives {
			if native.Symbol.Source.OutputPath == path && (native.Question != nil || native.Judge != nil || native.Fetch != nil || native.ArmDescription != nil) {
				regions = append(regions, native.Regions...)
			}
		}
		// Initializer references may name types absent from function signatures.
		allTypes := append(program.Model.Types(), regionTypes(regions...)...)
		localTypes, err := NativeTypeDeclarations(allTypes)
		if err != nil {
			return nil, err
		}
		body.WriteString(localTypes)
		for _, fn := range byPath[path] {
			emitter := RegionEmitter{Bindings: bindings, Functions: functions, DomainRuntime: "$canDomain", SourceID: fn.Symbol.Source.ID}
			code, err := emitter.Function(functions[fn.Identity()], fn.Region)
			if err != nil {
				return nil, err
			}
			body.WriteString("export ")
			body.WriteString(code)
		}
		for _, native := range program.Natives {
			if native.Symbol.Source.OutputPath != path || native.Question == nil && native.Judge == nil && native.Fetch == nil && native.ArmDescription == nil {
				continue
			}
			for _, region := range native.Regions {
				emitter := RegionEmitter{Bindings: bindings, Functions: functions, DomainRuntime: "$canDomain", SourceID: native.Symbol.Source.ID}
				code, e := emitter.Function(nativeNames[region.ID], region)
				if e != nil {
					return nil, e
				}
				body.WriteString("export " + code)
			}
			emitter := RegionEmitter{Bindings: bindings, Functions: functions, DomainRuntime: "$canDomain", SourceID: native.Symbol.Source.ID}
			var code string
			var e error
			if native.Question != nil {
				code, e = emitter.QuestionPreparation(nativeNames[native.Symbol.ID], native.Question, nativeNames)
			} else if native.ArmDescription != nil {
				continue
			} else if native.Fetch != nil {
				code, e = emitter.Fetch(nativeNames[native.Symbol.ID], native.Fetch, connectionNames[native.Connection])
			} else {
				policy := program.Connections[native.Connection]
				connectionName := connectionNames[native.Connection]
				code, e = emitter.Judge(nativeNames[native.Symbol.ID], native.Judge, questions, nativeNames, connectionName, policy.Model)
			}
			if e != nil {
				return nil, e
			}
			body.WriteString("export " + code)
		}
		imports := append(programImports(runtime), ModuleImport{Target: statePath, Names: []ImportName{{"$canText", "$canText"}, {"$canAmounts", "$canAmounts"}, {"$canNumbers", "$canNumbers"}, {"$canDomain", "$canDomain"}, {"$canValues", "$canValues"}, {"$canCLI", "$canCLI"}, {"$canBytes", "$canBytes"}}})
		imports = append(imports, ModuleImport{Target: runtime + "/ai/questions.ts", TypeOnly: true, Names: []ImportName{{"PreparedQuestion", "$canPreparedQuestion"}, {"Answer", "$canAnswer"}}})
		if fetches {
			imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{"$canFetch", "$canFetch"}}})
		}
		if judges {
			imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{"$canAI", "$canAI"}}})
		}
		for _, id := range connectionIDs {
			name := connectionNames[id]
			imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{name, name}}})
		}
		for _, id := range nativeIDs {
			target := nativePaths[id]
			if target != path {
				imports = append(imports, ModuleImport{Target: target, Names: []ImportName{{nativeNames[id], nativeNames[id]}}})
			}
		}
		for _, id := range collectionIDs {
			imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{collectionNames[id], collectionNames[id]}}})
		}
		for _, id := range codecIDs {
			imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{codecNames[id], codecNames[id]}}})
		}
		for _, fn := range program.Functions {
			target := fn.Symbol.Source.OutputPath
			if target != path {
				imports = append(imports, ModuleImport{Target: target, Names: []ImportName{{functions[fn.Identity()], functions[fn.Identity()]}}})
			}
		}
		modules = append(modules, Module{Path: path, Imports: imports, Body: body.String()})
	}
	if assertions {
		if len(program.Assertions) == 0 {
			return nil, fmt.Errorf("project has no concrete assertions")
		}
		entry := Module{Path: "entry.ts", Imports: []ModuleImport{
			{Target: runtime + "/assert/runner.ts", Names: []ImportName{{"runAssertions", "$canRunAssertions"}}},
			{Target: statePath, Names: []ImportName{{"$canInitialize", "$canInitialize"}}},
			{Target: runtime + "/diagnostics.ts", Names: []ImportName{{"configureDiagnostics", "$canConfigureDiagnostics"}}},
		}}
		var cases []string
		for i, test := range program.Assertions {
			rootJSON, err := json.Marshal(test.Root)
			if err != nil {
				return nil, err
			}
			digest := sha256.Sum256(append([]byte("can-assertion-root-v1\x00"), rootJSON...))
			path := fmt.Sprintf("assertions/%x.ts", digest)
			imports := append(programImports(runtime), ModuleImport{Target: statePath, Names: []ImportName{{"$canText", "$canText"}, {"$canAmounts", "$canAmounts"}, {"$canNumbers", "$canNumbers"}, {"$canDomain", "$canDomain"}, {"$canValues", "$canValues"}, {"$canCLI", "$canCLI"}, {"$canBytes", "$canBytes"}}})
			for _, id := range collectionIDs {
				imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{collectionNames[id], collectionNames[id]}}})
			}
			for _, id := range codecIDs {
				imports = append(imports, ModuleImport{Target: statePath, Names: []ImportName{{codecNames[id], codecNames[id]}}})
			}
			for _, fn := range program.Functions {
				imports = append(imports, ModuleImport{Target: fn.Symbol.Source.OutputPath, Names: []ImportName{{functions[fn.Identity()], functions[fn.Identity()]}}})
			}
			for _, id := range nativeIDs {
				target := nativePaths[id]
				if name := functions[id]; name != "" {
					imports = append(imports, ModuleImport{Target: target, Names: []ImportName{{name, name}}})
				}
			}
			graph := append(program.Model.Types(), regionTypes(test.Actual, test.Expected)...)
			body, err := NativeTypeDeclarations(graph)
			if err != nil {
				return nil, err
			}
			sourceID := ""
			for src := range program.World.Files {
				if src.Syntax.Source.Name() == test.Actual.Source {
					sourceID = src.ID
				}
			}
			if sourceID == "" {
				return nil, fmt.Errorf("assertion source is not indexed")
			}
			emitter := RegionEmitter{Bindings: bindings, Functions: functions, DomainRuntime: "$canDomain", SourceID: sourceID}
			actual, err := emitter.Function("$canActual", test.Actual)
			if err != nil {
				return nil, err
			}
			expected, err := emitter.Function("$canExpected", test.Expected)
			if err != nil {
				return nil, err
			}
			body += actual + expected + fmt.Sprintf("export const $canCase = Object.freeze({root: Object.freeze(%s), actual: $canActual, expected: $canExpected});\n", rootJSON)
			modules = append(modules, Module{Path: path, Imports: imports, Body: body})
			name := fmt.Sprintf("$canCase%d", i)
			cases = append(cases, name)
			entry.Imports = append(entry.Imports, ModuleImport{Target: path, Names: []ImportName{{"$canCase", name}}})
		}
		entry.Body = "process.exitCode = await $canRunAssertions([" + strings.Join(cases, ",") + "], () => {$canConfigureDiagnostics(import.meta.url); $canInitialize();});\n"
		modules = append(modules, entry)
		return Modules(modules, dependencies...)
	}
	main := program.Entry
	modules = append(modules, Module{Path: "entry.ts", Imports: []ModuleImport{
		{Target: runtime + "/entry.ts", Names: []ImportName{{"runEntry", "$canRunEntry"}}},
		{Target: statePath, Names: []ImportName{{"$canInitialize", "$canInitialize"}}},
		{Target: runtime + "/diagnostics.ts", Names: []ImportName{{"configureDiagnostics", "$canConfigureDiagnostics"}}},
		{Target: main.Symbol.Source.OutputPath, Names: []ImportName{{functions[main.Identity()], "$canMain"}}},
	}, Body: "process.exitCode = await $canRunEntry(() => {$canConfigureDiagnostics(import.meta.url); $canInitialize();}, $canMain, process.argv.slice(2));\n"})
	return Modules(modules, dependencies...)
}
